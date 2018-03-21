package client

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/xescugc/rebost/storing"
)

type client struct {
	createFile endpoint.Endpoint
	getFile    endpoint.Endpoint
	deleteFile endpoint.Endpoint
	hasFile    endpoint.Endpoint
}

// New returns an client to connect to a remote Storing service
func New(host string) (storing.Service, error) {
	c := client{}
	if host == "" {
		return nil, fmt.Errorf("can't initialize the %q with an empty host", "recipeservice")
	}
	if !strings.HasPrefix(host, "http") {
		host = fmt.Sprintf("http://%s", host)
	}
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	//c.createFile = makeCreatFileEndpoint(*u)
	c.getFile = makeGetFileEndpoint(*u)
	//c.deleteFile = makeDeleteFileEndpoint(*u)
	c.hasFile = makeHasFileEndpoint(*u)

	return c, nil
}

func (c client) CreateFile(ctx context.Context, key string, r io.Reader) error {
	return nil
}

type getFileRequest struct {
	Key string
}

type getFileResponse struct {
	IOR io.Reader
	Err error
}

func (c client) GetFile(ctx context.Context, key string) (io.Reader, error) {
	response, err := c.getFile(ctx, getFileRequest{Key: key})
	if err != nil {
		return nil, err
	}

	resp := response.(getFileResponse)

	return resp.IOR, resp.Err
}

type hasFileRequest struct {
	Key string
}

type hasFileResponse struct {
	Ok  bool
	Err error
}

func (c client) HasFile(ctx context.Context, key string) (bool, error) {
	response, err := c.hasFile(ctx, hasFileRequest{Key: key})
	if err != nil {
		return false, err
	}

	resp := response.(hasFileResponse)

	return resp.Ok, resp.Err
}

func (c client) DeleteFile(ctx context.Context, key string) error {
	return nil
}
