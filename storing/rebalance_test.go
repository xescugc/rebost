package storing

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/mock"
	httptransport "github.com/xescugc/rebost/storing/transport/http"
)

func TestRebalanceVolume(t *testing.T) {
	t.Run("SkipsFilesWhereNotOwner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		otherVID := "other-vid"
		localVID := "local-vid"

		files := []*file.File{
			{Keys: []string{"key1"}, VolumeIDs: []string{otherVID}},
			{Keys: []string{"key2"}, VolumeIDs: []string{otherVID, localVID}},
		}

		v.EXPECT().AllFiles(gomock.Any()).Return(files, nil)
		v.EXPECT().ID().Return(localVID).AnyTimes()
		m.EXPECT().AllVolumeIDs().Return([]string{localVID, otherVID})

		s.rebalanceVolume(v)
	})

	t.Run("SkipsWhenStillHRWWinner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		localVID := "local-vid"

		files := []*file.File{
			{Keys: []string{"somekey"}, VolumeIDs: []string{localVID}},
		}

		v.EXPECT().AllFiles(gomock.Any()).Return(files, nil)
		v.EXPECT().ID().Return(localVID).AnyTimes()
		// Only one vid in the cluster: localVID is always the winner.
		m.EXPECT().AllVolumeIDs().Return([]string{localVID})

		// transferOwnership must NOT be called — no GetNodeWithVolumeByID.
		s.rebalanceVolume(v)
	})

	t.Run("TransfersWhenNewWinnerExists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		const (
			localVID = "local-vid"
			otherVID = "other-vid"
			key      = "bucket/rebalance-test-key"
		)

		// Determine the HRW winner at test time.
		ranked := rankVolumeIDs(key, []string{localVID, otherVID})
		require.Equal(t, otherVID, ranked[0],
			"test requires otherVID to be HRW winner; adjust key or vids if this assertion fails")

		winnerVID := ranked[0]

		// Set up real HTTP test server for the winner node.
		winnerStore := mock.NewStoring(ctrl)
		winnerHandler := httptransport.MakeHandler(winnerStore, &config.Config{}, func() bool { return true })
		winnerServer := httptest.NewServer(winnerHandler)
		defer winnerServer.Close()
		winnerClient, err := client.New(winnerServer.URL)
		require.NoError(t, err)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		files := []*file.File{
			{Keys: []string{key}, VolumeIDs: []string{localVID}},
		}

		v.EXPECT().AllFiles(gomock.Any()).Return(files, nil)
		v.EXPECT().ID().Return(localVID).AnyTimes()
		m.EXPECT().AllVolumeIDs().Return([]string{localVID, otherVID})

		// transferOwnership path:
		m.EXPECT().GetNodeWithVolumeByID(winnerVID).Return(winnerClient, nil).Times(2)
		v.EXPECT().GetFile(gomock.Any(), key).Return(io.NopCloser(bytes.NewBufferString("content")), int64(-1), nil)
		winnerStore.EXPECT().CreateReplica(gomock.Any(), key, gomock.Any(), gomock.Any(), gomock.Any()).Return(winnerVID, nil)
		winnerStore.EXPECT().UpdateFileReplica(gomock.Any(), key, gomock.Any(), gomock.Any()).Return(nil)
		v.EXPECT().DeleteFile(gomock.Any(), key).Return(nil)

		s.rebalanceVolume(v)
	})
}

