package storing_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/boltdb"
	"github.com/xescugc/rebost/fs"
	"github.com/xescugc/rebost/storing"
	"github.com/xescugc/rebost/volume"
)

var (
	client *http.Client
	server *httptest.Server
)

func TestE2E(t *testing.T) {
	vp := "./test_data"
	os.MkdirAll(vp, os.ModePerm)
	defer os.RemoveAll(vp)

	osfs := afero.NewOsFs()
	bdb, err := createDB(vp)
	require.NoError(t, err)
	defer bdb.Close()
	files, err := boltdb.NewFileRepository(bdb)
	require.NoError(t, err)
	idxkeys, err := boltdb.NewIDXKeyRepository(bdb)
	require.NoError(t, err)
	suow := fs.UOWWithFs(boltdb.NewUOW(bdb))
	v, err := volume.New(vp, files, idxkeys, osfs, suow)
	require.NoError(t, err)

	s := storing.New([]volume.Volume{v})
	h := storing.MakeHandler(s)

	server = httptest.NewServer(h)
	client = server.Client()

	key := "xescugc"
	content := []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.")

	// TODO: Add test for multipart upload
	t.Run("CreateFile", func(t *testing.T) {
		resp := makeRequest(t, http.MethodPut, "/files/"+key, content)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("GetFile", func(t *testing.T) {
		resp := makeRequest(t, http.MethodGet, "/files/"+key, nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		b, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		resp.Body.Close()

		// Compare them with string to make it readable
		assert.Equal(t, string(content), string(b))
	})
}

func makeRequest(t *testing.T, method, url string, body []byte) *http.Response {
	req, err := http.NewRequest(method, server.URL+url, bytes.NewBuffer(body))
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func createDB(p string) (*bolt.DB, error) {
	db, err := bolt.Open(path.Join(p, "my.db"), 0600, nil)
	if err != nil {
		return nil, err
	}
	return db, nil
}
