package httptransport_test

import (
	"log/slog"

	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/mock"
	httptransport "github.com/xescugc/rebost/storing/transport/http"
)

func TestLiveHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := mock.NewStoring(ctrl)
	h := httptransport.MakeHandler(st, &config.Config{}, func() bool { return false }, slog.Default())
	server := httptest.NewServer(h)
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/live")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

func TestHealthHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := mock.NewStoring(ctrl)
	h := httptransport.MakeHandler(st, &config.Config{}, func() bool { return false }, slog.Default())
	server := httptest.NewServer(h)
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

func TestReadyHandler_NodeNotRunning(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := mock.NewStoring(ctrl)
	h := httptransport.MakeHandler(st, &config.Config{}, func() bool { return false }, slog.Default())
	server := httptest.NewServer(h)
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/ready")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "not_ready", body["status"])
	assert.Equal(t, "node not running", body["reason"])
}

func TestReadyHandler_VolumeUnhealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := mock.NewStoring(ctrl)
	st.EXPECT().Ready(gomock.Any()).Return(errors.New("volume abc unhealthy: disk error"))

	h := httptransport.MakeHandler(st, &config.Config{}, func() bool { return true }, slog.Default())
	server := httptest.NewServer(h)
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/ready")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "not_ready", body["status"])
	assert.Equal(t, "volume abc unhealthy: disk error", body["reason"])
}

func TestReadyHandler_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := mock.NewStoring(ctrl)
	st.EXPECT().Ready(gomock.Any()).Return(nil)

	h := httptransport.MakeHandler(st, &config.Config{}, func() bool { return true }, slog.Default())
	server := httptest.NewServer(h)
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/ready")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}
