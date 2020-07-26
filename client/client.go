package client

import (
	"context"
	"errors"
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

	createReplica     endpoint.Endpoint
	updateFileReplica endpoint.Endpoint
}

// New returns an client to connect to a remote Storing service
func New(host string) (*Client, error) {
	c := &Client{}
	if host == "" {
		return nil, fmt.Errorf("can't initialize the %q with an empty host", "rebost")
	}
	if !strings.HasPrefix(host, "http") {
		host = fmt.Sprintf("http://%s", host)
	}
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	c.createFile = makeCreatFileEndpoint(*u)
	c.createReplica = makeCreatReplicaEndpoint(*u)
	c.updateFileReplica = makeUpdateFileReplica(*u)
	c.getFile = makeGetFileEndpoint(*u)
	c.deleteFile = makeDeleteFileEndpoint(*u)
	c.hasFile = makeHasFileEndpoint(*u)
	c.getConfig = makeGetConfigEndpoint(*u)

	return c, nil
}

type getConfigResponse struct {
	Data model.Config `json:"data,omitempty"`
	Err  string       `json:"error,omitempty"`
}

// Config returns the config of the Node
func (c Client) Config(ctx context.Context) (*config.Config, error) {
	response, err := c.getConfig(ctx, nil)
	if err != nil {
		return nil, err
	}

	resp := response.(getConfigResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	cfg := config.Config(resp.Data)
	return &cfg, nil
}

type createFileRequest struct {
	Key     string
	IORC    io.ReadCloser
	Replica int
}

type createFileResponse struct {
	Err string `json:"error,omitempty"`
}

// CreateFile creates a file with the  given key and the r content with rep replicas
func (c Client) CreateFile(ctx context.Context, key string, r io.ReadCloser, rep int) error {
	response, err := c.createFile(ctx, createFileRequest{Key: key, IORC: r, Replica: rep})
	if err != nil {
		return err
	}

	resp := response.(createFileResponse)

	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}

type createReplicaRequest struct {
	Key  string
	IORC io.ReadCloser
}

type createReplicaResponse struct {
	Data model.CreateReplica `json:"data,omitempty"`
	Err  string              `json:"error,omitempty"`
}

// CreateReplica creates a new replica to the Node
func (c Client) CreateReplica(ctx context.Context, key string, reader io.ReadCloser) (string, error) {
	response, err := c.createReplica(ctx, createReplicaRequest{Key: key, IORC: reader})
	if err != nil {
		return "", err
	}

	resp := response.(createReplicaResponse)

	if resp.Err != "" {
		return "", errors.New(resp.Err)
	}

	return resp.Data.VolumeID, nil
}

type updateFileReplicaRequest struct {
	Key               string
	UpdateFileReplica model.UpdateFileReplica
}

type updateFileReplicaResponse struct {
	Err string `json:"error,omitempty"`
}

// UpdateFileReplica updtes the file replica information
func (c Client) UpdateFileReplica(ctx context.Context, key string, vids []string, replica int) error {
	response, err := c.updateFileReplica(ctx, updateFileReplicaRequest{
		Key:               key,
		UpdateFileReplica: model.UpdateFileReplica{VolumeIDs: vids, Replica: replica},
	})
	if err != nil {
		return err
	}

	resp := response.(updateFileReplicaResponse)

	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}

type getFileRequest struct {
	Key string
}

type getFileResponse struct {
	IORC io.ReadCloser
	Err  string `json:"error,omitempty"`
}

// GetFile returns the requested file
func (c Client) GetFile(ctx context.Context, key string) (io.ReadCloser, error) {
	response, err := c.getFile(ctx, getFileRequest{Key: key})
	if err != nil {
		return nil, err
	}

	resp := response.(getFileResponse)

	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return resp.IORC, nil
}

type hasFileRequest struct {
	Key string
}

type hasFileResponse struct {
	Ok  bool
	Err string `json:"error,omitempty"`
}

// HasFile returns if the file exists
func (c Client) HasFile(ctx context.Context, key string) (bool, error) {
	response, err := c.hasFile(ctx, hasFileRequest{Key: key})
	if err != nil {
		return false, err
	}

	resp := response.(hasFileResponse)

	if resp.Err != "" {
		return false, errors.New(resp.Err)
	}

	return resp.Ok, nil
}

type deleteFileRequest struct {
	Key string
}

type deleteFileResponse struct {
	Err string `json:"error,omitempty"`
}

// DeleteFile deletes the file with the given key
func (c Client) DeleteFile(ctx context.Context, key string) error {
	response, err := c.deleteFile(ctx, deleteFileRequest{Key: key})
	if err != nil {
		return err
	}

	resp := response.(deleteFileResponse)

	if resp.Err != "" {
		return errors.New(resp.Err)
	}

	return nil
}
