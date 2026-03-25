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
	"github.com/xescugc/rebost/client"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/scrub"
	httptransport "github.com/xescugc/rebost/storing/transport/http"
	"github.com/xescugc/rebost/volume"
)

func TestProcessNextScrub(t *testing.T) {
	t.Run("QueueEmpty", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		v.EXPECT().NextScrub(gomock.Any()).Return(nil, errors.New("not found"))

		assert.False(t, s.processNextScrub(v))
	})

	t.Run("SuccessfulRepair", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		const (
			key       = "bucket/corrupt"
			sig       = "abc123"
			remoteVID = "remote-vol-1"
			content   = "correct file content"
		)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		// Spin up a real HTTP test server acting as the remote node.
		remoteStore := mock.NewStoring(ctrl)
		h := httptransport.MakeHandler(remoteStore, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		defer server.Close()
		c, err := client.New(server.URL)
		if err != nil {
			t.Fatal(err)
		}

		sc := &scrub.Scrub{
			ScrubID:   []byte("1"),
			Key:       key,
			Signature: sig,
			VolumeIDs: []string{remoteVID},
		}

		stat := &volume.FileStat{Size: int64(len(content))}

		v.EXPECT().NextScrub(gomock.Any()).Return(sc, nil)
		m.EXPECT().GetNodeWithVolumeByID(remoteVID).Return(c, nil)
		// HEAD /{key}: HasFile + StatFile
		remoteStore.EXPECT().HasFile(gomock.Any(), key).Return(remoteVID, true, nil)
		remoteStore.EXPECT().StatFile(gomock.Any(), key).Return(stat, nil)
		// GET /{key}: StatFile + GetFile
		remoteStore.EXPECT().StatFile(gomock.Any(), key).Return(stat, nil)
		remoteStore.EXPECT().GetFile(gomock.Any(), key).Return(
			io.NopCloser(bytes.NewBufferString(content)), int64(len(content)), nil,
		)
		v.EXPECT().OverwriteFileContent(gomock.Any(), sig, gomock.Any()).Return(nil)
		v.EXPECT().DeleteScrub(gomock.Any(), sc).Return(nil)

		assert.True(t, s.processNextScrub(v))
	})

	t.Run("NoRemoteReplicas", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		sc := &scrub.Scrub{
			ScrubID:   []byte("1"),
			Key:       "bucket/file",
			Signature: "abc123",
			VolumeIDs: []string{},
		}

		v.EXPECT().NextScrub(gomock.Any()).Return(sc, nil)
		v.EXPECT().DeleteScrub(gomock.Any(), sc).Return(nil)

		assert.True(t, s.processNextScrub(v))
	})

	t.Run("AllReplicasUnreachable", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		sc := &scrub.Scrub{
			ScrubID:   []byte("1"),
			Key:       "bucket/file",
			Signature: "abc123",
			VolumeIDs: []string{"gone-vol"},
		}

		v.EXPECT().NextScrub(gomock.Any()).Return(sc, nil)
		m.EXPECT().GetNodeWithVolumeByID("gone-vol").Return(nil, errors.New("node not found"))
		v.EXPECT().DeleteScrub(gomock.Any(), sc).Return(nil)

		assert.True(t, s.processNextScrub(v))
	})

	t.Run("FileDeletedBeforeRepair", func(t *testing.T) {
		// Remote no longer has the file. HasFile returns false → skip, delete job.
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		const (
			key       = "bucket/gone"
			sig       = "abc123"
			remoteVID = "remote-vol-1"
		)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		s := newTestService(t, m)

		remoteStore := mock.NewStoring(ctrl)
		h := httptransport.MakeHandler(remoteStore, &config.Config{}, func() bool { return true })
		server := httptest.NewServer(h)
		defer server.Close()
		c, err := client.New(server.URL)
		if err != nil {
			t.Fatal(err)
		}

		sc := &scrub.Scrub{
			ScrubID:   []byte("1"),
			Key:       key,
			Signature: sig,
			VolumeIDs: []string{remoteVID},
		}

		v.EXPECT().NextScrub(gomock.Any()).Return(sc, nil)
		m.EXPECT().GetNodeWithVolumeByID(remoteVID).Return(c, nil)
		// HEAD /{key}: HasFile returns not-found (ok=false) → 404
		remoteStore.EXPECT().HasFile(gomock.Any(), key).Return("", false, nil)
		v.EXPECT().DeleteScrub(gomock.Any(), sc).Return(nil)

		assert.True(t, s.processNextScrub(v))
	})
}

// TestProcessNextScrub_UsesServiceContext verifies that the service context
// is passed to NextScrub.
func TestProcessNextScrub_UsesServiceContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	v := mock.NewVolumeLocal(ctrl)
	m := mock.NewMembership(ctrl)
	s := newTestService(t, m)

	var capturedCtx context.Context
	v.EXPECT().NextScrub(gomock.Any()).DoAndReturn(func(ctx context.Context) (*scrub.Scrub, error) {
		capturedCtx = ctx
		return nil, errors.New("not found")
	})

	s.processNextScrub(v)
	assert.Equal(t, s.ctx, capturedCtx)
}
