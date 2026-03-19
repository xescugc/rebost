package storing

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/xescugc/rebost/storing/model"
)

// MakeHandler returns a http.Handler that uses the storing.Service
// to make the http calls, it links each http endpoint to a
// storing.Service method
func MakeHandler(s Service) http.Handler {
	r := mux.NewRouter()

	r.Handle("/files/{key:.*}", createFileHandler(s)).Methods("PUT")
	r.Handle("/files/{key:.*}", getFileHandler(s)).Methods("GET")
	r.Handle("/files/{key:.*}", deleteFileHandler(s)).Methods("DELETE")
	r.Handle("/files/{key:.*}", hasFileHandler(s)).Methods("HEAD")

	r.Handle("/replicas/{key:.*}", createReplicaHandler(s)).Methods("PUT")
	r.Handle("/replicas/{key:.*}", updateFileReplicaHandler(s)).Methods("PATCH")

	r.Handle("/config", getConfigHandler(s)).Methods("GET")

	r.NotFoundHandler = http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Context-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error": "Path not found"}`)
		},
	)

	return r
}

func createFileHandler(s Service) http.HandlerFunc {
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
						log.Println(err)
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

		err = s.CreateFile(r.Context(), mux.Vars(r)["key"], iorc, rep, ttl, ca)
		if err != nil {
			encodeError(w, err)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func getFileHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		iorc, err := s.GetFile(r.Context(), mux.Vars(r)["key"])
		if err != nil {
			encodeError(w, err)
			return
		}
		defer iorc.Close()
		io.Copy(w, iorc)
	}
}

func deleteFileHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := s.DeleteFile(r.Context(), mux.Vars(r)["key"])
		if err != nil {
			encodeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func hasFileHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vid, ok, err := s.HasFile(r.Context(), mux.Vars(r)["key"])
		if err != nil {
			encodeError(w, err)
			return
		}
		w.Header().Add(model.HasFileVolumeIDHeader, vid)
		if ok {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
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
						log.Println(err)
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

func encodeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}
