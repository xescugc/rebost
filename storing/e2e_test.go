package storing_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/boltdb"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/fs"
	"github.com/xescugc/rebost/membership"
	"github.com/xescugc/rebost/storing"
	"github.com/xescugc/rebost/volume"
)

func TestE2E(t *testing.T) {
	n1 := newNode(t, &config.Config{Port: 5011, MemberlistName: "n1", MemberlistBindPort: 5001}, "")
	defer n1.Finish()

	n2 := newNode(t, &config.Config{Port: 5012, MemberlistName: "n2", MemberlistBindPort: 5002}, "localhost:5001")
	defer n2.Finish()

	n3 := newNode(t, &config.Config{Port: 5013, MemberlistName: "n3", MemberlistBindPort: 5003}, "localhost:5002")
	defer n3.Finish()

	key := "xescugc"
	content := []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.")
	// TODO: Add test for multipart upload
	t.Run("CreateFile", func(t *testing.T) {
		resp := n1.makeRequest(t, http.MethodPut, "/files/"+key, content)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("GetFile", func(t *testing.T) {
		tests := []struct {
			Name string
			Node *node
		}{
			{
				Name: "FromN1",
				Node: n1,
			},
			{
				Name: "FromN2",
				Node: n2,
			},
			{
				Name: "FromN3",
				Node: n3,
			},
		}
		for _, tt := range tests {
			t.Run(tt.Name, func(t *testing.T) {
				resp := tt.Node.makeRequest(t, http.MethodGet, "/files/"+key, nil)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				b, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)
				resp.Body.Close()

				// Compare them with string to make it readable
				assert.Equal(t, string(content), string(b))
			})
		}
	})
	//time.Sleep(time.Second * 30)
}

func createDB(p string) (*bolt.DB, error) {
	db, err := bolt.Open(path.Join(p, "my.db"), 0600, nil)
	if err != nil {
		return nil, err
	}
	return db, nil
}

type node struct {
	server *httptest.Server
	client *http.Client
	tmpDir string

	db *bolt.DB
}

func newNode(t *testing.T, cfg *config.Config, remote string) *node {
	tmpDir, err := ioutil.TempDir("", "rebost")
	if err != nil {
		panic(err)
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%s", strconv.Itoa(cfg.Port)))
	if err != nil {
		log.Fatal(err)
	}

	server := httptest.NewUnstartedServer(nil)
	server.Listener.Close()
	server.Listener = l
	server.Start()

	osfs := afero.NewOsFs()

	bdb, err := createDB(tmpDir)
	require.NoError(t, err)

	files, err := boltdb.NewFileRepository(bdb)
	require.NoError(t, err)

	idxkeys, err := boltdb.NewIDXKeyRepository(bdb)
	require.NoError(t, err)

	suow := fs.UOWWithFs(boltdb.NewUOW(bdb))

	v, err := volume.New(tmpDir, files, idxkeys, osfs, suow)
	require.NoError(t, err)

	m, err := membership.New(cfg, []volume.Volume{v}, remote)
	require.NoError(t, err)

	s := storing.New(m)
	h := storing.MakeHandler(s)

	server.Config.Handler = h
	client := server.Client()

	return &node{
		db:     bdb,
		server: server,
		client: client,
		tmpDir: tmpDir,
	}
}

func (n *node) makeRequest(t *testing.T, method, url string, body []byte) *http.Response {
	req, err := http.NewRequest(method, n.server.URL+url, bytes.NewBuffer(body))
	require.NoError(t, err)

	resp, err := n.client.Do(req)
	require.NoError(t, err)

	return resp
}

func (n *node) Finish() {
	n.db.Close()
	os.RemoveAll(n.tmpDir)
	n.server.Close()
}
