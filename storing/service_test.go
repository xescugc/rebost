package storing_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/storing"
	"github.com/xescugc/rebost/volume"
)

func TestCreateFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			key  = "expectedkey"
			buff = bytes.NewBufferString("expectedcontent")
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
		)

		v := mock.NewVolume(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		v.EXPECT().CreateFile(gomock.Any(), key, buff).Return(nil)

		m.EXPECT().LocalVolumes().Return([]volume.Volume{v})

		s := storing.New(m)

		err := s.CreateFile(ctx, key, buff)
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

		v := mock.NewVolume(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Volume{v})

		v.EXPECT().HasFile(gomock.Any(), key).Return(true, nil)
		v.EXPECT().GetFile(gomock.Any(), key).Return(bytes.NewBufferString("expectedcontent"), nil)

		s := storing.New(m)
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
		v := mock.NewVolume(ctrl)
		v2 := mock.NewVolume(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Volume{v})
		m.EXPECT().RemoteVolumes().Return([]volume.Volume{v2})

		v.EXPECT().HasFile(gomock.Any(), key).Return(false, nil)
		v2.EXPECT().HasFile(gomock.Any(), key).Return(true, nil)
		v2.EXPECT().GetFile(gomock.Any(), key).Return(bytes.NewBufferString("expectedcontent"), nil)

		s := storing.New(m)

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
		v := mock.NewVolume(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Volume{v})

		v.EXPECT().HasFile(gomock.Any(), key).Return(true, nil)
		v.EXPECT().DeleteFile(gomock.Any(), key).Return(nil)

		s := storing.New(m)

		err := s.DeleteFile(ctx, key)
		require.NoError(t, err)
	})
	t.Run("SuccessMultiVolume", func(t *testing.T) {
		var (
			key  = "expectedkey"
			ctrl = gomock.NewController(t)
			ctx  = context.Background()
		)
		v := mock.NewVolume(ctrl)
		v2 := mock.NewVolume(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Volume{v})
		m.EXPECT().RemoteVolumes().Return([]volume.Volume{v2})

		v.EXPECT().HasFile(gomock.Any(), key).Return(false, nil)
		v2.EXPECT().HasFile(gomock.Any(), key).Return(true, nil)
		v2.EXPECT().DeleteFile(gomock.Any(), key).Return(nil)

		s := storing.New(m)

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
		v := mock.NewVolume(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Volume{v})

		v.EXPECT().HasFile(gomock.Any(), key).Return(true, nil)

		s := storing.New(m)

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
		v := mock.NewVolume(ctrl)
		v2 := mock.NewVolume(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Volume{v, v2})

		v.EXPECT().HasFile(gomock.Any(), key).Return(false, nil)
		v2.EXPECT().HasFile(gomock.Any(), key).Return(true, nil)

		s := storing.New(m)

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
		v := mock.NewVolume(ctrl)
		m := mock.NewMembership(ctrl)
		defer ctrl.Finish()

		m.EXPECT().LocalVolumes().Return([]volume.Volume{v})

		v.EXPECT().HasFile(gomock.Any(), key).Return(false, nil)

		s := storing.New(m)

		ok, err := s.HasFile(ctx, key)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}
