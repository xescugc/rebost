package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/boltdb/bolt"
	"github.com/gorilla/handlers"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xescugc/rebost/boltdb"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/fs"
	"github.com/xescugc/rebost/membership"
	"github.com/xescugc/rebost/storing"
	"github.com/xescugc/rebost/volume"
)

var (
	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Starts the rebost server",
		Long:  "Starts the rebost server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.New(viper.GetViper())
			if err != nil {
				return err
			}

			if len(cfg.Volumes) == 0 {
				return errors.New("at last one volume is required")
			}

			osfs := afero.NewOsFs()

			vs := make([]volume.Volume, 0, len(cfg.Volumes))
			for _, vp := range cfg.Volumes {
				bdb, err := createDB(vp)
				if err != nil {
					return fmt.Errorf("error creating the BoltDB: %s", err)
				}
				files, err := boltdb.NewFileRepository(bdb)
				if err != nil {
					return fmt.Errorf("error creating File Repository: %s", err)
				}
				idxkeys, err := boltdb.NewIDXKeyRepository(bdb)
				if err != nil {
					return fmt.Errorf("error creating IDXKeys Repository: %s", err)
				}
				suow := fs.UOWWithFs(boltdb.NewUOW(bdb))

				v, err := volume.New(vp, files, idxkeys, osfs, suow)
				if err != nil {
					return fmt.Errorf("error creating Volume: %s", err)
				}

				vs = append(vs, v)
			}

			m, err := membership.New(cfg, vs, "")
			if err != nil {
				return err
			}

			s := storing.New(m)

			mux := http.NewServeMux()

			mux.Handle("/", storing.MakeHandler(s))

			http.Handle("/", handlers.LoggingHandler(os.Stdout, mux))

			http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), nil)

			return nil
		},
	}
)

func createDB(p string) (*bolt.DB, error) {
	db, err := bolt.Open(path.Join(p, "my.db"), 0600, nil)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func init() {
	serveCmd.PersistentFlags().StringSliceP("volumes", "v", []string{}, "Volumes to store the data")
	viper.BindPFlag("volumes", serveCmd.PersistentFlags().Lookup("volumes"))

	serveCmd.PersistentFlags().StringSliceP("remote", "r", []string{}, "The address of the remote node.")
	viper.BindPFlag("remote", serveCmd.PersistentFlags().Lookup("remote"))

	serveCmd.PersistentFlags().StringSlice("memberlist-bind-port", []string{}, "The port is used for both UDP and TCP gossip. It is assumed other nodes are running on this port, but they do not need to.")
	viper.BindPFlag("memberlist-bind-port", serveCmd.PersistentFlags().Lookup("memberlist-bind-port"))

	serveCmd.PersistentFlags().StringSlice("memberlist-name", []string{}, "The name of this node. This must be unique in the cluster.")
	viper.BindPFlag("memberlist-name", serveCmd.PersistentFlags().Lookup("memberlist-name"))

	RootCmd.AddCommand(serveCmd)
}
