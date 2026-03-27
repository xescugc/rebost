package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/storing/model"
	"github.com/xescugc/rebost/volume"
)

// Service is the interface the HTTP transport requires. It is satisfied by
// Service and any compatible implementation.
type Service interface {
	CreateFile(ctx context.Context, key string, reader io.ReadCloser, replica int, ttl time.Duration, ca time.Time) error
	GetFile(ctx context.Context, key string) (io.ReadCloser, int64, error)
	StatFile(ctx context.Context, key string) (*volume.FileStat, error)
	HasFile(ctx context.Context, key string) (string, bool, error)
	DeleteFile(ctx context.Context, key string) error
	UpdateFileReplica(ctx context.Context, key string, volumeIDs []string, replica int) error
	Config(context.Context) (*config.Config, error)
	CreateReplica(ctx context.Context, key string, reader io.ReadCloser, ttl time.Duration, ca time.Time) (vID string, err error)
	GetReplicaInfo(ctx context.Context, key string) ([]string, int, error)
	Ready(ctx context.Context) error
}

// MakeHandler returns an http.Handler that exposes an S3-compatible API backed
// by the Service. Internal inter-node routes (/replicas/, /config) are
// registered first so they are not shadowed by the S3 bucket/key patterns.
// Health routes (/live, /health, /ready) are always reachable regardless of
// readiness state.
func MakeHandler(s Service, cfg *config.Config, ready func() bool) http.Handler {
	top := mux.NewRouter()

	// Health routes — always reachable, bypass readyMiddleware
	top.Handle("/live", liveHandler()).Methods("GET")
	top.Handle("/health", liveHandler()).Methods("GET")
	top.Handle("/ready", readyCheckHandler(s, ready)).Methods("GET")

	// Operational routes — gated by readiness + auth
	r := top.NewRoute().Subrouter()
	r.Use(readyMiddleware(ready))
	r.Use(S3AuthMiddleware(cfg.S3.AccessKey, cfg.S3.SecretKey, cfg.S3.AuthMode))

	// Internal inter-node routes (JSON, always pass auth)
	r.Handle("/replicas/{key:.*}", createReplicaHandler(s)).Methods("PUT")
	r.Handle("/replicas/{key:.*}", updateFileReplicaHandler(s)).Methods("PATCH")
	r.Handle("/replicas/{key:.*}", getReplicaInfoHandler(s)).Methods("GET")
	r.Handle("/config", getConfigHandler(s)).Methods("GET")

	// S3 object routes: /{bucket}/{key}
	r.Handle("/{bucket}/{key:.*}", putObjectHandler(s)).Methods("PUT")
	r.Handle("/{bucket}/{key:.*}", getObjectHandler(s)).Methods("GET")
	r.Handle("/{bucket}/{key:.*}", deleteObjectHandler(s)).Methods("DELETE")
	r.Handle("/{bucket}/{key:.*}", headObjectHandler(s)).Methods("HEAD")
	r.Handle("/{bucket}/{key:.*}", multipartStubHandler()).Methods("POST")

	// S3 bucket routes: /{bucket}
	r.Handle("/{bucket}", listObjectsHandler()).Methods("GET")
	r.Handle("/{bucket}", createBucketHandler()).Methods("PUT")
	r.Handle("/{bucket}", deleteBucketHandler()).Methods("DELETE")

	top.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encodeS3Error(w, "NoSuchKey", "Path not found", http.StatusNotFound)
	})

	return top
}

func readyMiddleware(ready func() bool) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !ready() {
				http.Error(w, "node not ready", http.StatusServiceUnavailable)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func putObjectHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var iorc io.ReadCloser

		if mr, _ := r.MultipartReader(); mr != nil {
			ppr, ppw := io.Pipe()

			go func() {
				for {
					p, err := mr.NextPart()
					if err == io.EOF {
						ppw.Close()
						return
					}
					if err != nil {
						ppw.CloseWithError(err)
						return
					}
					io.Copy(ppw, p)
				}
			}()

			iorc = ppr
		} else {
			iorc = r.Body
		}

		rep, err := strconv.Atoi(r.URL.Query().Get("replica"))
		if err != nil {
			rep = 0
		}

		ttl, err := time.ParseDuration(r.URL.Query().Get("ttl"))
		if err != nil {
			ttl = 0
		}

		ca, err := time.Parse(time.RFC3339, r.URL.Query().Get("created_at"))
		if err != nil {
			ca = time.Time{}
		}

		vars := mux.Vars(r)
		key := vars["bucket"] + "/" + vars["key"]

		err = s.CreateFile(r.Context(), key, iorc, rep, ttl, ca)
		if err != nil {
			if errors.Is(err, volume.ErrClusterFull) {
				encodeS3Error(w, "InsufficientStorage", err.Error(), http.StatusInsufficientStorage)
				return
			}
			encodeS3Error(w, "InternalError", err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func getObjectHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		key := vars["bucket"] + "/" + vars["key"]

		stat, err := s.StatFile(r.Context(), key)
		if err != nil {
			encodeS3Error(w, "NoSuchKey", err.Error(), http.StatusNotFound)
			return
		}
		iorc, _, err := s.GetFile(r.Context(), key)
		if err != nil {
			encodeS3Error(w, "NoSuchKey", err.Error(), http.StatusNotFound)
			return
		}
		defer iorc.Close()
		if stat.Size >= 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(stat.Size, 10))
		}
		if stat.ETag != "" {
			w.Header().Set("ETag", `"`+stat.ETag+`"`)
		}
		if !stat.ModTime.IsZero() {
			w.Header().Set("Last-Modified", stat.ModTime.UTC().Format(http.TimeFormat))
		}
		io.Copy(w, iorc)
	}
}

func deleteObjectHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		key := vars["bucket"] + "/" + vars["key"]

		err := s.DeleteFile(r.Context(), key)
		if err != nil {
			encodeS3Error(w, "InternalError", err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func headObjectHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		key := vars["bucket"] + "/" + vars["key"]

		vid, ok, err := s.HasFile(r.Context(), key)
		if err != nil {
			encodeS3Error(w, "InternalError", err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			w.Header().Add(model.HasFileVolumeIDHeader, vid)
			encodeS3Error(w, "NoSuchKey", "The specified key does not exist.", http.StatusNotFound)
			return
		}
		stat, err := s.StatFile(r.Context(), key)
		if err != nil {
			encodeS3Error(w, "NoSuchKey", err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Add(model.HasFileVolumeIDHeader, vid)
		if stat.Size >= 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(stat.Size, 10))
		}
		if stat.ETag != "" {
			w.Header().Set("ETag", `"`+stat.ETag+`"`)
		}
		if !stat.ModTime.IsZero() {
			w.Header().Set("Last-Modified", stat.ModTime.UTC().Format(http.TimeFormat))
		}
		w.WriteHeader(http.StatusOK)
	}
}

func createBucketHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Bucket-as-prefix: no state needed, bucket creation is a no-op
		w.WriteHeader(http.StatusOK)
	}
}

func deleteBucketHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Bucket-as-prefix: no state to remove
		w.WriteHeader(http.StatusNoContent)
	}
}

func listObjectsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		encodeS3Error(w, "NotImplemented", "List is not supported by this server.", http.StatusNotImplemented)
	}
}

func multipartStubHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		encodeS3Error(w, "NotImplemented", "Multipart upload is not supported by this server.", http.StatusNotImplemented)
	}
}

func createReplicaHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var iorc io.ReadCloser

		if mr, _ := r.MultipartReader(); mr != nil {
			ppr, ppw := io.Pipe()

			go func() {
				for {
					p, err := mr.NextPart()
					if err == io.EOF {
						ppw.Close()
						return
					}
					if err != nil {
						ppw.CloseWithError(err)
						return
					}
					io.Copy(ppw, p)
				}
			}()

			iorc = ppr
		} else {
			iorc = r.Body
		}

		ttl, err := time.ParseDuration(r.URL.Query().Get("ttl"))
		if err != nil {
			ttl = 0
		}

		ca, err := time.Parse(time.RFC3339, r.URL.Query().Get("created_at"))
		if err != nil {
			ca = time.Time{}
		}

		volID, err := s.CreateReplica(r.Context(), mux.Vars(r)["key"], iorc, ttl, ca)
		if err != nil {
			encodeError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		b, _ := json.Marshal(map[string]interface{}{"data": model.CreateReplica{VolumeID: volID}})
		w.Write(b)
	}
}

func updateFileReplicaHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ufr model.UpdateFileReplica
		if err := json.NewDecoder(r.Body).Decode(&ufr); err != nil {
			encodeError(w, err)
			return
		}
		err := s.UpdateFileReplica(r.Context(), mux.Vars(r)["key"], ufr.VolumeIDs, ufr.Replica)
		if err != nil {
			encodeError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func getConfigHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, err := s.Config(r.Context())
		if err != nil {
			encodeError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		b, _ := json.Marshal(map[string]interface{}{"data": model.ConfigToModel(cfg)})
		w.Write(b)
	}
}

func getReplicaInfoHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := mux.Vars(r)["key"]
		vids, replica, err := s.GetReplicaInfo(r.Context(), key)
		if err != nil {
			encodeError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		b, _ := json.Marshal(map[string]interface{}{
			"data": model.ReplicaInfo{VolumeIDs: vids, Replica: replica},
		})
		w.Write(b)
	}
}

func encodeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}
