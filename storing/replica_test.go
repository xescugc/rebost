package storing

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/deletion"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/replica"
	httptransport "github.com/xescugc/rebost/storing/transport/http"
	"github.com/xescugc/rebost/volume"
)

func newTestService(t *testing.T, m *mock.Membership) *service {
	t.Helper()
	cache, err := lru.NewARC[string, string](config.DefaultCacheSize)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Replica: -1, Cache: config.Cache{Size: config.DefaultCacheSize}}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return &service{
		members: m,
		cfg:     cfg,
		cache:   cache,
		ctx:     ctx,
		cancel:  cancel,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestProcessNextReplica(t *testing.T) {
	t.Run("QueueEmpty", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		v.EXPECT().NextReplica(gomock.Any()).Return(nil, errors.New("not found"))

		assert.False(t, s.processNextReplica(v))
	})

	t.Run("StaleJobFileDeleted", func(t *testing.T) {
		// Scenario from issue #100: file was deleted but replica job remains.
		// processNextReplica must remove the stale job and return true.
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		key := "somekey"
		rp := &replica.Replica{Key: key, Count: 1, OriginalCount: 3}

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		v.EXPECT().NextReplica(gomock.Any()).Return(rp, nil)
		v.EXPECT().HasFile(gomock.Any(), key).Return("", false, nil)
		v.EXPECT().DeleteReplica(gomock.Any(), rp).Return(nil)

		assert.True(t, s.processNextReplica(v))
	})

	t.Run("NoAvailableNodes", func(t *testing.T) {
		// File exists locally but no peers are missing the replica.
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		key := "somekey"
		vid := "localvid"
		rp := &replica.Replica{Key: key, VolumeIDs: []string{vid}, Count: 1, OriginalCount: 2}

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		v.EXPECT().NextReplica(gomock.Any()).Return(rp, nil)
		v.EXPECT().HasFile(gomock.Any(), key).Return(vid, true, nil)
		m.EXPECT().NodesWithoutVolumeIDs(rp.VolumeIDs).Return(nil)

		assert.False(t, s.processNextReplica(v))
	})

	t.Run("Success", func(t *testing.T) {
		// Full happy path: file exists locally, one peer needs a copy.
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		var (
			key           = "testbucket/somekey"
			localVID      = "localvid"
			newVID        = "newvid"
			originalCount = 2
			content       = "filecontent"
			ttl           = 2 * time.Minute
			ca            = time.Now()
		)

		rp := &replica.Replica{
			Key:           key,
			VolumeIDs:     []string{localVID},
			Count:         1,
			OriginalCount: originalCount,
			TTL:           ttl,
			CreatedAt:     ca,
		}

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		// Set up a real HTTP test server acting as the remote node.
		remoteNode := mock.NewStoring(ctrl)
		h := httptransport.MakeHandler(remoteNode, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		defer server.Close()
		c, err := client.New(server.URL)
		if err != nil {
			t.Fatal(err)
		}

		v.EXPECT().NextReplica(gomock.Any()).Return(rp, nil)
		v.EXPECT().HasFile(gomock.Any(), key).Return(localVID, true, nil)
		m.EXPECT().NodesWithoutVolumeIDs(rp.VolumeIDs).Return([]*client.Client{c})

		// Remote node does not have the file.
		remoteNode.EXPECT().HasFile(gomock.Any(), key).Return("", false, nil)

		v.EXPECT().GetFile(gomock.Any(), key).Return(io.NopCloser(bytes.NewBufferString(content)), int64(-1), nil)

		// Remote node accepts the replica and returns its volume ID.
		// gomock.Any() for ca: time.Time loses its monotonic component over HTTP.
		remoteNode.EXPECT().CreateReplica(gomock.Any(), key, gomock.Any(), ttl, gomock.Any()).Return(newVID, nil)

		// After replication, peers are notified: newVID's node gets UpdateFileReplica.
		// localVID is skipped (it's the local volume), newVID gets notified.
		v.EXPECT().ID().Return(localVID).AnyTimes()
		m.EXPECT().GetNodeWithVolumeByID(newVID).Return(c, nil)
		remoteNode.EXPECT().UpdateFileReplica(gomock.Any(), key, []string{localVID, newVID}, originalCount).Return(nil)

		v.EXPECT().UpdateReplica(gomock.Any(), rp, newVID).Return(nil)

		assert.True(t, s.processNextReplica(v))
	})
}

func TestProcessNextDeletion(t *testing.T) {
	t.Run("QueueEmpty", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		v.EXPECT().NextDeletion(gomock.Any()).Return(nil, errors.New("not found"))

		assert.False(t, s.processNextDeletion(v))
	})

	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		key := "testbucket/todelete"
		remoteVID := "remote-vid-1"

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		// Set up a real HTTP test server acting as the remote node.
		remoteNode := mock.NewStoring(ctrl)
		h := httptransport.MakeHandler(remoteNode, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		defer server.Close()
		c, err := client.New(server.URL)
		if err != nil {
			t.Fatal(err)
		}

		d := &deletion.Deletion{Key: key, VolumeIDs: []string{remoteVID}}

		v.EXPECT().NextDeletion(gomock.Any()).Return(d, nil)
		m.EXPECT().GetNodeWithVolumeByID(remoteVID).Return(c, nil)
		remoteNode.EXPECT().DeleteFile(gomock.Any(), key).Return(nil)
		v.EXPECT().DeleteDeletion(gomock.Any(), d).Return(nil)

		assert.True(t, s.processNextDeletion(v))
	})

	t.Run("NodeGone", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		key := "testbucket/todelete"
		remoteVID := "gone-vid"

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		d := &deletion.Deletion{Key: key, VolumeIDs: []string{remoteVID}}

		v.EXPECT().NextDeletion(gomock.Any()).Return(d, nil)
		m.EXPECT().GetNodeWithVolumeByID(remoteVID).Return(nil, errors.New("node not found"))
		v.EXPECT().DeleteDeletion(gomock.Any(), d).Return(nil)

		assert.True(t, s.processNextDeletion(v))
	})
}

func TestProcessRemovedVolumeIDs(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		// No calls expected at all.
		s.processRemovedVolumeIDs([]string{})
	})

	t.Run("SingleVolume", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		vid := "removed-vid"
		lv := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		m.EXPECT().LocalVolumes().Return([]volume.Local{lv})
		lv.EXPECT().SynchronizeReplicas(gomock.Any(), vid).Return(nil)

		s.processRemovedVolumeIDs([]string{vid})
	})

	t.Run("MultipleLocalVolumes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		vid := "removed-vid"
		lv1 := mock.NewVolumeLocal(ctrl)
		lv2 := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		m.EXPECT().LocalVolumes().Return([]volume.Local{lv1, lv2})
		lv1.EXPECT().SynchronizeReplicas(gomock.Any(), vid).Return(nil)
		lv2.EXPECT().SynchronizeReplicas(gomock.Any(), vid).Return(nil)

		s.processRemovedVolumeIDs([]string{vid})
	})
}
