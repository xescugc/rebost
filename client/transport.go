package client

import (
	"context"
	"io"
	"net/http"
)

func encodeHasFileRequest(_ context.Context, r *http.Request, request interface{}) error {
	hfr := request.(hasFileRequest)
	r.URL.Path += "/" + hfr.Key
	return nil
}

func decodeHasFileResponse(_ context.Context, r *http.Response) (interface{}, error) {
	return hasFileResponse{
		Ok: r.StatusCode == http.StatusNoContent,
	}, nil
}

func encodeGetFileRequest(_ context.Context, r *http.Request, request interface{}) error {
	gfr := request.(getFileRequest)
	r.URL.Path += "/" + gfr.Key
	return nil
}

func decodeGetFileResponse(_ context.Context, r *http.Response) (interface{}, error) {
	iorc := r.Body
	pr, pw := io.Pipe()

	go func() {
		defer iorc.Close()
		defer pw.Close()
		io.Copy(pw, iorc)
	}()

	return getFileResponse{IOR: pr}, nil
}
