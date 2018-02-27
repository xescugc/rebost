package storing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
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

	r := mux.NewRouter()

	r.Handle("/files/{key:.*}", createFileHandler).Methods("PUT")
	r.Handle("/files/{key:.*}", getFileHandler).Methods("GET")
	r.Handle("/files/{key:.*}", deleteFileHandler).Methods("DELETE")

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
	pr, pw := io.Pipe()

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

	go func() {
		defer iorc.Close()
		defer pw.Close()
		io.Copy(pw, iorc)
	}()

	return createFileRequest{
		Key:  mux.Vars(r)["key"],
		Body: pr,
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
	_, err := io.Copy(w, gfr.IOR)
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
