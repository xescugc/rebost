package storing_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/storing"
	httptransport "github.com/xescugc/rebost/storing/transport/http"
	"github.com/xescugc/rebost/volume"
)

var (
	noRep int
)

func TestCreateFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			buff = io.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 2
			ttl  = 2 * time.Minute
			ca   = time.Now()
		)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		v.EXPECT().CreateFile(gomock.Any(), key, gomock.Any(), rep, ttl, ca).Return(nil)

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})
		m.EXPECT().NodesWithCapacity(gomock.Any()).Return(nil).AnyTimes()

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.CreateFile(ctx, key, buff, rep, ttl, ca)
		require.NoError(t, err)
	})
	t.Run("SuccessWithConfigReplica", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			buff = io.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 2
			ttl  = 2 * time.Minute
			ca   = time.Now()
		)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		v.EXPECT().CreateFile(gomock.Any(), key, gomock.Any(), rep, ttl, ca).Return(nil)

		// It's AnyTimes as we have the config witha number of replicas
		// which activates the goroutines that also calls this
		m.EXPECT().LocalVolumes().Return([]volume.Local{v}).AnyTimes()
		m.EXPECT().NodesWithCapacity(gomock.Any()).Return(nil).AnyTimes()

		// This is also because of the goroutine, it may call it or not
		v.EXPECT().NextReplica(gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()
		v.EXPECT().NextDeletion(gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()
		v.EXPECT().NextScrub(gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()
		m.EXPECT().RemovedVolumeIDs().Return(nil).AnyTimes()

		s, err := storing.New(&config.Config{Replica: rep, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.CreateFile(ctx, key, buff, noRep, ttl, ca)
		require.NoError(t, err)
	})
	t.Run("FirstVolumeFullFallsBackToSecond", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			buff = io.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 2
			ttl  = 2 * time.Minute
			ca   = time.Now()
		)

		v1 := mock.NewVolumeLocal(ctrl)
		v2 := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		// Either volume may be randomly selected as primary by getLocalVolume.
		// Both expectations use AnyTimes so the test is not sensitive to selection order;
		// require.NoError below still verifies the fallback succeeds.
		v1.EXPECT().CreateFile(gomock.Any(), key, gomock.Any(), rep, ttl, ca).Return(volume.ErrNoSpace).AnyTimes()
		v2.EXPECT().CreateFile(gomock.Any(), key, gomock.Any(), rep, ttl, ca).Return(nil).AnyTimes()

		m.EXPECT().LocalVolumes().Return([]volume.Local{v1, v2})

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.CreateFile(ctx, key, buff, rep, ttl, ca)
		require.NoError(t, err)
	})
	t.Run("ClusterFullReturnsErrClusterFull", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			buff = io.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 2
			ttl  = 2 * time.Minute
			ca   = time.Now()
		)

		v1 := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		v1.EXPECT().CreateFile(gomock.Any(), key, gomock.Any(), rep, ttl, ca).Return(volume.ErrNoSpace)

		m.EXPECT().LocalVolumes().Return([]volume.Local{v1})
		m.EXPECT().NodesWithCapacity(gomock.Any()).Return(nil)

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.CreateFile(ctx, key, buff, rep, ttl, ca)
		assert.True(t, errors.Is(err, volume.ErrClusterFull))
	})
	t.Run("AllLocalVolumesFallBackToRemoteNode", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			buff = io.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 2
			ttl  = 2 * time.Minute
			ca   = time.Now()
		)

		v1 := mock.NewVolumeLocal(ctrl)
		s2 := mock.NewStoring(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		h := httptransport.MakeHandler(s2, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		defer server.Close()
		remoteClient, err := client.New(server.URL)
		require.NoError(t, err)

		v1.EXPECT().CreateFile(gomock.Any(), key, gomock.Any(), rep, ttl, ca).Return(volume.ErrNoSpace)

		m.EXPECT().LocalVolumes().Return([]volume.Local{v1})
		m.EXPECT().NodesWithCapacity(gomock.Any()).Return([]*client.Client{remoteClient})

		// ttl and ca may be truncated/rounded during HTTP serialization, so use Any()
		s2.EXPECT().CreateFile(gomock.Any(), key, gomock.Any(), rep, gomock.Any(), gomock.Any()).Return(nil)

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.CreateFile(ctx, key, buff, rep, ttl, ca)
		require.NoError(t, err)
	})
	t.Run("SuccessMultiVolume", func(t *testing.T) {
		t.Skip("Not yet thought")
	})
	t.Run("ProxyNodeDelegatesToRemote", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			buff = io.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 2
			ttl  = 2 * time.Minute
			ca   = time.Now()
		)

		s2 := mock.NewStoring(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		h := httptransport.MakeHandler(s2, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		defer server.Close()
		remoteClient, err := client.New(server.URL)
		require.NoError(t, err)

		m.EXPECT().LocalVolumes().Return([]volume.Local{})
		m.EXPECT().NodesWithCapacity(gomock.Any()).Return([]*client.Client{remoteClient})

		s2.EXPECT().CreateFile(gomock.Any(), key, gomock.Any(), rep, gomock.Any(), gomock.Any()).Return(nil)

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.CreateFile(ctx, key, buff, rep, ttl, ca)
		require.NoError(t, err)
	})
}

func TestGetFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			vid  = "vid"
		)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})

		v.EXPECT().HasFile(gomock.Any(), key).Return(vid, true, nil)
		v.EXPECT().GetFile(gomock.Any(), key).Return(io.NopCloser(bytes.NewBufferString("expectedcontent")), int64(-1), nil)

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		ior, _, err := s.GetFile(ctx, key)

		require.NoError(t, err)

		b, _ := io.ReadAll(ior)
		assert.Equal(t, "expectedcontent", string(b))
	})
	t.Run("SuccessMultiVolume", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			vid  = "vid"
		)
		v := mock.NewVolumeLocal(ctrl)
		s2 := mock.NewStoring(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		h := httptransport.MakeHandler(s2, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})
		m.EXPECT().Nodes().Return([]*client.Client{c})

		v.EXPECT().HasFile(gomock.Any(), key).Return("", false, nil)
		s2.EXPECT().HasFile(gomock.Any(), key).Return(vid, true, nil)
		s2.EXPECT().StatFile(gomock.Any(), key).Return(&volume.FileStat{Size: int64(len("expectedcontent"))}, nil)
		s2.EXPECT().GetFile(gomock.Any(), key).Return(io.NopCloser(bytes.NewBufferString("expectedcontent")), int64(-1), nil)

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		ior, _, err := s.GetFile(ctx, key)
		require.NoError(t, err)

		b, _ := io.ReadAll(ior)
		assert.Equal(t, "expectedcontent", string(b))
	})
	t.Run("ProxyNodeFindsOnRemote", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			vid  = "vid"
		)
		s2 := mock.NewStoring(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		h := httptransport.MakeHandler(s2, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		defer server.Close()
		c, err := client.New(server.URL)
		require.NoError(t, err)

		m.EXPECT().LocalVolumes().Return([]volume.Local{})
		m.EXPECT().Nodes().Return([]*client.Client{c})

		s2.EXPECT().HasFile(gomock.Any(), key).Return(vid, true, nil)
		s2.EXPECT().StatFile(gomock.Any(), key).Return(&volume.FileStat{Size: int64(len("expectedcontent"))}, nil)
		s2.EXPECT().GetFile(gomock.Any(), key).Return(io.NopCloser(bytes.NewBufferString("expectedcontent")), int64(-1), nil)

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		ior, _, err := s.GetFile(ctx, key)
		require.NoError(t, err)

		b, _ := io.ReadAll(ior)
		assert.Equal(t, "expectedcontent", string(b))
	})
}

func TestDeleteFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			ctrl = gomock.NewController(t)
			key  = "testbucket/expectedkey"
			ctx  = context.Background()
			vid  = "vid"
		)
		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})

		v.EXPECT().HasFile(gomock.Any(), key).Return(vid, true, nil)
		v.EXPECT().DeleteFile(gomock.Any(), key).Return(nil)

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.DeleteFile(ctx, key)
		require.NoError(t, err)
	})
	t.Run("SuccessMultiVolume", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			vid  = "vid"
		)
		v := mock.NewVolumeLocal(ctrl)
		s2 := mock.NewStoring(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		h := httptransport.MakeHandler(s2, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		c, err := client.New(server.URL)
		require.NoError(t, err)

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})
		m.EXPECT().Nodes().Return([]*client.Client{c})

		v.EXPECT().HasFile(gomock.Any(), key).Return("", false, nil)
		s2.EXPECT().HasFile(gomock.Any(), key).Return(vid, true, nil)
		s2.EXPECT().DeleteFile(gomock.Any(), key).Return(nil)

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.DeleteFile(ctx, key)
		require.NoError(t, err)
	})
	t.Run("ProxyNodeDeletesOnRemote", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			vid  = "vid"
		)
		s2 := mock.NewStoring(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		h := httptransport.MakeHandler(s2, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		defer server.Close()
		c, err := client.New(server.URL)
		require.NoError(t, err)

		m.EXPECT().LocalVolumes().Return([]volume.Local{})
		m.EXPECT().Nodes().Return([]*client.Client{c})

		s2.EXPECT().HasFile(gomock.Any(), key).Return(vid, true, nil)
		s2.EXPECT().DeleteFile(gomock.Any(), key).Return(nil)

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.DeleteFile(ctx, key)
		require.NoError(t, err)
	})
}

func TestHasFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			ctrl = gomock.NewController(t)
			key  = "testbucket/expectedkey"
			ctx  = context.Background()
			evid = "vid"
		)
		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})

		v.EXPECT().HasFile(gomock.Any(), key).Return(evid, true, nil)

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		vid, ok, err := s.HasFile(ctx, key)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, evid, vid)
	})
	t.Run("SuccessMultiVolume", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			evid = "vid"
		)
		v := mock.NewVolumeLocal(ctrl)
		v2 := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Local{v, v2})

		// This call is execute in parallel to all the volumes
		// so the order is "unexpected". This means that some
		// times the first call it's not done. That's why the
		// AnyTimes is used
		v.EXPECT().HasFile(gomock.Any(), key).Return("", false, nil).AnyTimes()
		v2.EXPECT().HasFile(gomock.Any(), key).Return(evid, true, nil)

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		vid, ok, err := s.HasFile(ctx, key)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, evid, vid)
	})
	t.Run("SuccessFalse", func(t *testing.T) {
		var (
			ctrl = gomock.NewController(t)
			key  = "testbucket/expectedkey"
			ctx  = context.Background()
		)
		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})

		v.EXPECT().HasFile(gomock.Any(), key).Return("", false, nil)

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		vid, ok, err := s.HasFile(ctx, key)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, "", vid)
	})
}

func TestConfig(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			ctrl   = gomock.NewController(t)
			ctx    = context.Background()
			expcfg = config.Config{Name: "Pepito", Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}
		)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		s, err := storing.New(&expcfg, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		cfg, err := s.Config(ctx)
		require.NoError(t, err)
		assert.Equal(t, &expcfg, cfg)
	})
}

