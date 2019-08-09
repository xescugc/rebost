package integration_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http/httptest"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/boltdb"
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/fs"
	"github.com/xescugc/rebost/membership"
	"github.com/xescugc/rebost/storing"
	"github.com/xescugc/rebost/util"
	"github.com/xescugc/rebost/volume"
)

// newClient initializes a new client.Client with a random Port and random MemberlistBindPort with the
// given name and remote to connect to.
// It returns the clien.Client, the URL of the server the client it's connected to and a cancelFn that
// cleans the server.
func newClient(t *testing.T, name string, remote string) (*client.Client, string, cancelFn) {
	port, err := util.FreePort()
	require.NoError(t, err)

	tmpDir, err := ioutil.TempDir("", "rebost")
	if err != nil {
		panic(err)
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%s", strconv.Itoa(port)))
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

	replicas, err := boltdb.NewReplicaRepository(bdb)
	require.NoError(t, err)

	suow := fs.UOWWithFs(boltdb.NewUOW(bdb))

	v, err := volume.New(tmpDir, files, idxkeys, replicas, osfs, suow)
	require.NoError(t, err)

	mbp, err := util.FreePort()
	require.NoError(t, err)

	cfg := &config.Config{
		Port:               port,
		MemberlistName:     name,
		MemberlistBindPort: mbp,
		Remote:             remote,
	}

	m, err := membership.New(cfg, []volume.Local{v}, cfg.Remote)
	require.NoError(t, err)

	s := storing.New(cfg, m)
	h := storing.MakeHandler(s)

	server.Config.Handler = h

	u := fmt.Sprintf("http://localhost:%d", port)

	cl, err := client.New(u)
	require.NoError(t, err)

	return cl, u, func() {
		bdb.Close()
		os.RemoveAll(tmpDir)
		server.Close()
	}
}

func createDB(p string) (*bolt.DB, error) {
	db, err := bolt.Open(path.Join(p, "my.db"), 0600, nil)
	if err != nil {
		return nil, err
	}
	return db, nil
}
