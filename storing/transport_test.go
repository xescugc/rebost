package storing_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/storing"
)

func TestMakeHandler(t *testing.T) {
	var (
		st  mock.Storing
		key = "fileName"
	)

	h := storing.MakeHandler(&st)
	server := httptest.NewServer(h)
	client := server.Client()

	st.CreateFileFn = func(k string, r io.Reader) error {
		assert.Equal(t, key, k)
		b, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, "content", string(b))
		return nil
	}

	st.GetFileFn = func(k string) (io.Reader, error) {
		assert.Equal(t, key, k)
		buff := bytes.NewBufferString("content")
		return buff, nil
	}

	st.DeleteFileFn = func(k string) error {
		assert.Equal(t, key, k)
		return nil
	}

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
			URL:         "/files/fileName",
			Method:      http.MethodPut,
			Body:        []byte("content"),
			EStatusCode: http.StatusCreated,
		},
		{
			Name:   "GetFile",
			URL:    "/files/fileName",
			Method: http.MethodGet,
			EBody: func() []byte {
				return []byte("content")
			},
			EStatusCode: http.StatusOK,
		},
		{
			Name:        "DeleteFile",
			URL:         "/files/fileName",
			Method:      http.MethodDelete,
			EStatusCode: http.StatusNoContent,
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
				b, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, tt.EBody(), b)
			}

			require.Equal(t, tt.EStatusCode, resp.StatusCode)
		})
	}
}
