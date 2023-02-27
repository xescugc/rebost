package storing_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/storing"
	"github.com/xescugc/rebost/volume"
)

func TestCreateFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			key  = "expectedkey"
			buff = io.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 2
		)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		v.EXPECT().CreateFile(gomock.Any(), key, buff, rep).Return(nil)

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})

		s := storing.New(&config.Config{Replica: -1}, m, kitlog.NewNopLogger())

		err := s.CreateFile(ctx, key, buff, rep)
		require.NoError(t, err)
	})
	t.Run("SuccessWithConfigReplica", func(t *testing.T) {
		var (
			key  = "expectedkey"
			buff = io.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 2
		)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		v.EXPECT().CreateFile(gomock.Any(), key, buff, rep).Return(nil)

		// It's AnyTimes as we have the config witha number of replicas
		// which activates the goroutines that also calls this
		m.EXPECT().LocalVolumes().Return([]volume.Local{v}).AnyTimes()

		// This is also because of the goroutine, it may call it or not
		v.EXPECT().NextReplica(gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()
		m.EXPECT().RemovedVolumeIDs().Return(nil).AnyTimes()

		s := storing.New(&config.Config{Replica: rep}, m, kitlog.NewNopLogger())

		err := s.CreateFile(ctx, key, buff, 0)
		require.NoError(t, err)
	})
	t.Run("SuccessMultiVolume", func(t *testing.T) {
		t.Skip("Not yet thought")
	})
}

func TestGetFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			key  = "expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
		)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})

		v.EXPECT().HasFile(gomock.Any(), key).Return(true, nil)
		v.EXPECT().GetFile(gomock.Any(), key).Return(io.NopCloser(bytes.NewBufferString("expectedcontent")), nil)

		s := storing.New(&config.Config{Replica: -1}, m, kitlog.NewNopLogger())
		ior, err := s.GetFile(ctx, key)

		require.NoError(t, err)

		b, err := io.ReadAll(ior)
		assert.Equal(t, "expectedcontent", string(b))
	})
	t.Run("SuccessMultiVolume", func(t *testing.T) {
		var (
			key  = "expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
		)
		v := mock.NewVolumeLocal(ctrl)
		s2 := mock.NewStoring(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})
		m.EXPECT().Nodes().Return([]storing.Service{s2})

		v.EXPECT().HasFile(gomock.Any(), key).Return(false, nil)
		s2.EXPECT().HasFile(gomock.Any(), key).Return(true, nil)
		s2.EXPECT().GetFile(gomock.Any(), key).Return(io.NopCloser(bytes.NewBufferString("expectedcontent")), nil)

		s := storing.New(&config.Config{Replica: -1}, m, kitlog.NewNopLogger())

		ior, err := s.GetFile(ctx, key)
		require.NoError(t, err)

		b, err := io.ReadAll(ior)
		assert.Equal(t, "expectedcontent", string(b))
	})
}

func TestDeleteFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			ctrl = gomock.NewController(t)
			key  = "expectedkey"
			ctx  = context.Background()
		)
		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})

		v.EXPECT().HasFile(gomock.Any(), key).Return(true, nil)
		v.EXPECT().DeleteFile(gomock.Any(), key).Return(nil)

		s := storing.New(&config.Config{Replica: -1}, m, kitlog.NewNopLogger())

		err := s.DeleteFile(ctx, key)
		require.NoError(t, err)
	})
	t.Run("SuccessMultiVolume", func(t *testing.T) {
		var (
			key  = "expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
		)
		v := mock.NewVolumeLocal(ctrl)
		s2 := mock.NewStoring(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})
		m.EXPECT().Nodes().Return([]storing.Service{s2})

		v.EXPECT().HasFile(gomock.Any(), key).Return(false, nil)
		s2.EXPECT().HasFile(gomock.Any(), key).Return(true, nil)
		s2.EXPECT().DeleteFile(gomock.Any(), key).Return(nil)

		s := storing.New(&config.Config{Replica: -1}, m, kitlog.NewNopLogger())

		err := s.DeleteFile(ctx, key)
		require.NoError(t, err)
	})
}

func TestHasFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			ctrl = gomock.NewController(t)
			key  = "expectedkey"
			ctx  = context.Background()
		)
		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})

		v.EXPECT().HasFile(gomock.Any(), key).Return(true, nil)

		s := storing.New(&config.Config{Replica: -1}, m, kitlog.NewNopLogger())

		ok, err := s.HasFile(ctx, key)
		require.NoError(t, err)
		assert.True(t, ok)
	})
	t.Run("SuccessMultiVolume", func(t *testing.T) {
		var (
			key  = "expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
		)
		v := mock.NewVolumeLocal(ctrl)
		v2 := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Local{v, v2})

		// This call is execute in paralel to all the volumes
		// so the orther is "unexpected". This means that some
		// times the first call it's not done. That's why the
		// AnyTimes is used
		v.EXPECT().HasFile(gomock.Any(), key).Return(false, nil).AnyTimes()
		v2.EXPECT().HasFile(gomock.Any(), key).Return(true, nil)

		s := storing.New(&config.Config{Replica: -1}, m, kitlog.NewNopLogger())

		ok, err := s.HasFile(ctx, key)
		require.NoError(t, err)
		assert.True(t, ok)
	})
	t.Run("SuccessFalse", func(t *testing.T) {
		var (
			ctrl = gomock.NewController(t)
			key  = "expectedkey"
			ctx  = context.Background()
		)
		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})

		v.EXPECT().HasFile(gomock.Any(), key).Return(false, nil)

		s := storing.New(&config.Config{Replica: -1}, m, kitlog.NewNopLogger())

		ok, err := s.HasFile(ctx, key)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestConfig(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			ctrl   = gomock.NewController(t)
			ctx    = context.Background()
			expcfg = config.Config{MemberlistName: "Pepito", Replica: -1}
		)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		s := storing.New(&expcfg, m, kitlog.NewNopLogger())

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
		)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		v.EXPECT().CreateFile(gomock.Any(), key, buff, 1).Return(nil)

		// It's AnyTimes as we have the config witha number of replicas
		// which activates the goroutines that also calls this
		m.EXPECT().LocalVolumes().Return([]volume.Local{v}).AnyTimes()

		// This is also because of the goroutine, it may call it or not
		v.EXPECT().NextReplica(gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()
		m.EXPECT().RemovedVolumeIDs().Return(nil).AnyTimes()

		v.EXPECT().ID().Return(createdToVolID)

		s := storing.New(&config.Config{}, m, kitlog.NewNopLogger())

		volID, err := s.CreateReplica(ctx, key, buff)
		require.NoError(t, err)
		assert.Equal(t, createdToVolID, volID)
	})
	t.Run("ErrorNoReplica", func(t *testing.T) {
		var (
			key  = "expectedkey"
			buff = io.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
		)

		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		s := storing.New(&config.Config{Replica: -1}, m, kitlog.NewNopLogger())

		volID, err := s.CreateReplica(ctx, key, buff)
		assert.EqualError(t, err, "can not store replicas")
		assert.Equal(t, "", volID)
	})
}

func TestUpdateFileReplica(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			key  = "expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 4
			vids = []string{"1", "2"}
		)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		v.EXPECT().HasFile(gomock.Any(), key).Return(true, nil)
		v.EXPECT().UpdateFileReplica(gomock.Any(), key, vids, rep).Return(nil)

		// It's AnyTimes as we have the config witha number of replicas
		// which activates the goroutines that also calls this
		m.EXPECT().LocalVolumes().Return([]volume.Local{v}).AnyTimes()

		// This is also because of the goroutine, it may call it or not
		v.EXPECT().NextReplica(gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()
		m.EXPECT().RemovedVolumeIDs().Return(nil).AnyTimes()

		s := storing.New(&config.Config{}, m, kitlog.NewNopLogger())

		err := s.UpdateFileReplica(ctx, key, vids, rep)
		require.NoError(t, err)
	})
	t.Run("ErrorNoReplica", func(t *testing.T) {
		var (
			key  = "expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 4
			vids = []string{"1", "2"}
		)

		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		s := storing.New(&config.Config{Replica: -1}, m, kitlog.NewNopLogger())

		err := s.UpdateFileReplica(ctx, key, vids, rep)
		assert.EqualError(t, err, "can not store replicas")
	})
}
