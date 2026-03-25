package integration_test

import (
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"log/slog"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/boltdb"
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/fs"
	"github.com/xescugc/rebost/membership"
	"github.com/xescugc/rebost/storing"
	httptransport "github.com/xescugc/rebost/storing/transport/http"
	"github.com/xescugc/rebost/util"
	"github.com/xescugc/rebost/volume"
	bolt "go.etcd.io/bbolt"
)

// newNode starts a test node and returns its URL, volume ID, service, membership,
// and a cancel function. The cfg argument allows callers to set extra config fields
// (e.g. S3 credentials) before the node is wired up. Pass nil for defaults.
func newNode(t *testing.T, name string, remote string, cfgFn func(*config.Config)) (string, string, storing.Service, storing.Membership, cancelFn) {
	t.Helper()

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
	cfg.Timing.VolumeDowntime = time.Second

	if cfgFn != nil {
		cfgFn(cfg)
	}

	tmpDir, err := os.MkdirTemp("", "rebost")
	require.NoError(t, err)

	l, err := net.Listen("tcp", fmt.Sprintf(":%s", strconv.Itoa(port)))
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true}))

	server := httptest.NewUnstartedServer(nil)
	server.Listener.Close()
	server.Listener = l
	server.Start()

	osfs := afero.NewOsFs()

	bdb, err := createDB(tmpDir)
	require.NoError(t, err)

	suow, err := boltdb.NewUOW(bdb)
	require.NoError(t, err)

	suow = fs.UOWWithFs(osfs, suow)

	v, err := volume.New(tmpDir, osfs, logger, suow, 0, 0)
	require.NoError(t, err)

	m, err := membership.New(cfg, []volume.Local{v}, cfg.Remote, logger)
	require.NoError(t, err)

	s, err := storing.New(cfg, m, logger)
	require.NoError(t, err)

	m.SetStatus(membership.StatusRunning)

	server.Config.Handler = httptransport.MakeHandler(s, cfg, m.IsRunning)

	u := fmt.Sprintf("http://localhost:%d", port)

	return u, v.ID(), s, m, func() {
		m.Leave()
		bdb.Close()
		os.RemoveAll(tmpDir)
		server.Close()
	}
}

// newClient initializes a new client.Client with a random Port and random MemberlistBindPort with the
// given name and remote to connect to.
// It returns the client.Client, the URL of the server the client is connected to, the volume ID and a cancelFn that
// cleans the server.
func newClient(t *testing.T, name string, remote string) (*client.Client, string, string, cancelFn) {
	t.Helper()

	u, vid, _, _, cancel := newNode(t, name, remote, nil)

	cl, err := client.New(u)
	require.NoError(t, err)

	return cl, u, vid, cancel
}

// newClientWithAuth is like newClient but starts the node with S3 auth enabled.
// It returns the server URL and a cancel function (no rebost client since S3 clients are used directly).
func newClientWithAuth(t *testing.T, name string, remote string, accessKey string, secretKey string) (string, cancelFn) {
	t.Helper()
	return newClientWithAuthMode(t, name, remote, accessKey, secretKey, "")
}

// newClientWithAuthMode is like newClientWithAuth but also sets the S3 auth mode.
func newClientWithAuthMode(t *testing.T, name string, remote string, accessKey string, secretKey string, authMode string) (string, cancelFn) {
	t.Helper()

	u, _, _, _, cancel := newNode(t, name, remote, func(cfg *config.Config) {
		cfg.S3.AccessKey = accessKey
		cfg.S3.SecretKey = secretKey
		cfg.S3.AuthMode = authMode
	})

	return u, cancel
}

func createDB(p string) (*bolt.DB, error) {
	db, err := bolt.Open(path.Join(p, "my.db"), 0600, nil)
	if err != nil {
		return nil, err
	}
	return db, nil
}
