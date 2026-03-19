package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
)

var successCodeRe = regexp.MustCompile(`2\d\d`)

type errorResponse struct {
	Err string `json:"error,omitempty"`
}

func (c *client) requestStream(ctx context.Context, method, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request %q: %w", url, err)
	}
	hresp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request %q: %w", url, err)
	}
	if !successCodeRe.MatchString(strconv.Itoa(hresp.StatusCode)) {
		defer hresp.Body.Close()
		var eresp errorResponse
		json.NewDecoder(hresp.Body).Decode(&eresp)
		if eresp.Err != "" {
			return nil, errors.New(eresp.Err)
		}
		return nil, fmt.Errorf("unexpected status %d", hresp.StatusCode)
	}
	return hresp.Body, nil
}

func (c *client) request(ctx context.Context, method, url string, body, resp interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		if r, ok := body.(io.Reader); ok {
			bodyReader = r
		} else {
			buff := bytes.NewBuffer(nil)
			if err := json.NewEncoder(buff).Encode(body); err != nil {
				return nil, fmt.Errorf("failed to marshal body on %q: %w", url, err)
			}
			bodyReader = buff
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request %q: %w", url, err)
	}

	hresp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request %q: %w", url, err)
	}
	defer hresp.Body.Close()

	if !successCodeRe.MatchString(strconv.Itoa(hresp.StatusCode)) {
		var eresp errorResponse
		json.NewDecoder(hresp.Body).Decode(&eresp)
		if eresp.Err != "" {
			return hresp, errors.New(eresp.Err)
		}
		return hresp, nil
	} else if resp != nil && hresp.StatusCode != http.StatusNoContent {
		if err = json.NewDecoder(hresp.Body).Decode(resp); err != nil {
			return hresp, fmt.Errorf("failed to decode body on %q: %w", url, err)
		}
	}

	return hresp, nil
}