func TestCreateReplica(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			key            = "expectedkey"
			buff           = io.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl           = gomock.NewController(t)
			ctx            = context.Background()
			createdToVolID = "createdToVolID"
			ttl            = 2 * time.Minute
			ca             = time.Now()
		)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		v.EXPECT().CreateFile(gomock.Any(), key, buff, 1, ttl, ca).Return(nil)

		// It's AnyTimes as we have the config witha number of replicas
		// which activates the goroutines that also calls this
		m.EXPECT().LocalVolumes().Return([]volume.Local{v}).AnyTimes()

		// This is also because of the goroutine, it may call it or not
		v.EXPECT().NextReplica(gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()
		v.EXPECT().NextDeletion(gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()
		v.EXPECT().NextScrub(gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()
		m.EXPECT().RemovedVolumeIDs().Return(nil).AnyTimes()

		v.EXPECT().ID().Return(createdToVolID)

		s, err := storing.New(&config.Config{Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		volID, err := s.CreateReplica(ctx, key, buff, ttl, ca)
		require.NoError(t, err)
		assert.Equal(t, createdToVolID, volID)
	})
	t.Run("ErrorNoReplica", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			buff = io.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			ttl  = 2 * time.Minute
			ca   = time.Now()
		)

		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		volID, err := s.CreateReplica(ctx, key, buff, ttl, ca)
		assert.EqualError(t, err, "can not store replicas")
		assert.Equal(t, "", volID)
	})
	t.Run("ErrorNoLocalVolumes", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			buff = io.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			ttl  = 2 * time.Minute
			ca   = time.Now()
		)

		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Local{}).AnyTimes()
		m.EXPECT().RemovedVolumeIDs().Return(nil).AnyTimes()

		s, err := storing.New(&config.Config{Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		volID, err := s.CreateReplica(ctx, key, buff, ttl, ca)
		assert.EqualError(t, err, "no local volumes to store replica")
		assert.Equal(t, "", volID)
	})
}

func TestUpdateFileReplica(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 4
			vids = []string{"1", "2"}
			vid  = "vid"
		)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		v.EXPECT().HasFile(gomock.Any(), key).Return(vid, true, nil)
		v.EXPECT().UpdateFileReplica(gomock.Any(), key, vids, rep).Return(nil)

		// It's AnyTimes as we have the config witha number of replicas
		// which activates the goroutines that also calls this
		m.EXPECT().LocalVolumes().Return([]volume.Local{v}).AnyTimes()

		// This is also because of the goroutine, it may call it or not
		v.EXPECT().NextReplica(gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()
		v.EXPECT().NextDeletion(gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()
		v.EXPECT().NextScrub(gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()
		m.EXPECT().RemovedVolumeIDs().Return(nil).AnyTimes()

		s, err := storing.New(&config.Config{Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.UpdateFileReplica(ctx, key, vids, rep)
		require.NoError(t, err)
	})
	t.Run("ErrorNoReplica", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 4
			vids = []string{"1", "2"}
		)

		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.UpdateFileReplica(ctx, key, vids, rep)
		assert.EqualError(t, err, "can not store replicas")
	})
	t.Run("ErrorNoLocalVolumes", func(t *testing.T) {
		var (
			key  = "testbucket/expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 4
			vids = []string{"1", "2"}
		)

		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Local{}).AnyTimes()
		m.EXPECT().RemovedVolumeIDs().Return(nil).AnyTimes()

		s, err := storing.New(&config.Config{Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.UpdateFileReplica(ctx, key, vids, rep)
		assert.EqualError(t, err, "no local volumes to update replica")
	})
}

func TestService_Drain(t *testing.T) {
	t.Run("ProxyMode", func(t *testing.T) {
		var (
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
		)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().SetDraining(true)
		m.EXPECT().LocalVolumes().Return([]volume.Local{})
		m.EXPECT().Leave()

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.Drain(ctx)
		require.NoError(t, err)
	})
	t.Run("Success", func(t *testing.T) {
		var (
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
		)
		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		// First call returns true (pending), second returns false (done)
		gomock.InOrder(
			m.EXPECT().SetDraining(true),
			v.EXPECT().PrepareForDrain(gomock.Any()).Return(nil),
			v.EXPECT().HasPendingReplicas(gomock.Any()).Return(true, nil),
			v.EXPECT().HasPendingReplicas(gomock.Any()).Return(false, nil),
			v.EXPECT().PurgeAllFiles(gomock.Any()).Return(nil),
		)

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})
		m.EXPECT().Leave()

		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.Drain(ctx)
		require.NoError(t, err)
	})
	t.Run("PrepareForDrainError", func(t *testing.T) {
		var (
			ctrl   = gomock.NewController(t)
			ctx    = context.Background()
			expErr = errors.New("prepare error")
		)
		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().SetDraining(true)
		v.EXPECT().PrepareForDrain(gomock.Any()).Return(expErr)

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})

		// Replica: -1 disables background loops so mock expectations stay deterministic.
		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.Drain(ctx)
		require.Error(t, err)
		assert.True(t, errors.Is(err, expErr))
	})
	t.Run("HasPendingReplicasError", func(t *testing.T) {
		var (
			ctrl   = gomock.NewController(t)
			ctx    = context.Background()
			expErr = errors.New("pending replicas error")
		)
		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		gomock.InOrder(
			m.EXPECT().SetDraining(true),
			v.EXPECT().PrepareForDrain(gomock.Any()).Return(nil),
			v.EXPECT().HasPendingReplicas(gomock.Any()).Return(false, expErr),
		)

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})

		// Replica: -1 disables background loops so mock expectations stay deterministic.
		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.Drain(ctx)
		require.Error(t, err)
		assert.True(t, errors.Is(err, expErr))
	})
	t.Run("PurgeAllFilesError", func(t *testing.T) {
		var (
			ctrl   = gomock.NewController(t)
			ctx    = context.Background()
			expErr = errors.New("purge error")
		)
		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		gomock.InOrder(
			m.EXPECT().SetDraining(true),
			v.EXPECT().PrepareForDrain(gomock.Any()).Return(nil),
			v.EXPECT().HasPendingReplicas(gomock.Any()).Return(false, nil),
			v.EXPECT().PurgeAllFiles(gomock.Any()).Return(expErr),
		)

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})

		// Replica: -1 disables background loops so mock expectations stay deterministic.
		s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		err = s.Drain(ctx)
		require.Error(t, err)
		assert.True(t, errors.Is(err, expErr))
	})
}

