package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"log/slog"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xescugc/rebost/boltdb"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/dashboard"
	"github.com/xescugc/rebost/dashboard/assets"
	dhttp "github.com/xescugc/rebost/dashboard/transport/http"
	"github.com/xescugc/rebost/fs"
	"github.com/xescugc/rebost/membership"
	"github.com/xescugc/rebost/otelwrap"
	"github.com/xescugc/rebost/state"
	"github.com/xescugc/rebost/storing"
	httptransport "github.com/xescugc/rebost/storing/transport/http"
	"github.com/xescugc/rebost/uow"
	"github.com/xescugc/rebost/volume"
	bolt "go.etcd.io/bbolt"
	"go.opentelemetry.io/otel"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Starts the rebost server",
		Long:  "Starts the rebost server",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			cfg, err := config.New(viper.GetViper())
			if err != nil {
				return err
			}
			var level slog.Level
		switch strings.ToLower(cfg.Log.Level) {
		case "debug":
			level = slog.LevelDebug
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		default:
			level = slog.LevelInfo
		}
		opts := &slog.HandlerOptions{AddSource: true, Level: level}
		var handler slog.Handler
		if cfg.Log.Format == "json" {
			handler = slog.NewJSONHandler(os.Stdout, opts)
		} else {
			handler = slog.NewTextHandler(os.Stdout, opts)
		}
		logger := slog.New(handler)

			if cfg.Name == "" {
				return errors.New("the 'name' is required")
			}

			promExporter, err := otelprometheus.New()
			if err != nil {
				return fmt.Errorf("init prometheus exporter: %w", err)
			}
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(promExporter))
			otel.SetMeterProvider(mp)
			defer mp.Shutdown(ctx)

			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			))
			var tpOpts []sdktrace.TracerProviderOption
			tpOpts = append(tpOpts, sdktrace.WithSampler(sdktrace.AlwaysSample()))
			if cfg.Tracing.OTLPEndpoint != "" {
				exp, err := otlptracehttp.New(ctx,
					otlptracehttp.WithEndpoint(cfg.Tracing.OTLPEndpoint),
					otlptracehttp.WithInsecure(),
				)
				if err != nil {
					return fmt.Errorf("init OTLP trace exporter: %w", err)
				}
				tpOpts = append(tpOpts, sdktrace.WithBatcher(exp))
			}
			tp := sdktrace.NewTracerProvider(tpOpts...)
			otel.SetTracerProvider(tp)
			defer tp.Shutdown(ctx)

			osfs := afero.NewOsFs()

			vs := make([]volume.Local, 0, len(cfg.Volumes))
			for _, vp := range cfg.Volumes {
				// We split the vp as it may contain the size of the volume as the second position like : /root:20G
				bdb, err := createDB(strings.Split(vp, ":")[0])
				if err != nil {
					return fmt.Errorf("error creating the BoltDB: %s", err)
				}
				boltdbUow, err := boltdb.NewUOW(bdb)
				if err != nil {
					return fmt.Errorf("error creating BoltDB UoW: %s", err)
				}
				suow := fs.UOWWithFs(osfs, boltdbUow)
				suow, err = otelwrap.UOWWithOTEL(suow)
				if err != nil {
					return fmt.Errorf("error creating OTEL UoW wrapper: %w", err)
				}

				var (
					st *state.State
				)

				err = suow(ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
					st, err = uw.State().Find(ctx)
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					return fmt.Errorf("error getting the state of Volume: %s", err)
				}

				v, err := volume.New(vp, osfs, logger, suow, cfg.Timing.ScrubInterval, cfg.Timing.ReplicaCheckInterval)
				if err != nil {
					return fmt.Errorf("error creating Volume: %s", err)
				}

				if st.IsInDowntimeRange(cfg.Timing.VolumeDowntime) {
					logger.Info(fmt.Sprintf("Resetting the volume due to downtime: %q", vp))
					err = v.Reset(ctx)
					if err != nil {
						return fmt.Errorf("error resetting the Volume due to downtime: %s", err)
					}
				} else {
					if err := v.ReconcileReplicas(ctx); err != nil {
						return fmt.Errorf("error reconciling replicas: %s", err)
					}
				}

				logger.Info(fmt.Sprintf("Attached to volume: %q", vp))
				vs = append(vs, otelwrap.LocalVolumeWithTracing(v))
			}

			if len(vs) == 0 {
				logger.Info("starting in proxy mode (no local volumes)")
			}

			m, err := membership.New(cfg, vs, cfg.Remote, logger)
			if err != nil {
				return err
			}

			s, err := storing.New(cfg, m, logger)
			if err != nil {
				return err
			}
			s = otelwrap.ServiceWithTracing(s)

			m.SetStatus(membership.StatusRunning)

			mux := http.NewServeMux()

			mux.Handle("/metrics", promhttp.Handler())
			mux.Handle("/", httptransport.MakeHandler(s, cfg, m.IsRunning, logger))

			errs := make(chan error)

			svr := &http.Server{
				Addr:    fmt.Sprintf(":%d", cfg.Port),
				Handler: httpRequestLogger(logger, mux),
			}

			go func() {
				c := make(chan os.Signal, 1)
				signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
				errs <- fmt.Errorf("%s", <-c)
			}()

			go func() {
				c := make(chan os.Signal, 1)
				signal.Notify(c, syscall.SIGQUIT)
				<-c
				logger.Info("received SIGQUIT, starting graceful drain")
				if err := s.Drain(ctx); err != nil {
					errs <- fmt.Errorf("drain failed: %s", err)
					return
				}
				errs <- fmt.Errorf("drain complete, shutting down")
			}()

			go func() {
				logger.Info("started storing server", "port", cfg.Port)
				errs <- svr.ListenAndServe()
			}()

			if cfg.Dashboard.Enabled {
				d := dashboard.New(cfg, m, logger)

				dhandler := dhttp.MakeHandler(d, logger)

				dmux := http.NewServeMux()
				dmux.Handle("/", dhandler)
				dmux.Handle("/css/", http.FileServer(http.FS(assets.Assets)))
				dmux.Handle("/js/", http.FileServer(http.FS(assets.Assets)))

				dsvr := &http.Server{
					Addr:    fmt.Sprintf(":%d", cfg.Dashboard.Port),
					Handler: httpRequestLogger(logger, dmux),
				}

				go func() {
					logger.Info("started dashboard server", "port", cfg.Dashboard.Port)
					errs <- dsvr.ListenAndServe()
				}()
			}

			logger.Info("exit", "err", <-errs)

			return nil
		},
	}
)

