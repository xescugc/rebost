package httptransport_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/storing/model"
	httptransport "github.com/xescugc/rebost/storing/transport/http"
	"github.com/xescugc/rebost/volume"
)

func TestMakeHandler(t *testing.T) {
	var (
		key                  = "testbucket/fileName"
		content              = []byte("content")
		ctrl                 = gomock.NewController(t)
		cfg                  = config.Config{Name: "Pepito"}
		createReplicaVolmeID = "createReplicaVolmeID"
		rep                  = 2
		ttl                  = 2 * time.Minute
		ca                   = time.Now()
		vid                  = "vid"
	)

	st := mock.NewStoring(ctrl)
	defer ctrl.Finish()

	h := httptransport.MakeHandler(st, &config.Config{})
	server := httptest.NewServer(h)
	client := server.Client()

	st.EXPECT().CreateFile(gomock.Any(), key, gomock.Any(), rep, ttl, timeMatcher{ca}).Do(func(_ context.Context, _ string, r io.Reader, _ int, _ time.Duration, _ time.Time) {
		b, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, content, b)
	}).Return(nil).AnyTimes()
	st.EXPECT().StatFile(gomock.Any(), key).Return(&volume.FileStat{Size: int64(len(content))}, nil).AnyTimes()
	st.EXPECT().GetFile(gomock.Any(), key).Return(io.NopCloser(bytes.NewBuffer(content)), int64(-1), nil).AnyTimes()
	st.EXPECT().DeleteFile(gomock.Any(), key).Return(nil).AnyTimes()
	st.EXPECT().HasFile(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, k string) (string, bool, error) {
		if k == key {
			return vid, true, nil
		}
		return "", false, nil
	}).AnyTimes()
	st.EXPECT().Config(gomock.Any()).Return(&cfg, nil)
	st.EXPECT().CreateReplica(gomock.Any(), "fileName", gomock.Any(), ttl, timeMatcher{ca}).Do(func(_ context.Context, _ string, r io.Reader, _ time.Duration, _ time.Time) {
		b, err := io.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, content, b)
	}).Return(createReplicaVolmeID, nil).AnyTimes()
	st.EXPECT().UpdateFileReplica(gomock.Any(), "fileName", []string{"1", "2"}, rep).Return(nil)

	tests := []struct {
		Name        string
		URL         string
		Method      string
		Body        []byte
		EBody       func() []byte
		EStatusCode int
	}{
		{
			Name:        "CreateFile",
			URL:         fmt.Sprintf("/testbucket/fileName?replica=%d&ttl=%s&created_at=%s", rep, ttl, url.QueryEscape(ca.Format(time.RFC3339))),
			Method:      http.MethodPut,
			Body:        []byte("content"),
			EStatusCode: http.StatusOK,
		},
		{
			Name:   "GetFile",
			URL:    "/testbucket/fileName",
			Method: http.MethodGet,
			EBody: func() []byte {
				return content
			},
			EStatusCode: http.StatusOK,
		},
		{
			Name:        "DeleteFile",
			URL:         "/testbucket/fileName",
			Method:      http.MethodDelete,
			EStatusCode: http.StatusNoContent,
		},
		{
			Name:        "HasFile(true)",
			URL:         "/testbucket/fileName",
			Method:      http.MethodHead,
			EStatusCode: http.StatusOK,
		},
		{
			Name:        "HasFile(false)",
			URL:         "/testbucket/otherFile",
			Method:      http.MethodHead,
			EStatusCode: http.StatusNotFound,
		},
		{
			Name:        "Config",
			URL:         "/config",
			Method:      http.MethodGet,
			EStatusCode: http.StatusOK,
			EBody: func() []byte {
				cfg := model.Config{Name: "Pepito"}
				b, _ := json.Marshal(cfg)
				return []byte(fmt.Sprintf(`{"data":%s}`, b))
			},
		},
		{
			Name:        "CreateReplica",
			URL:         fmt.Sprintf("/replicas/fileName?ttl=%s&created_at=%s", ttl, url.QueryEscape(ca.Format(time.RFC3339))),
			Method:      http.MethodPut,
			Body:        []byte("content"),
			EStatusCode: http.StatusOK,
			EBody: func() []byte {
				cr := model.CreateReplica{VolumeID: createReplicaVolmeID}
				b, _ := json.Marshal(cr)
				return []byte(fmt.Sprintf(`{"data":%s}`, b))
			},
		},
		{
			Name:        "UpdateFileReplica",
			URL:         "/replicas/fileName",
			Body:        []byte(fmt.Sprintf(`{"volume_ids": ["1","2"], "replica": %d}`, rep)),
			Method:      http.MethodPatch,
			EStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			req, err := http.NewRequest(tt.Method, server.URL+tt.URL, bytes.NewBuffer(tt.Body))
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)

			if tt.EBody != nil {
				defer resp.Body.Close()
				b, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, tt.EBody(), b)
			}

			require.Equal(t, tt.EStatusCode, resp.StatusCode)
		})
	}
}

