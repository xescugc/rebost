package client

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/storing/model"
)

// Client is the client structure that fulfills the storing.Service
// interface and it's ment to be used to access to a remote node
type Client struct {
	createFile endpoint.Endpoint
	getFile    endpoint.Endpoint
	deleteFile endpoint.Endpoint
	hasFile    endpoint.Endpoint
	getConfig  endpoint.Endpoint
}

// New returns an client to connect to a remote Storing service
func New(host string) (*Client, error) {
	c := &Client{}
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
	c.getConfig = makeGetConfigEndpoint(*u)

	return c, nil
}

type getConfigResponse struct {
	Data model.Config `json:"data,omitempty"`
	Err  error        `json:"error,omitempty"`
}

// Config returns the config of the Node
func (c Client) Config(ctx context.Context) (*config.Config, error) {
	response, err := c.getConfig(ctx, nil)
	if err != nil {
		return nil, err
	}

	resp := response.(getConfigResponse)
	if resp.Err != nil {
		return nil, resp.Err
	}

	cfg := config.Config(resp.Data)
	return &cfg, nil
}

// CreateFile WIP
func (c Client) CreateFile(ctx context.Context, key string, r io.Reader) error {
	return nil
}

type getFileRequest struct {
	Key string
}

type getFileResponse struct {
	IOR io.Reader
	Err error
}

// GetFile returns the requested file
func (c Client) GetFile(ctx context.Context, key string) (io.Reader, error) {
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

// HasFile returns if the file exists
func (c Client) HasFile(ctx context.Context, key string) (bool, error) {
	response, err := c.hasFile(ctx, hasFileRequest{Key: key})
	if err != nil {
		return false, err
	}

	resp := response.(hasFileResponse)

	return resp.Ok, resp.Err
}

// DeleteFile WIP
func (c Client) DeleteFile(ctx context.Context, key string) error {
	return nil
}
