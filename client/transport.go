package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

func encodeHasFileRequest(_ context.Context, r *http.Request, request interface{}) error {
	hfr := request.(hasFileRequest)
	r.URL.Path += "/" + hfr.Key
	return nil
}

func decodeHasFileResponse(_ context.Context, r *http.Response) (interface{}, error) {
	// As it's a HEAD request it's not possible to return an error on the body
	// so we directly add this
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
	var response getFileResponse
	// It has a different content type depending if it failed or not, if it fails
	// it returns a JSON and if it does not fail it returns a File/Stream
	if strings.Contains(r.Header["Content-Type"][0], "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
			return nil, err
		}
	} else {
		response.IORC = r.Body
	}
	return response, nil
}

func encodeGetConfigRequest(_ context.Context, r *http.Request, request interface{}) error { return nil }

func decodeGetConfigResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getConfigResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeDeleteFileRequest(_ context.Context, r *http.Request, request interface{}) error {
	dfr := request.(deleteFileRequest)
	r.URL.Path += "/" + dfr.Key
	return nil
}

func decodeDeleteFileResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response deleteFileResponse
	if r.StatusCode == http.StatusNoContent {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeCreateFileRequest(_ context.Context, r *http.Request, request interface{}) error {
	cfr := request.(createFileRequest)
	r.URL.Path += "/" + cfr.Key
	q := r.URL.Query()
	q.Set("replica", strconv.Itoa(cfr.Replica))
	r.URL.RawQuery = q.Encode()
	r.Body = cfr.IORC
	return nil
}

func decodeCreateFileResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response createFileResponse
	if r.StatusCode == http.StatusCreated {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeCreateReplicaRequest(_ context.Context, r *http.Request, request interface{}) error {
	crr := request.(createReplicaRequest)
	r.URL.Path += "/" + crr.Key
	r.Body = crr.IORC
	return nil
}

func decodeCreateReplicaResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response createReplicaResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeUpdateFileReplicaRequest(_ context.Context, r *http.Request, request interface{}) error {
	ufr := request.(updateFileReplicaRequest)
	r.URL.Path += "/" + ufr.Key
	b, err := json.Marshal(ufr.UpdateFileReplica)
	if err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	return nil
}

func decodeUpdateFileReplicaResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response updateFileReplicaResponse
	if r.StatusCode == http.StatusOK {
		return response, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}
