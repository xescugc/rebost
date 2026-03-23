package membership_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"io"
	"log/slog"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/membership"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/state"
	"github.com/xescugc/rebost/storing"
	httptransport "github.com/xescugc/rebost/storing/transport/http"
	"github.com/xescugc/rebost/util"
	"github.com/xescugc/rebost/volume"
)

func TestNodesWithCapacity(t *testing.T) {
	t.Run("ReturnsNodeWithEnoughCapacity", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Local node (m): just needs a volume with an ID
		v := mock.NewVolumeLocal(ctrl)
		v.EXPECT().ID().Return("local-id").Times(2)
		v.EXPECT().GetState(context.Background()).Return(&state.State{}, nil).AnyTimes()

		// Remote node A (m2): 10 bytes free (100 total, 90 used)
		stateA := &state.State{VolumeTotalSize: 100, VolumeUsedSize: 90}
		v2 := mock.NewVolumeLocal(ctrl)
		v2.EXPECT().ID().Return("remote-id-a").Times(2)
		v2.EXPECT().GetState(context.Background()).Return(stateA, nil).AnyTimes()

		p2, err := util.FreePort()
		require.NoError(t, err)
		cfg2 := &config.Config{Name: "nwc2", Replica: -1, Memberlist: config.Memberlist{Port: p2}, Cache: config.Cache{Size: config.DefaultCacheSize}}
		m2, err := membership.New(cfg2, []volume.Local{v2}, "", slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		s2, err := storing.New(cfg2, m2, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)
		server2 := httptest.NewServer(httptransport.MakeHandler(s2, &config.Config{}))
		defer server2.Close()

		p3, err := util.FreePort()
		require.NoError(t, err)
		cfg := &config.Config{Name: "nwc", Memberlist: config.Memberlist{Port: p3}, Cache: config.Cache{Size: config.DefaultCacheSize}}
		m, err := membership.New(cfg, []volume.Local{v}, server2.URL, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)
		require.Len(t, m.Nodes(), 1)

		// Wait for gossip state to propagate
		require.Eventually(t, func() bool {
			ns := m.NodesWithCapacity(5)
			return len(ns) == 1
		}, 5*time.Second, 100*time.Millisecond, "expected node A to appear with capacity for 5 bytes")

		// Node A has 10 bytes free; asking for 20 should return empty
		assert.Empty(t, m.NodesWithCapacity(20), "no node should have 20 bytes free")
	})

	t.Run("ExcludesFullNode", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Local node (m): just needs a volume with an ID
		v := mock.NewVolumeLocal(ctrl)
		v.EXPECT().ID().Return("local-id").AnyTimes()
		v.EXPECT().GetState(context.Background()).Return(&state.State{}, nil).AnyTimes()

		// Remote node A: 10 bytes free (100 total, 90 used)
		stateA := &state.State{VolumeTotalSize: 100, VolumeUsedSize: 90}
		v2 := mock.NewVolumeLocal(ctrl)
		v2.EXPECT().ID().Return("remote-id-a").AnyTimes()
		v2.EXPECT().GetState(context.Background()).Return(stateA, nil).AnyTimes()

		p2, err := util.FreePort()
		require.NoError(t, err)
		cfg2 := &config.Config{Name: "efn-a", Replica: -1, Memberlist: config.Memberlist{Port: p2}, Cache: config.Cache{Size: config.DefaultCacheSize}}
		m2, err := membership.New(cfg2, []volume.Local{v2}, "", slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		s2, err := storing.New(cfg2, m2, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)
		server2 := httptest.NewServer(httptransport.MakeHandler(s2, &config.Config{}))
		defer server2.Close()

		// Remote node B: completely full (100 total, 100 used)
		stateB := &state.State{VolumeTotalSize: 100, VolumeUsedSize: 100}
		v3 := mock.NewVolumeLocal(ctrl)
		v3.EXPECT().ID().Return("remote-id-b").AnyTimes()
		v3.EXPECT().GetState(context.Background()).Return(stateB, nil).AnyTimes()

		p3, err := util.FreePort()
		require.NoError(t, err)
		cfg3 := &config.Config{Name: "efn-b", Replica: -1, Memberlist: config.Memberlist{Port: p3}, Cache: config.Cache{Size: config.DefaultCacheSize}}
		// Node B joins the cluster via node A
		m3, err := membership.New(cfg3, []volume.Local{v3}, server2.URL, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		s3, err := storing.New(cfg3, m3, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)
		server3 := httptest.NewServer(httptransport.MakeHandler(s3, &config.Config{}))
		defer server3.Close()

		// Local node joins via node A; gossip will propagate node B as well
		p4, err := util.FreePort()
		require.NoError(t, err)
		cfg := &config.Config{Name: "efn-local", Memberlist: config.Memberlist{Port: p4}, Cache: config.Cache{Size: config.DefaultCacheSize}}
		m, err := membership.New(cfg, []volume.Local{v}, server2.URL, slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)

		// Wait until both remote nodes are visible
		require.Eventually(t, func() bool {
			return len(m.Nodes()) == 2
		}, 5*time.Second, 100*time.Millisecond, "expected both remote nodes to appear")

		// Wait for gossip state to propagate so NodesWithCapacity can evaluate correctly
		require.Eventually(t, func() bool {
			ns := m.NodesWithCapacity(1)
			return len(ns) == 1
		}, 5*time.Second, 100*time.Millisecond, "expected only node A to have capacity for 1 byte")

		ns := m.NodesWithCapacity(1)
		assert.Len(t, ns, 1, "only node A should be returned")
		assert.NotContains(t, m.NodesWithCapacity(1), m3, "node B (full) must not be included")
	})
}

func TestDrainingFiltering(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Local node (m): observer
	v := mock.NewVolumeLocal(ctrl)
	v.EXPECT().ID().Return("local-drain-id").AnyTimes()
	v.EXPECT().GetState(context.Background()).Return(&state.State{}, nil).AnyTimes()

	// Remote node (m2): the one that drains
	v2 := mock.NewVolumeLocal(ctrl)
	v2.EXPECT().ID().Return("remote-drain-id").AnyTimes()
	v2.EXPECT().GetState(context.Background()).Return(&state.State{}, nil).AnyTimes()

	p2, err := util.FreePort()
	require.NoError(t, err)
	cfg2 := &config.Config{Name: "drain-m2", Replica: -1, Memberlist: config.Memberlist{Port: p2}, Cache: config.Cache{Size: config.DefaultCacheSize}}
	m2, err := membership.New(cfg2, []volume.Local{v2}, "", slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	defer m2.Leave()

	s2, err := storing.New(cfg2, m2, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	server2 := httptest.NewServer(httptransport.MakeHandler(s2, &config.Config{}))
	defer server2.Close()

	p3, err := util.FreePort()
	require.NoError(t, err)
	cfg := &config.Config{Name: "drain-m", Memberlist: config.Memberlist{Port: p3}, Cache: config.Cache{Size: config.DefaultCacheSize}}
	m, err := membership.New(cfg, []volume.Local{v}, server2.URL, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	defer m.Leave()

	// Wait for m2 to appear in m's Nodes()
	require.Eventually(t, func() bool {
		return len(m.Nodes()) == 1
	}, 5*time.Second, 100*time.Millisecond, "expected m2 to appear in Nodes()")

	// Set m2 to draining; it should disappear from m's Nodes() and NodesWithoutVolumeIDs()
	m2.SetDraining(true)
	require.Eventually(t, func() bool {
		return len(m.Nodes()) == 0
	}, 5*time.Second, 100*time.Millisecond, "expected m2 to be hidden from Nodes() while draining")
	assert.Empty(t, m.NodesWithoutVolumeIDs([]string{}), "draining node must not appear in NodesWithoutVolumeIDs()")

	// Unset draining; m2 should reappear
	m2.SetDraining(false)
	require.Eventually(t, func() bool {
		return len(m.Nodes()) == 1
	}, 5*time.Second, 100*time.Millisecond, "expected m2 to reappear in Nodes() after draining cleared")
}

func TestVolumes(t *testing.T) {
	t.Run("WithoutNodes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		v := mock.NewVolumeLocal(ctrl)
		v.EXPECT().ID().Return("id")

		p, err := util.FreePort()
		require.NoError(t, err)

		m, err := membership.New(&config.Config{Memberlist: config.Memberlist{Port: p}, Cache: config.Cache{Size: config.DefaultCacheSize}}, []volume.Local{v}, "", slog.New(slog.NewTextHandler(io.Discard, nil)))
		require.NoError(t, err)
		assert.Len(t, m.Nodes(), 0)
		assert.Equal(t, []volume.Local{v}, m.LocalVolumes())
		assert.Equal(t, []string{}, m.RemovedVolumeIDs())
	})
	t.Run("WithNodes", func(t *testing.T) {
		t.Run("Add", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			v := mock.NewVolumeLocal(ctrl)
			v.EXPECT().ID().Return("id").Times(2)
			v.EXPECT().GetState(context.Background()).Return(&state.State{}, nil)

			v2 := mock.NewVolumeLocal(ctrl)
			v2.EXPECT().ID().Return("id").Times(2)
			v2.EXPECT().GetState(context.Background()).Return(&state.State{}, nil)
			p2, err := util.FreePort()
			require.NoError(t, err)
			cfg2 := &config.Config{Name: "am2", Replica: -1, Memberlist: config.Memberlist{Port: p2}, Cache: config.Cache{Size: config.DefaultCacheSize}}
			m2, err := membership.New(cfg2, []volume.Local{v2}, "", slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)

			s, err := storing.New(cfg2, m2, slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			server := httptest.NewServer(httptransport.MakeHandler(s, &config.Config{}))
			defer server.Close()

			p3, err := util.FreePort()
			require.NoError(t, err)
			cfg := &config.Config{Name: "am", Memberlist: config.Memberlist{Port: p3}, Cache: config.Cache{Size: config.DefaultCacheSize}}
			m, err := membership.New(cfg, []volume.Local{v}, server.URL, slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			assert.Len(t, m.Nodes(), 1)
			assert.Equal(t, []volume.Local{v}, m.LocalVolumes())
			assert.Equal(t, []string{}, m.RemovedVolumeIDs())
		})
		t.Run("Remove", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			v := mock.NewVolumeLocal(ctrl)
			v.EXPECT().ID().Return("id").Times(2)
			v.EXPECT().GetState(context.Background()).Return(&state.State{}, nil)

			v2 := mock.NewVolumeLocal(ctrl)
			v2.EXPECT().ID().Return("id2").Times(2)
			v2.EXPECT().GetState(context.Background()).Return(&state.State{}, nil)
			p2, err := util.FreePort()
			require.NoError(t, err)
			cfg2 := &config.Config{Name: "rm2", Replica: -1, Memberlist: config.Memberlist{Port: p2}, Cache: config.Cache{Size: config.DefaultCacheSize}}
			m2, err := membership.New(cfg2, []volume.Local{v2}, "", slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			s, err := storing.New(cfg2, m2, slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			server := httptest.NewServer(httptransport.MakeHandler(s, &config.Config{}))
			defer server.Close()

			p3, err := util.FreePort()
			require.NoError(t, err)
			cfg := &config.Config{Name: "rm", Memberlist: config.Memberlist{Port: p3}, Cache: config.Cache{Size: config.DefaultCacheSize}}
			m, err := membership.New(cfg, []volume.Local{v}, server.URL, slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			assert.Len(t, m.Nodes(), 1)
			assert.Equal(t, []volume.Local{v}, m.LocalVolumes())

			m2.Leave()
			require.Eventually(t, func() bool { return len(m.Nodes()) == 0 }, time.Second, 10*time.Millisecond)
			assert.Equal(t, []volume.Local{v}, m.LocalVolumes())
			assert.Equal(t, []string{"id2"}, m.RemovedVolumeIDs())
			assert.Equal(t, []string{}, m.RemovedVolumeIDs())
		})
		t.Run("RemoveWithVolumeDowntime", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			v := mock.NewVolumeLocal(ctrl)
			v.EXPECT().ID().Return("id").Times(2)
			v.EXPECT().GetState(context.Background()).Return(&state.State{}, nil)

			v2 := mock.NewVolumeLocal(ctrl)
			v2.EXPECT().ID().Return("id2").Times(2)
			v2.EXPECT().GetState(context.Background()).Return(&state.State{}, nil)

			p2, err := util.FreePort()
			require.NoError(t, err)
			cfg2 := &config.Config{Name: "rm2", Replica: -1, VolumeDowntime: 2 * time.Second, Memberlist: config.Memberlist{Port: p2}, Cache: config.Cache{Size: config.DefaultCacheSize}}
			m2, err := membership.New(cfg2, []volume.Local{v2}, "", slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			s, err := storing.New(cfg2, m2, slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			server := httptest.NewServer(httptransport.MakeHandler(s, &config.Config{}))
			defer server.Close()

			p3, err := util.FreePort()
			require.NoError(t, err)
			cfg := &config.Config{Name: "rm", VolumeDowntime: time.Second, Memberlist: config.Memberlist{Port: p3}, Cache: config.Cache{Size: config.DefaultCacheSize}}
			m, err := membership.New(cfg, []volume.Local{v}, server.URL, slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			assert.Len(t, m.Nodes(), 1)
			assert.Equal(t, []volume.Local{v}, m.LocalVolumes())

			m2.Leave()
			require.Eventually(t, func() bool { return len(m.Nodes()) == 0 }, time.Second, 10*time.Millisecond)
			assert.Equal(t, []volume.Local{v}, m.LocalVolumes())
			assert.Equal(t, []string{}, m.RemovedVolumeIDs())
			assert.Equal(t, []string{}, m.RemovedVolumeIDs())

			time.Sleep(time.Second)
			assert.Equal(t, []string{"id2"}, m.RemovedVolumeIDs())
		})
	})
}
