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

			fs := afero.NewOsFs()

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
				suow := boltdb.NewUOW(bdb)

				v, err := volume.New(vp, files, idxkeys, fs, suow)
				if err != nil {
					return fmt.Errorf("error creating Volume: %s", err)
				}

				vs = append(vs, v)
			}

			s := storing.New(vs)

			mux := http.NewServeMux()

			mux.Handle("/", storing.MakeHandler(s))

			http.Handle("/", handlers.LoggingHandler(os.Stdout, mux))

			http.ListenAndServe(":"+cfg.Port, nil)

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

	RootCmd.AddCommand(serveCmd)
}