func TestTransferOwnership(t *testing.T) {
	t.Run("WinnerNodeNotFound", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		f := &file.File{
			Keys:      []string{"testkey"},
			VolumeIDs: []string{"local-vid"},
		}

		m.EXPECT().GetNodeWithVolumeByID("winner-vid").Return(nil, errors.New("not found"))

		s.transferOwnership(v, f, "winner-vid")
	})

	t.Run("GetFileFails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		winnerStore := mock.NewStoring(ctrl)
		h := httptransport.MakeHandler(winnerStore, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		defer server.Close()
		c, err := client.New(server.URL)
		require.NoError(t, err)

		f := &file.File{
			Keys:      []string{"testkey"},
			VolumeIDs: []string{"local-vid"},
		}

		m.EXPECT().GetNodeWithVolumeByID("winner-vid").Return(c, nil)
		v.EXPECT().GetFile(gomock.Any(), "testkey").Return(nil, int64(0), errors.New("disk error"))

		s.transferOwnership(v, f, "winner-vid")
	})

	t.Run("CreateReplicaFails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		winnerStore := mock.NewStoring(ctrl)
		h := httptransport.MakeHandler(winnerStore, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		defer server.Close()
		c, err := client.New(server.URL)
		require.NoError(t, err)

		f := &file.File{
			Keys:      []string{"testkey"},
			VolumeIDs: []string{"local-vid"},
		}

		m.EXPECT().GetNodeWithVolumeByID("winner-vid").Return(c, nil)
		v.EXPECT().GetFile(gomock.Any(), "testkey").Return(io.NopCloser(bytes.NewBufferString("content")), int64(-1), nil)
		winnerStore.EXPECT().CreateReplica(gomock.Any(), "testkey", gomock.Any(), gomock.Any(), gomock.Any()).Return("", errors.New("volume full"))

		// UpdateFileReplica must NOT be called after CreateReplica failure.
		s.transferOwnership(v, f, "winner-vid")
	})

	t.Run("SuccessWinnerAtIndex0", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		const (
			localVID   = "local-vid"
			winnerVID  = "winner-vid"
			replicaVID = "replica-vid"
			key        = "bucket/testkey"
		)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		// Set up winner HTTP test server.
		winnerStore := mock.NewStoring(ctrl)
		winnerH := httptransport.MakeHandler(winnerStore, &config.Config{}, func() bool { return true })
		winnerServer := httptest.NewServer(winnerH)
		defer winnerServer.Close()
		winnerClient, err := client.New(winnerServer.URL)
		require.NoError(t, err)

		// Set up replica HTTP test server.
		replicaStore := mock.NewStoring(ctrl)
		replicaH := httptransport.MakeHandler(replicaStore, &config.Config{}, func() bool { return true })
		replicaServer := httptest.NewServer(replicaH)
		defer replicaServer.Close()
		replicaClient, err := client.New(replicaServer.URL)
		require.NoError(t, err)

		f := &file.File{
			Keys:      []string{key},
			VolumeIDs: []string{localVID, replicaVID},
			Replica:   2,
		}

		// GetNodeWithVolumeByID(winnerVID) is called twice:
		//   1. To resolve the winner node for CreateReplica.
		//   2. In the UpdateFileReplica notification loop (winnerVID is in newVolumeIDs).
		m.EXPECT().GetNodeWithVolumeByID(winnerVID).Return(winnerClient, nil).Times(2)
		m.EXPECT().GetNodeWithVolumeByID(replicaVID).Return(replicaClient, nil)

		v.EXPECT().ID().Return(localVID).AnyTimes()
		v.EXPECT().GetFile(gomock.Any(), key).Return(io.NopCloser(bytes.NewBufferString("content")), int64(-1), nil)
		v.EXPECT().DeleteFile(gomock.Any(), key).Return(nil)

		winnerStore.EXPECT().CreateReplica(gomock.Any(), key, gomock.Any(), gomock.Any(), gomock.Any()).Return(winnerVID, nil)

		var capturedVolumeIDs []string
		winnerStore.EXPECT().UpdateFileReplica(gomock.Any(), key, gomock.Any(), 2).DoAndReturn(
			func(_ context.Context, _ string, vids []string, _ int) error {
				capturedVolumeIDs = make([]string, len(vids))
				copy(capturedVolumeIDs, vids)
				return nil
			},
		)
		replicaStore.EXPECT().UpdateFileReplica(gomock.Any(), key, gomock.Any(), 2).Return(nil)

		s.transferOwnership(v, f, winnerVID)

		// winnerVID must be at index 0.
		require.NotEmpty(t, capturedVolumeIDs)
		assert.Equal(t, winnerVID, capturedVolumeIDs[0], "winnerVID must be first in newVolumeIDs")

		// localVID must be excluded.
		assert.NotContains(t, capturedVolumeIDs, localVID, "old owner must not appear in newVolumeIDs")

		// replicaVID must still be present.
		assert.Contains(t, capturedVolumeIDs, replicaVID)
	})
}
