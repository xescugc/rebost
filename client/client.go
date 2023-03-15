package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"

	"github.com/go-kit/kit/endpoint"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/storing/model"
)

// Client is the client structure that fulfills the storing.Service
// interface and it's ment to be used to access to a remote node
type Client struct {
	clientsLock sync.Mutex
	nextClient  int
	clients     []*client
}

type client struct {
	createFile endpoint.Endpoint
	getFile    endpoint.Endpoint
	deleteFile endpoint.Endpoint
	hasFile    endpoint.Endpoint
	getConfig  endpoint.Endpoint

	createReplica     endpoint.Endpoint
	updateFileReplica endpoint.Endpoint
}

// New returns an client to connect to a remote Storing service
func New(hosts ...string) (*Client, error) {
	cl := &Client{
		clients: make([]*client, len(hosts)),
	}
	for i, h := range hosts {
		c := &client{}
		if h == "" {
			return nil, fmt.Errorf("can't initialize the %q with an empty host", "rebost")
		}
		if !strings.HasPrefix(h, "http") {
			h = fmt.Sprintf("http://%s", h)
		}
		u, err := url.Parse(h)
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

		cl.clients[i] = c
	}

	return cl, nil
}

// getClient returns the next client from the
// list of clients to use
func (cl *Client) getClient() *client {
	cl.clientsLock.Lock()
	defer cl.clientsLock.Unlock()

	rc := cl.clients[cl.nextClient]
	if cl.nextClient == len(cl.clients)-1 {
		cl.nextClient = 0
	} else {
		cl.nextClient++
	}
	return rc
}

type getConfigResponse struct {
	Data model.Config `json:"data,omitempty"`
	Err  string       `json:"error,omitempty"`
}

// Config returns the config of the Node
func (cl *Client) Config(ctx context.Context) (*config.Config, error) {
	c := cl.getClient()
	response, err := c.getConfig(ctx, nil)
	if err != nil {
		return nil, err
	}

	resp := response.(getConfigResponse)
	if resp.Err != "" {
		return nil, errors.New(resp.Err)
	}

	return model.ToConfig(resp.Data), nil
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
func (cl *Client) CreateFile(ctx context.Context, key string, r io.ReadCloser, rep int) error {
	c := cl.getClient()
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
func (cl *Client) CreateReplica(ctx context.Context, key string, reader io.ReadCloser) (string, error) {
	c := cl.getClient()
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
func (cl *Client) UpdateFileReplica(ctx context.Context, key string, vids []string, replica int) error {
	c := cl.getClient()
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
func (cl *Client) GetFile(ctx context.Context, key string) (io.ReadCloser, error) {
	c := cl.getClient()
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
	Ok       bool
	VolumeID string
	Err      string `json:"error,omitempty"`
}

// HasFile returns if the file exists
func (cl *Client) HasFile(ctx context.Context, key string) (string, bool, error) {
	c := cl.getClient()
	response, err := c.hasFile(ctx, hasFileRequest{Key: key})
	if err != nil {
		return "", false, err
	}

	resp := response.(hasFileResponse)

	if resp.Err != "" {
		return "", false, errors.New(resp.Err)
	}

	return resp.VolumeID, resp.Ok, nil
}

type deleteFileRequest struct {
	Key string
}

type deleteFileResponse struct {
	Err string `json:"error,omitempty"`
}

// DeleteFile deletes the file with the given key
func (cl *Client) DeleteFile(ctx context.Context, key string) error {
	c := cl.getClient()
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
