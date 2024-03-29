package integration_test

import (
	"fmt"
	"log"
	"net"
	"net/http/httptest"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	kitlog "github.com/go-kit/kit/log"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/boltdb"
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/fs"
	"github.com/xescugc/rebost/membership"
	"github.com/xescugc/rebost/storing"
	"github.com/xescugc/rebost/util"
	"github.com/xescugc/rebost/volume"
	bolt "go.etcd.io/bbolt"
)

// newClient initializes a new client.Client with a random Port and random MemberlistBindPort with the
// given name and remote to connect to.
// It returns the clien.Client, the URL of the server the client it's connected to, the volume ID  and a cancelFn that
// cleans the server.
func newClient(t *testing.T, name string, remote string) (*client.Client, string, string, cancelFn) {
	port, err := util.FreePort()
	require.NoError(t, err)

	vp := viper.New()
	vp.Set("name", name)
	vp.Set("remote", remote)
	vp.Set("port", port)
	cfg, err := config.New(vp)
	require.NoError(t, err)

	// We set it outside of the New because we want to be fast for testing
	// and it has a validation on the New to not allow it
	cfg.VolumeDowntime = time.Second

	tmpDir, err := os.MkdirTemp("", "rebost")
	if err != nil {
		panic(err)
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%s", strconv.Itoa(port)))
	if err != nil {
		log.Fatal(err)
	}

	logger := kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stdout))
	logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC, "caller", kitlog.DefaultCaller)

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

	idxvolumes, err := boltdb.NewIDXVolumeRepository(bdb)
	require.NoError(t, err)

	replicas, err := boltdb.NewReplicaRepository(bdb)
	require.NoError(t, err)

	state, err := boltdb.NewStateRepository(bdb)
	require.NoError(t, err)

	suow := fs.UOWWithFs(boltdb.NewUOW(bdb))

	v, err := volume.New(tmpDir, files, idxkeys, idxvolumes, replicas, state, osfs, logger, suow)
	require.NoError(t, err)

	m, err := membership.New(cfg, []volume.Local{v}, cfg.Remote, logger)
	require.NoError(t, err)

	s, err := storing.New(cfg, m, logger)
	require.NoError(t, err)

	h := storing.MakeHandler(s)

	server.Config.Handler = h

	u := fmt.Sprintf("http://localhost:%d", port)

	cl, err := client.New(u)
	require.NoError(t, err)

	return cl, u, v.ID(), func() {
		m.Leave()
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
