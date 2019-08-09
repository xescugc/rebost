package storing_test

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/storing"
	"github.com/xescugc/rebost/volume"
)

func TestCreateFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			key  = "expectedkey"
			buff = ioutil.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 2
		)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		v.EXPECT().CreateFile(gomock.Any(), key, buff, rep).Return(nil)

		m.EXPECT().LocalVolumes().Return([]volume.Local{v})

		s := storing.New(&config.Config{Replica: -1}, m)

		err := s.CreateFile(ctx, key, buff, rep)
		require.NoError(t, err)
	})
	t.Run("SuccessWithConfigReplica", func(t *testing.T) {
		var (
			key  = "expectedkey"
			buff = ioutil.NopCloser(bytes.NewBufferString("expectedcontent"))
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

		s := storing.New(&config.Config{Replica: rep}, m)

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
		v.EXPECT().GetFile(gomock.Any(), key).Return(ioutil.NopCloser(bytes.NewBufferString("expectedcontent")), nil)

		s := storing.New(&config.Config{Replica: -1}, m)
		ior, err := s.GetFile(ctx, key)

		require.NoError(t, err)

		b, err := ioutil.ReadAll(ior)
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
		s2.EXPECT().GetFile(gomock.Any(), key).Return(ioutil.NopCloser(bytes.NewBufferString("expectedcontent")), nil)

		s := storing.New(&config.Config{Replica: -1}, m)

		ior, err := s.GetFile(ctx, key)
		require.NoError(t, err)

		b, err := ioutil.ReadAll(ior)
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

		s := storing.New(&config.Config{Replica: -1}, m)

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

		s := storing.New(&config.Config{Replica: -1}, m)

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

		s := storing.New(&config.Config{Replica: -1}, m)

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

		s := storing.New(&config.Config{Replica: -1}, m)

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

		s := storing.New(&config.Config{Replica: -1}, m)

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

		s := storing.New(&expcfg, m)

		cfg, err := s.Config(ctx)
		require.NoError(t, err)
		assert.Equal(t, &expcfg, cfg)
	})
}

func TestCreateReplica(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			key            = "expectedkey"
			buff           = ioutil.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl           = gomock.NewController(t)
			ctx            = context.Background()
			rep            = 4
			originVolID    = "originVolID"
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

		v.EXPECT().UpdateReplica(ctx, &replica.Replica{
			Key:           key,
			OriginalCount: rep,
		}, originVolID)

		v.EXPECT().ID().Return(createdToVolID)

		s := storing.New(&config.Config{}, m)

		volID, err := s.CreateReplica(ctx, key, buff, "originVolID", 4)
		require.NoError(t, err)
		assert.Equal(t, createdToVolID, volID)
	})
	t.Run("ErrorOriginVolumeID", func(t *testing.T) {
		var (
			key  = "expectedkey"
			buff = ioutil.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 4
		)

		v := mock.NewVolumeLocal(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		// It's AnyTimes as we have the config witha number of replicas
		// which activates the goroutines that also calls this
		m.EXPECT().LocalVolumes().Return([]volume.Local{v}).AnyTimes()

		// This is also because of the goroutine, it may call it or not
		v.EXPECT().NextReplica(gomock.Any()).Return(nil, errors.New("not found")).AnyTimes()

		s := storing.New(&config.Config{}, m)

		volID, err := s.CreateReplica(ctx, key, buff, "", rep)
		assert.EqualError(t, err, "the originVolumeID is required")
		assert.Equal(t, "", volID)
	})
	t.Run("ErrorNoReplica", func(t *testing.T) {
		var (
			key  = "expectedkey"
			buff = ioutil.NopCloser(bytes.NewBufferString("expectedcontent"))
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
			rep  = 4
		)

		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		s := storing.New(&config.Config{Replica: -1}, m)

		volID, err := s.CreateReplica(ctx, key, buff, "", rep)
		assert.EqualError(t, err, "can not store replicas")
		assert.Equal(t, "", volID)
	})
}
