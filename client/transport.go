package client

import (
	"context"
	"encoding/json"
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
	return getFileResponse{IORC: r.Body}, nil
}

func encodeGetConfigRequest(_ context.Context, r *http.Request, request interface{}) error { return nil }

func decodeGetConfigResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getConfigResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}
