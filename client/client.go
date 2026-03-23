package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/storing/model"
	"github.com/xescugc/rebost/volume"
)

// Client is the client structure that fulfills the storing.Service
// interface and it's meant to be used to access to a remote node
type Client struct {
	clientsLock sync.Mutex
	nextClient  int
	clients     []*client
}

type client struct {
	url  string
	http *http.Client
}

// New returns a client to connect to a remote Storing service
func New(hosts ...string) (*Client, error) {
	cl := &Client{
		clients: make([]*client, len(hosts)),
	}
	for i, h := range hosts {
		if h == "" {
			return nil, fmt.Errorf("can't initialize the %q with an empty host", "rebost")
		}
		if !strings.HasPrefix(h, "http") {
			h = fmt.Sprintf("http://%s", h)
		}
		cl.clients[i] = &client{
			url:  strings.TrimRight(h, "/"),
			http: &http.Client{},
		}
	}

	return cl, nil
}

// getClient returns the next client from the list of clients to use
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
	var resp getConfigResponse
	if _, err := c.request(ctx, http.MethodGet, fmt.Sprintf("%s/config", c.url), nil, &resp); err != nil {
		return nil, err
	}
	return model.ToConfig(resp.Data), nil
}

// CreateFile creates a file with the given key and the r content with rep replicas
func (cl *Client) CreateFile(ctx context.Context, key string, r io.ReadCloser, rep int, ttl time.Duration, ca time.Time) error {
	c := cl.getClient()
	q := url.Values{}
	q.Set("replica", strconv.Itoa(rep))
	q.Set("ttl", ttl.String())
	q.Set("created_at", ca.Format(time.RFC3339))
	u := fmt.Sprintf("%s/%s?%s", c.url, key, q.Encode())
	_, err := c.request(ctx, http.MethodPut, u, r, nil)
	return err
}

type createReplicaResponse struct {
	Data model.CreateReplica `json:"data,omitempty"`
	Err  string              `json:"error,omitempty"`
}

// CreateReplica creates a new replica to the Node
func (cl *Client) CreateReplica(ctx context.Context, key string, reader io.ReadCloser, ttl time.Duration, ca time.Time) (string, error) {
	c := cl.getClient()
	q := url.Values{}
	q.Set("ttl", ttl.String())
	q.Set("created_at", ca.Format(time.RFC3339))
	u := fmt.Sprintf("%s/replicas/%s?%s", c.url, key, q.Encode())
	var cresp createReplicaResponse
	if _, err := c.request(ctx, http.MethodPut, u, reader, &cresp); err != nil {
		return "", err
	}
	return cresp.Data.VolumeID, nil
}

// UpdateFileReplica updates the file replica information
func (cl *Client) UpdateFileReplica(ctx context.Context, key string, vids []string, replica int) error {
	c := cl.getClient()
	u := fmt.Sprintf("%s/replicas/%s", c.url, key)
	body := model.UpdateFileReplica{VolumeIDs: vids, Replica: replica}
	_, err := c.request(ctx, http.MethodPatch, u, body, nil)
	return err
}

// GetFile returns the requested file
func (cl *Client) GetFile(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	c := cl.getClient()
	u := fmt.Sprintf("%s/%s", c.url, key)
	return c.requestStream(ctx, http.MethodGet, u)
}

// StatFile returns metadata for the file by issuing a HEAD request to the remote node.
func (cl *Client) StatFile(ctx context.Context, key string) (*volume.FileStat, error) {
	c := cl.getClient()
	u := fmt.Sprintf("%s/%s", c.url, key)
	hresp, err := c.request(ctx, http.MethodHead, u, nil, nil)
	if err != nil {
		if hresp != nil && hresp.StatusCode == http.StatusNotFound {
			return nil, errors.New("not found")
		}
		return nil, err
	}
	stat := &volume.FileStat{
		Size: hresp.ContentLength,
		ETag: hresp.Header.Get("ETag"),
	}
	if lm := hresp.Header.Get("Last-Modified"); lm != "" {
		stat.ModTime, _ = http.ParseTime(lm)
	}
	return stat, nil
}

// HasFile returns if the file exists
func (cl *Client) HasFile(ctx context.Context, key string) (string, bool, error) {
	c := cl.getClient()
	u := fmt.Sprintf("%s/%s", c.url, key)
	hresp, err := c.request(ctx, http.MethodHead, u, nil, nil)
	if err != nil {
		if hresp != nil {
			// HTTP-level error (non-2xx): file not found or unavailable, not a network failure
			vid := hresp.Header.Get(model.HasFileVolumeIDHeader)
			return vid, false, nil
		}
		return "", false, err
	}
	vid := hresp.Header.Get(model.HasFileVolumeIDHeader)
	return vid, hresp.StatusCode == http.StatusOK, nil
}

// DeleteFile deletes the file with the given key
func (cl *Client) DeleteFile(ctx context.Context, key string) error {
	c := cl.getClient()
	u := fmt.Sprintf("%s/%s", c.url, key)
	_, err := c.request(ctx, http.MethodDelete, u, nil, nil)
	return err
}
