package storing

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/mock"
	httptransport "github.com/xescugc/rebost/storing/transport/http"
	"github.com/xescugc/rebost/volume"
)

func TestRunConsistencyCheck(t *testing.T) {
	t.Run("SkipsOwnedFiles", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		localVID := "local-vol"
		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		f := &file.File{
			Keys:      []string{"bucket/owned"},
			VolumeIDs: []string{localVID, "remote-1"},
			Replica:   3,
		}

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})
		v.EXPECT().AllFiles(gomock.Any()).Return([]*file.File{f}, nil)
		v.EXPECT().ID().Return(localVID).AnyTimes()
		// No GetNodeWithVolumeByID or PurgeFile expected.

		s.runConsistencyCheck()
	})

	t.Run("SkipsEmptyVolumeIDs", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		f := &file.File{
			Keys:      []string{"bucket/empty"},
			VolumeIDs: []string{},
			Replica:   3,
		}

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})
		v.EXPECT().AllFiles(gomock.Any()).Return([]*file.File{f}, nil)
		v.EXPECT().ID().Return("any-vol").AnyTimes()
		// No further calls expected.

		s.runConsistencyCheck()
	})

	t.Run("ChecksNonOwnedFile_StillListed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		localVID := "local-vol"
		ownerVID := "owner-vol"

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		remoteNode := mock.NewStoring(ctrl)
		h := httptransport.MakeHandler(remoteNode, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		defer server.Close()
		c, err := client.New(server.URL)
		if err != nil {
			t.Fatal(err)
		}

		f := &file.File{
			Keys:      []string{"bucket/nonowned"},
			VolumeIDs: []string{ownerVID, localVID},
			Replica:   2,
		}

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})
		v.EXPECT().AllFiles(gomock.Any()).Return([]*file.File{f}, nil)
		v.EXPECT().ID().Return(localVID).AnyTimes()
		m.EXPECT().GetNodeWithVolumeByID(ownerVID).Return(c, nil)
		// Owner still lists localVID.
		remoteNode.EXPECT().GetReplicaInfo(gomock.Any(), "bucket/nonowned").Return([]string{ownerVID, localVID}, 2, nil)
		// No PurgeFile expected.

		s.runConsistencyCheck()
	})

	t.Run("ChecksNonOwnedFile_Stale", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		localVID := "local-vol"
		ownerVID := "owner-vol"

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		remoteNode := mock.NewStoring(ctrl)
		h := httptransport.MakeHandler(remoteNode, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		defer server.Close()
		c, err := client.New(server.URL)
		if err != nil {
			t.Fatal(err)
		}

		f := &file.File{
			Keys:      []string{"bucket/stale"},
			VolumeIDs: []string{ownerVID, localVID},
			Replica:   2,
		}

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})
		v.EXPECT().AllFiles(gomock.Any()).Return([]*file.File{f}, nil)
		v.EXPECT().ID().Return(localVID).AnyTimes()
		m.EXPECT().GetNodeWithVolumeByID(ownerVID).Return(c, nil)
		// Owner no longer lists localVID.
		remoteNode.EXPECT().GetReplicaInfo(gomock.Any(), "bucket/stale").Return([]string{ownerVID}, 2, nil)
		v.EXPECT().PurgeFile(gomock.Any(), "bucket/stale").Return(nil)

		s.runConsistencyCheck()
	})
}

func TestCheckFileConsistency(t *testing.T) {
	t.Run("StillListed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		localVID := "local-vol"
		ownerVID := "owner-vol"

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		remoteNode := mock.NewStoring(ctrl)
		h := httptransport.MakeHandler(remoteNode, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		defer server.Close()
		c, err := client.New(server.URL)
		if err != nil {
			t.Fatal(err)
		}

		f := &file.File{
			Keys:      []string{"bucket/key"},
			VolumeIDs: []string{ownerVID, localVID},
		}

		v.EXPECT().ID().Return(localVID).AnyTimes()
		m.EXPECT().GetNodeWithVolumeByID(ownerVID).Return(c, nil)
		remoteNode.EXPECT().GetReplicaInfo(gomock.Any(), "bucket/key").Return([]string{ownerVID, localVID}, 2, nil)
		// No PurgeFile expected.

		s.checkFileConsistency(v, f)
	})

	t.Run("NoLongerListed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		localVID := "local-vol"
		ownerVID := "owner-vol"

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		remoteNode := mock.NewStoring(ctrl)
		h := httptransport.MakeHandler(remoteNode, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		defer server.Close()
		c, err := client.New(server.URL)
		if err != nil {
			t.Fatal(err)
		}

		f := &file.File{
			Keys:      []string{"bucket/key"},
			VolumeIDs: []string{ownerVID, localVID},
		}

		v.EXPECT().ID().Return(localVID).AnyTimes()
		m.EXPECT().GetNodeWithVolumeByID(ownerVID).Return(c, nil)
		remoteNode.EXPECT().GetReplicaInfo(gomock.Any(), "bucket/key").Return([]string{ownerVID}, 2, nil)
		v.EXPECT().PurgeFile(gomock.Any(), "bucket/key").Return(nil)

		s.checkFileConsistency(v, f)
	})

	t.Run("OwnerNodeNotFound", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		localVID := "local-vol"
		ownerVID := "owner-vol"

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		f := &file.File{
			Keys:      []string{"bucket/key"},
			VolumeIDs: []string{ownerVID, localVID},
		}

		v.EXPECT().ID().Return(localVID).AnyTimes()
		m.EXPECT().GetNodeWithVolumeByID(ownerVID).Return(nil, errors.New("node not found"))
		// All holders unreachable — no purge.

		s.checkFileConsistency(v, f)
	})

	t.Run("OwnerHTTPUnreachable", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		localVID := "local-vol"
		ownerVID := "owner-vol"

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		// Start and immediately close the server to simulate unreachable host.
		server := httptest.NewServer(nil)
		server.Close()
		c, err := client.New(server.URL)
		if err != nil {
			t.Fatal(err)
		}

		f := &file.File{
			Keys:      []string{"bucket/key"},
			VolumeIDs: []string{ownerVID, localVID},
		}

		v.EXPECT().ID().Return(localVID).AnyTimes()
		m.EXPECT().GetNodeWithVolumeByID(ownerVID).Return(c, nil)
		// CheckReplica call will fail — no purge.

		s.checkFileConsistency(v, f)
	})

	t.Run("SelfVolumeSkipped", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		localVID := "local-vol"
		ownerVID := "owner-vol"

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		remoteNode := mock.NewStoring(ctrl)
		h := httptransport.MakeHandler(remoteNode, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		defer server.Close()
		c, err := client.New(server.URL)
		if err != nil {
			t.Fatal(err)
		}

		// localVID listed first, ownerVID second.
		f := &file.File{
			Keys:      []string{"bucket/key"},
			VolumeIDs: []string{localVID, ownerVID},
		}

		v.EXPECT().ID().Return(localVID).AnyTimes()
		// Self (localVID) is skipped; ownerVID is the one queried.
		m.EXPECT().GetNodeWithVolumeByID(ownerVID).Return(c, nil)
		remoteNode.EXPECT().GetReplicaInfo(gomock.Any(), "bucket/key").Return([]string{localVID, ownerVID}, 2, nil)
		// Still listed — no purge.

		s.checkFileConsistency(v, f)
	})
}
