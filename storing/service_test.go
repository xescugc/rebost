package storing_test

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/file"
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
		)

		v := mock.NewVolume(ctrl)
		defer ctrl.Finish()

		v.EXPECT().CreateFile(key, buff).Return(&file.File{
			Keys:      []string{key},
			Signature: "signature",
		}, nil)

		s := storing.New([]volume.Volume{v})

		err := s.CreateFile(key, buff)
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
		)

		v := mock.NewVolume(ctrl)
		defer ctrl.Finish()

		v.EXPECT().HasFile(key).Return(true, nil)
		v.EXPECT().GetFile(key).Return(bytes.NewBufferString("expectedcontent"), nil)

		s := storing.New([]volume.Volume{v})
		ior, err := s.GetFile(key)

		require.NoError(t, err)

		b, err := ioutil.ReadAll(ior)
		assert.Equal(t, "expectedcontent", string(b))
	})
	t.Run("SuccessMultiVolume", func(t *testing.T) {
		var (
			key   = "expectedkey"
			ctrl  = gomock.NewController(t)
			ctrl2 = gomock.NewController(t)
		)
		v := mock.NewVolume(ctrl)
		defer ctrl.Finish()
		v2 := mock.NewVolume(ctrl2)
		defer ctrl2.Finish()

		v.EXPECT().HasFile(key).Return(false, nil)
		v2.EXPECT().HasFile(key).Return(true, nil)
		v2.EXPECT().GetFile(key).Return(bytes.NewBufferString("expectedcontent"), nil)

		s := storing.New([]volume.Volume{v, v2})

		ior, err := s.GetFile(key)
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
		)
		v := mock.NewVolume(ctrl)
		defer ctrl.Finish()

		v.EXPECT().HasFile(key).Return(true, nil)
		v.EXPECT().DeleteFile(key).Return(nil)

		s := storing.New([]volume.Volume{v})

		err := s.DeleteFile(key)
		require.NoError(t, err)
	})
	t.Run("SuccessMultiVolume", func(t *testing.T) {
		var (
			key   = "expectedkey"
			ctrl  = gomock.NewController(t)
			ctrl2 = gomock.NewController(t)
		)
		v := mock.NewVolume(ctrl)
		defer ctrl.Finish()
		v2 := mock.NewVolume(ctrl2)
		defer ctrl2.Finish()

		v.EXPECT().HasFile(key).Return(false, nil)
		v2.EXPECT().HasFile(key).Return(true, nil)
		v2.EXPECT().DeleteFile(key).Return(nil)

		s := storing.New([]volume.Volume{v, v2})

		err := s.DeleteFile(key)
		require.NoError(t, err)
	})
}
