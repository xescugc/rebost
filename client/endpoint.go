package client

import (
	"net/http"
	"net/url"

	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
)

func makeCreatFileEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/files"
	return kithttp.NewClient(
		http.MethodPut,
		&u,
		encodeCreateFileRequest,
		decodeCreateFileResponse,
	).Endpoint()
}

func makeGetFileEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/files"
	return kithttp.NewClient(
		http.MethodGet,
		&u,
		encodeGetFileRequest,
		decodeGetFileResponse,
		kithttp.BufferedStream(true),
	).Endpoint()
}

func makeDeleteFileEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/files"
	return kithttp.NewClient(
		http.MethodDelete,
		&u,
		encodeDeleteFileRequest,
		decodeDeleteFileResponse,
	).Endpoint()
}

func makeHasFileEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/files"
	return kithttp.NewClient(
		http.MethodHead,
		&u,
		encodeHasFileRequest,
		decodeHasFileResponse,
	).Endpoint()
}

func makeGetConfigEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/config"
	return kithttp.NewClient(
		http.MethodGet,
		&u,
		encodeGetConfigRequest,
		decodeGetConfigResponse,
	).Endpoint()
}

func makeCreatReplicaEndpoint(u url.URL) endpoint.Endpoint {
	u.Path = "/replicas"
	return kithttp.NewClient(
		http.MethodPut,
		&u,
		encodeCreateReplicaRequest,
		decodeCreateReplicaResponse,
	).Endpoint()
}
