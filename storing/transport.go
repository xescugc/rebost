package storing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/xescugc/rebost/storing/model"
)

// MakeHandler returns a http.Handler that uses the storing.Service
// to make the http calls, it links eac http endpoint to a
// storing.Service method
func MakeHandler(s Service) http.Handler {
	createFileHandler := kithttp.NewServer(
		makeCreateFileEndpoint(s),
		decodeCreateFileRequest,
		encodeCreateFileResponse,
	)

	getFileHandler := kithttp.NewServer(
		makeGetFileEndpoint(s),
		decodeGetFileRequest,
		encodeGetFileResponse,
	)

	deleteFileHandler := kithttp.NewServer(
		makeDeleteFileEndpoint(s),
		decodeDeleteFileRequest,
		encodeDeleteFileResponse,
	)

	hasFileHandler := kithttp.NewServer(
		makeHasFileEndpoint(s),
		decodeHasFileRequest,
		encodeHasFileResponse,
	)

	createReplicaHandler := kithttp.NewServer(
		makeCreateReplicaEndpoint(s),
		decodeCreateReplicaRequest,
		encodeJSONResponse,
	)

	updateFileReplicaHandler := kithttp.NewServer(
		makeUpdateFileReplicaEndpoint(s),
		decodeUpdateFileReplicaRequest,
		encodeUpdateFileReplicaResponse,
	)

	getConfigHandler := kithttp.NewServer(
		makeGetConfigEndpoint(s),
		decodeGetConfigRequest,
		encodeJSONResponse,
	)

	r := mux.NewRouter()

	r.Handle("/files/{key:.*}", createFileHandler).Methods("PUT")
	r.Handle("/files/{key:.*}", getFileHandler).Methods("GET")
	r.Handle("/files/{key:.*}", deleteFileHandler).Methods("DELETE")
	r.Handle("/files/{key:.*}", hasFileHandler).Methods("HEAD")

	r.Handle("/replicas/{key:.*}", createReplicaHandler).Methods("PUT")
	r.Handle("/replicas/{key:.*}", updateFileReplicaHandler).Methods("PATCH")

	r.Handle("/config", getConfigHandler).Methods("GET")

	r.NotFoundHandler = http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Context-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error": "Path not found"}`)
		},
	)

	return r
}

func decodeCreateFileRequest(_ context.Context, r *http.Request) (interface{}, error) {
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
		// If we can not transform the replica to an Int, we
		// just use the default value of int, which is 0
		rep = 0
	}

	return createFileRequest{
		Key:     mux.Vars(r)["key"],
		Body:    iorc,
		Replica: rep,
	}, nil
}

func encodeCreateFileResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}
	w.WriteHeader(http.StatusCreated)
	return nil
}

func decodeGetFileRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return getFileRequest{
		Key: mux.Vars(r)["key"],
	}, nil
}

func encodeGetFileResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}

	gfr := response.(getFileResponse)
	defer gfr.IORC.Close()
	_, err := io.Copy(w, gfr.IORC)
	return err
}

func decodeDeleteFileRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return deleteFileRequest{
		Key: mux.Vars(r)["key"],
	}, nil
}

func encodeDeleteFileResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func decodeHasFileRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return hasFileRequest{
		Key: mux.Vars(r)["key"],
	}, nil
}

func encodeHasFileResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	hfr := response.(hasFileResponse)
	w.Header().Add(model.HasFileVolumeIDHeader, hfr.VolumeID)
	if hfr.Ok {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
	return nil
}

func decodeGetConfigRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func decodeCreateReplicaRequest(_ context.Context, r *http.Request) (interface{}, error) {
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

	return createReplicaRequest{
		Key:  mux.Vars(r)["key"],
		Body: iorc,
	}, nil
}

func decodeUpdateFileReplicaRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var ufr model.UpdateFileReplica
	err := json.NewDecoder(r.Body).Decode(&ufr)
	if err != nil {
		return nil, err
	}
	return updateFileReplicaRequest{
		Key:       mux.Vars(r)["key"],
		VolumeIDs: ufr.VolumeIDs,
		Replica:   ufr.Replica,
	}, nil
}

func encodeUpdateFileReplicaResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

func encodeJSONResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	b, err := json.Marshal(response)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, string(b))

	return err
}

type errorer interface {
	error() error
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch err.(type) {
	//case errors.NotFound:
	//w.WriteHeader(http.StatusNotFound)
	//case errors.Invalid:
	//w.WriteHeader(http.StatusBadRequest)
	//case errors.AlreadyExists:
	//w.WriteHeader(http.StatusUnprocessableEntity)
	//case errors.Unexpected:
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}
