package client

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"

	"github.com/xescugc/rebost/volume"
)

var successCodeRe = regexp.MustCompile(`2\d\d`)

type errorResponse struct {
	Err string `json:"error,omitempty"`
}

type s3ErrorResponse struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

// parseErrorBody reads the response body and tries to extract an error message.
// It first tries JSON (internal routes), then XML (S3 routes).
func parseErrorBody(body io.Reader) string {
	b, err := io.ReadAll(body)
	if err != nil {
		return ""
	}

	var jerr errorResponse
	if json.Unmarshal(b, &jerr) == nil && jerr.Err != "" {
		return jerr.Err
	}

	var xerr s3ErrorResponse
	if xml.Unmarshal(b, &xerr) == nil && xerr.Message != "" {
		return xerr.Message
	}

	return ""
}

func (c *client) requestStream(ctx context.Context, method, url string) (io.ReadCloser, int64, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to create request %q: %w", url, err)
	}
	hresp, err := c.http.Do(req)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to do request %q: %w", url, err)
	}
	if !successCodeRe.MatchString(strconv.Itoa(hresp.StatusCode)) {
		if hresp.StatusCode == http.StatusInsufficientStorage {
			hresp.Body.Close()
			return nil, -1, volume.ErrNoSpace
		}
		defer hresp.Body.Close()
		if msg := parseErrorBody(hresp.Body); msg != "" {
			return nil, -1, errors.New(msg)
		}
		return nil, -1, fmt.Errorf("unexpected status %d", hresp.StatusCode)
	}
	return hresp.Body, hresp.ContentLength, nil
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
		if hresp.StatusCode == http.StatusInsufficientStorage {
			return hresp, volume.ErrNoSpace
		}
		if msg := parseErrorBody(hresp.Body); msg != "" {
			return hresp, errors.New(msg)
		}
		return hresp, fmt.Errorf("unexpected status code %d", hresp.StatusCode)
	} else if resp != nil && hresp.StatusCode != http.StatusNoContent {
		if err = json.NewDecoder(hresp.Body).Decode(resp); err != nil {
			return hresp, fmt.Errorf("failed to decode body on %q: %w", url, err)
		}
	}

	return hresp, nil
}