func TestService_CreateFileWhileDraining(t *testing.T) {
	var (
		ctrl = gomock.NewController(t)
		ctx  = context.Background()
		key  = "testbucket/key"
		buff = io.NopCloser(bytes.NewBufferString("content"))
		ttl  = time.Minute
		ca   = time.Now()
	)
	m := mock.NewMembership(ctrl)
	defer ctrl.Finish()

	// Put service into draining state via proxy-mode Drain
	m.EXPECT().SetDraining(true)
	m.EXPECT().LocalVolumes().Return([]volume.Local{})
	m.EXPECT().Leave()

	s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)

	err = s.Drain(ctx)
	require.NoError(t, err)

	// Now try to create a file — should be rejected
	err = s.CreateFile(ctx, key, buff, 2, ttl, ca)
	assert.EqualError(t, err, "node is draining")
}

func TestService_CreateReplicaWhileDraining(t *testing.T) {
	var (
		ctrl   = gomock.NewController(t)
		ctx    = context.Background()
		key    = "testbucket/key"
		reader = io.NopCloser(bytes.NewBufferString("content"))
		ttl    = time.Minute
		ca     = time.Now()
	)
	m := mock.NewMembership(ctrl)
	defer ctrl.Finish()

	// Put service into draining state via proxy-mode Drain
	m.EXPECT().SetDraining(true)
	m.EXPECT().LocalVolumes().Return([]volume.Local{})
	m.EXPECT().Leave()

	s, err := storing.New(&config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}, m, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)

	err = s.Drain(ctx)
	require.NoError(t, err)

	// Now try to create a replica — should be rejected
	_, err = s.CreateReplica(ctx, key, reader, ttl, ca)
	assert.EqualError(t, err, "node is draining")
}
