package storing_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/storing"
)

func TestMakeHandler(t *testing.T) {
	var (
		key     = "fileName"
		content = []byte("content")
		ctrl    = gomock.NewController(t)
	)

	st := mock.NewStoring(ctrl)
	defer ctrl.Finish()

	h := storing.MakeHandler(st)
	server := httptest.NewServer(h)
	client := server.Client()

	st.EXPECT().CreateFile(key, gomock.Any()).Do(func(_ string, r io.Reader) {
		b, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, content, b)
	}).Return(nil).AnyTimes()
	st.EXPECT().GetFile(key).Return(bytes.NewBuffer(content), nil).AnyTimes()
	st.EXPECT().DeleteFile(key).Return(nil).AnyTimes()
	st.EXPECT().HasFile(gomock.Any()).DoAndReturn(func(k string) (bool, error) {
		if k == key {
			return true, nil
		}
		return false, nil
	}).AnyTimes()

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
				return content
			},
			EStatusCode: http.StatusOK,
		},
		{
			Name:        "DeleteFile",
			URL:         "/files/fileName",
			Method:      http.MethodDelete,
			EStatusCode: http.StatusNoContent,
		},
		{
			Name:        "HasFile(true)",
			URL:         "/files/fileName",
			Method:      http.MethodHead,
			EStatusCode: http.StatusNoContent,
		},
		{
			Name:        "HasFile(false)",
			URL:         "/files/file",
			Method:      http.MethodHead,
			EStatusCode: http.StatusNotFound,
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
