package httptransport_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/volume"
	httptransport "github.com/xescugc/rebost/storing/transport/http"
)

func TestAuditMiddleware(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		path        string
		wantLogged  bool
		wantEvent   string
		wantKey     string
	}{
		{
			name:       "PUT object logs audit.create",
			method:     http.MethodPut,
			path:       "/bucket/key",
			wantLogged: true,
			wantEvent:  "audit.create",
			wantKey:    "bucket/key",
		},
		{
			name:       "GET object logs audit.access",
			method:     http.MethodGet,
			path:       "/bucket/key",
			wantLogged: true,
			wantEvent:  "audit.access",
			wantKey:    "bucket/key",
		},
		{
			name:       "DELETE object logs audit.delete",
			method:     http.MethodDelete,
			path:       "/bucket/key",
			wantLogged: true,
			wantEvent:  "audit.delete",
			wantKey:    "bucket/key",
		},
		{
			name:       "HEAD object logs audit.stat",
			method:     http.MethodHead,
			path:       "/bucket/key",
			wantLogged: true,
			wantEvent:  "audit.stat",
			wantKey:    "bucket/key",
		},
		{
			name:       "PUT /replicas/ does not log",
			method:     http.MethodPut,
			path:       "/replicas/somekey",
			wantLogged: false,
		},
		{
			name:       "GET /config does not log",
			method:     http.MethodGet,
			path:       "/config",
			wantLogged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			st := mock.NewStoring(ctrl)

			var buf bytes.Buffer
			auditLogger := slog.New(slog.NewJSONHandler(&buf, nil))

			// Set up expectations based on route
			switch {
			case tt.method == http.MethodPut && tt.path == "/bucket/key":
				st.EXPECT().CreateFile(gomock.Any(), "bucket/key", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			case tt.method == http.MethodGet && tt.path == "/bucket/key":
				st.EXPECT().StatFile(gomock.Any(), "bucket/key").Return(&volume.FileStat{Size: 5, VolumeID: "v1"}, nil)
				st.EXPECT().GetFile(gomock.Any(), "bucket/key").Return(io.NopCloser(bytes.NewBufferString("hello")), int64(5), nil)
			case tt.method == http.MethodDelete && tt.path == "/bucket/key":
				st.EXPECT().DeleteFile(gomock.Any(), "bucket/key").Return(nil)
			case tt.method == http.MethodHead && tt.path == "/bucket/key":
				st.EXPECT().StatFile(gomock.Any(), "bucket/key").Return(&volume.FileStat{Size: 5, VolumeID: "v1"}, nil)
			case tt.method == http.MethodPut && tt.path == "/replicas/somekey":
				st.EXPECT().CreateReplica(gomock.Any(), "somekey", gomock.Any(), gomock.Any(), gomock.Any()).Return("v1", nil)
			case tt.method == http.MethodGet && tt.path == "/config":
				st.EXPECT().Config(gomock.Any()).Return(&config.Config{}, nil)
			}

			h := httptransport.MakeHandler(st, &config.Config{}, func() bool { return true }, auditLogger)
			server := httptest.NewServer(h)
			defer server.Close()

			req, err := http.NewRequest(tt.method, server.URL+tt.path, bytes.NewBufferString("data"))
			require.NoError(t, err)

			resp, err := server.Client().Do(req)
			require.NoError(t, err)
			resp.Body.Close()

			// Give middleware a moment (it logs after ServeHTTP returns, still synchronous)
			_ = resp

			if !tt.wantLogged {
				assert.Empty(t, buf.String(), "expected no audit log for %s %s", tt.method, tt.path)
				return
			}

			require.NotEmpty(t, buf.String(), "expected audit log for %s %s", tt.method, tt.path)

			var entry map[string]interface{}
			err = json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry)
			require.NoError(t, err)

			assert.Equal(t, tt.wantEvent, entry["event"])
			assert.Equal(t, tt.wantKey, entry["key"])
			assert.NotEmpty(t, entry["caller_ip"])

			// Validate "time" field is RFC3339
			timeStr, ok := entry["time"].(string)
			require.True(t, ok)
			_, err = time.Parse(time.RFC3339, timeStr)
			assert.NoError(t, err)
		})
	}
}