// httpRequestLogger wraps h and logs every request via logger.
func httpRequestLogger(logger *slog.Logger, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		h.ServeHTTP(rw, r)

		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}

		uri := r.RequestURI
		if r.ProtoMajor == 2 && r.Method == "CONNECT" {
			uri = r.Host
		}
		if uri == "" {
			uri = r.URL.RequestURI()
		}

		logger.Info("request",
			"method", r.Method,
			"uri", uri,
			"host", host,
			"status", rw.status,
			"size", rw.size,
		)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.size += n
	return n, err
}

func createDB(p string) (*bolt.DB, error) {
	db, err := bolt.Open(path.Join(p, "my.db"), 0600, nil)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func init() {
	serveCmd.PersistentFlags().IntP("port", "p", config.DefaultPort, "Destination port")
	viper.BindPFlag("port", serveCmd.PersistentFlags().Lookup("port"))

	serveCmd.PersistentFlags().String("name", "", "The name of this node. This must be unique in the cluster.")
	viper.BindPFlag("name", serveCmd.PersistentFlags().Lookup("name"))

	serveCmd.PersistentFlags().StringSliceP("volumes", "v", []string{}, "Volumes to store the data, to specify a fixed size for the volume do it as so '/:20G'")
	viper.BindPFlag("volumes", serveCmd.PersistentFlags().Lookup("volumes"))

	serveCmd.PersistentFlags().StringP("remote", "r", "", "The URL of a remote Node to join on the cluster")
	viper.BindPFlag("remote", serveCmd.PersistentFlags().Lookup("remote"))

	serveCmd.PersistentFlags().Int("replica", config.DefaultReplica, "The default number of replicas used if none specified on the requests")
	viper.BindPFlag("replica", serveCmd.PersistentFlags().Lookup("replica"))

	serveCmd.PersistentFlags().Duration("timing.volume-downtime", config.DefaultVolumeDowntime, fmt.Sprintf("The time a volume can be down before start replicating and also the time the node can be restarted before it's old and all the data would be cleaned. The value cannot be lower or equal to %s", volume.TickerDuration))
	viper.BindPFlag("timing.volume-downtime", serveCmd.PersistentFlags().Lookup("timing.volume-downtime"))

	serveCmd.PersistentFlags().Duration("timing.scrub-interval", config.DefaultScrubInterval, "How often each volume re-verifies file checksums and auto-repairs corrupt copies from replicas")
	viper.BindPFlag("timing.scrub-interval", serveCmd.PersistentFlags().Lookup("timing.scrub-interval"))

	serveCmd.PersistentFlags().Duration("timing.replica-check-interval", config.DefaultReplicaCheckInterval, "How often each volume checks for under-replicated files and re-queues replication jobs")
	viper.BindPFlag("timing.replica-check-interval", serveCmd.PersistentFlags().Lookup("timing.replica-check-interval"))

	serveCmd.PersistentFlags().Duration("timing.replica-consistency-interval", config.DefaultReplicaConsistencyInterval, "How often non-owner replicas verify they are still expected by the file owner and purge stale local copies")
	viper.BindPFlag("timing.replica-consistency-interval", serveCmd.PersistentFlags().Lookup("timing.replica-consistency-interval"))

	serveCmd.PersistentFlags().Duration("timing.rebalance-interval", config.DefaultRebalanceInterval, "How often Rebost checks whether any locally-held file primaries have been displaced by a higher-ranking HRW volume and transfers ownership. Set to 0 to disable.")
	viper.BindPFlag("timing.rebalance-interval", serveCmd.PersistentFlags().Lookup("timing.rebalance-interval"))

	serveCmd.PersistentFlags().Int("memberlist.port", 0, "The port is used for both UDP and TCP gossip. By default a free port will be used")
	viper.BindPFlag("memberlist.port", serveCmd.PersistentFlags().Lookup("memberlist.port"))

	serveCmd.PersistentFlags().Int("dashboard.port", 3806, "Destination port of the dashboard")
	viper.BindPFlag("dashboard.port", serveCmd.PersistentFlags().Lookup("dashboard.port"))

	serveCmd.PersistentFlags().Bool("dashboard.enabled", true, "Enable or not the Dashboard on this node")
	viper.BindPFlag("dashboard.enabled", serveCmd.PersistentFlags().Lookup("dashboard.enabled"))

	serveCmd.PersistentFlags().Int("cache.size", config.DefaultCacheSize, "Size of the cache used to store reference to object location on other nodes")
	viper.BindPFlag("cache.size", serveCmd.PersistentFlags().Lookup("cache.size"))

	serveCmd.PersistentFlags().String("s3.access_key", "", "S3 access key for AWS Signature V4 authentication (leave empty to disable auth)")
	viper.BindPFlag("s3.access_key", serveCmd.PersistentFlags().Lookup("s3.access_key"))

	serveCmd.PersistentFlags().String("s3.secret_key", "", "S3 secret key for AWS Signature V4 authentication")
	viper.BindPFlag("s3.secret_key", serveCmd.PersistentFlags().Lookup("s3.secret_key"))

	serveCmd.PersistentFlags().String("s3.auth_mode", "", `S3 auth mode: "all" (default) requires auth for all requests; "write" requires auth only for write operations (PUT, DELETE, PATCH, POST)`)
	viper.BindPFlag("s3.auth_mode", serveCmd.PersistentFlags().Lookup("s3.auth_mode"))

	serveCmd.PersistentFlags().StringToString("tag", map[string]string{}, "Arbitrary key=value label for this node (repeatable: --tag rack=us-east-1 --tag env=prod)")
	viper.BindPFlag("tags", serveCmd.PersistentFlags().Lookup("tag"))

	serveCmd.Flags().String("tracing.otlp-endpoint", "", "OTLP HTTP endpoint for trace export (e.g. localhost:4318); empty disables export")
	viper.BindPFlag("tracing.otlp-endpoint", serveCmd.Flags().Lookup("tracing.otlp-endpoint"))

	serveCmd.PersistentFlags().String("log.format", "text", `Log output format: "text" (default) or "json"`)
	viper.BindPFlag("log.format", serveCmd.PersistentFlags().Lookup("log.format"))

	serveCmd.PersistentFlags().String("log.level", "info", `Log level: "debug", "info" (default), "warn", or "error"`)
	viper.BindPFlag("log.level", serveCmd.PersistentFlags().Lookup("log.level"))

	RootCmd.AddCommand(serveCmd)
}