func TestPutObjectErrClusterFull(t *testing.T) {
	var (
		ctrl = gomock.NewController(t)
		key  = "testbucket/fileName"
		rep  = 0
		ttl  = time.Duration(0)
		ca   = time.Time{}
	)
	st := mock.NewStoring(ctrl)
	defer ctrl.Finish()

	st.EXPECT().CreateFile(gomock.Any(), key, gomock.Any(), rep, ttl, ca).Return(volume.ErrClusterFull)

	h := httptransport.MakeHandler(st, &config.Config{})
	server := httptest.NewServer(h)
	client := server.Client()

	req, err := http.NewRequest(http.MethodPut, server.URL+"/testbucket/fileName", bytes.NewBuffer([]byte("data")))
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInsufficientStorage, resp.StatusCode)

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(b), "InsufficientStorage")
}

// TestPutObjectMultipartPipeClosedOnBodyError verifies that when a multipart
// request body errors mid-stream, the pipe writer is closed with the error so
// the CreateFile reader is unblocked rather than hanging indefinitely.
func TestPutObjectMultipartPipeClosedOnBodyError(t *testing.T) {
	ctrl := gomock.NewController(t)
	st := mock.NewStoring(ctrl)
	defer ctrl.Finish()

	createFileDone := make(chan struct{})
	st.EXPECT().CreateFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, r io.Reader, _ int, _ time.Duration, _ time.Time) error {
			io.ReadAll(r) // blocks forever if ppw is not closed with error
			close(createFileDone)
			return nil
		})

	h := httptransport.MakeHandler(st, &config.Config{})
	server := httptest.NewServer(h)
	defer server.Close()

	pr, pw := io.Pipe()
	req, err := http.NewRequest(http.MethodPut, server.URL+"/testbucket/key", pr)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=testboundary")

	go server.Client().Do(req) //nolint:errcheck

	// Write start boundary then error — NextPart() fails reading part headers
	_, werr := pw.Write([]byte("--testboundary\r\n"))
	require.NoError(t, werr)
	pw.CloseWithError(errors.New("connection reset"))

	select {
	case <-createFileDone:
		// ppw.CloseWithError unblocked the pipe reader
	case <-time.After(3 * time.Second):
		t.Fatal("handler blocked: pipe writer not closed with error on NextPart failure")
	}
}

// TestCreateReplicaMultipartPipeClosedOnBodyError verifies the same guarantee
// for the createReplicaHandler.
func TestCreateReplicaMultipartPipeClosedOnBodyError(t *testing.T) {
	ctrl := gomock.NewController(t)
	st := mock.NewStoring(ctrl)
	defer ctrl.Finish()

	createReplicaDone := make(chan struct{})
	st.EXPECT().CreateReplica(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, r io.Reader, _ time.Duration, _ time.Time) (string, error) {
			io.ReadAll(r) // blocks forever if ppw is not closed with error
			close(createReplicaDone)
			return "", nil
		})

	h := httptransport.MakeHandler(st, &config.Config{})
	server := httptest.NewServer(h)
	defer server.Close()

	pr, pw := io.Pipe()
	req, err := http.NewRequest(http.MethodPut, server.URL+"/replicas/testbucket/key", pr)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=testboundary")

	go server.Client().Do(req) //nolint:errcheck

	_, werr := pw.Write([]byte("--testboundary\r\n"))
	require.NoError(t, werr)
	pw.CloseWithError(errors.New("connection reset"))

	select {
	case <-createReplicaDone:
		// ppw.CloseWithError unblocked the pipe reader
	case <-time.After(3 * time.Second):
		t.Fatal("handler blocked: pipe writer not closed with error on NextPart failure")
	}
}

type timeMatcher struct {
	t time.Time
}

func (tm timeMatcher) Matches(x interface{}) bool {
	t, ok := x.(time.Time)
	if !ok {
		return false
	}

	return t.Format(time.RFC3339) == tm.t.Format(time.RFC3339)
}

func (tm timeMatcher) String() string {
	return fmt.Sprintf("is equal to %s", tm.t.Format(time.RFC3339))
}
