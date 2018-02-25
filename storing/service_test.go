package storing_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

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
			v    mock.Volume
			key  = "expectedkey"
			buff = bytes.NewBufferString("expectedcontent")
		)

		v.CreateFileFn = func(k string, r io.Reader) (*file.File, error) {
			assert.Equal(t, key, k)
			return &file.File{
				Keys:      []string{key},
				Signature: "signature",
			}, nil
		}

		s := storing.New([]volume.Volume{&v})
		err := s.CreateFile(key, buff)

		require.NoError(t, err)
		assert.True(t, v.CreateFileInvoked)
	})
	t.Run("SuccessMultiVolume", func(t *testing.T) {
		t.Skip("Not yet thought")
	})
}

func TestGetFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			v   mock.Volume
			key = "expectedkey"
		)

		v.HasFileFn = func(k string) (bool, error) {
			assert.Equal(t, key, k)
			return true, nil
		}

		v.GetFileFn = func(k string) (io.Reader, error) {
			assert.Equal(t, key, k)
			return bytes.NewBufferString("expectedcontent"), nil
		}

		s := storing.New([]volume.Volume{&v})
		ior, err := s.GetFile(key)

		require.NoError(t, err)
		assert.True(t, v.GetFileInvoked)
		assert.True(t, v.HasFileInvoked)

		b, err := ioutil.ReadAll(ior)
		assert.Equal(t, "expectedcontent", string(b))
	})
	t.Run("SuccessMultiVolume", func(t *testing.T) {
		var (
			v   mock.Volume
			v2  mock.Volume
			key = "expectedkey"
		)

		v.HasFileFn = func(k string) (bool, error) {
			assert.Equal(t, key, k)
			return false, nil
		}

		v2.HasFileFn = func(k string) (bool, error) {
			assert.Equal(t, key, k)
			return true, nil
		}

		v2.GetFileFn = func(k string) (io.Reader, error) {
			assert.Equal(t, key, k)
			return bytes.NewBufferString("expectedcontent"), nil
		}

		s := storing.New([]volume.Volume{&v, &v2})
		ior, err := s.GetFile(key)

		require.NoError(t, err)
		assert.True(t, v.HasFileInvoked)
		assert.True(t, v2.HasFileInvoked)
		assert.True(t, v2.GetFileInvoked)

		b, err := ioutil.ReadAll(ior)
		assert.Equal(t, "expectedcontent", string(b))
	})
}

func TestDeleteFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			v   mock.Volume
			key = "expectedkey"
		)

		v.HasFileFn = func(k string) (bool, error) {
			assert.Equal(t, key, k)
			return true, nil
		}

		v.DeleteFileFn = func(k string) error {
			assert.Equal(t, key, k)
			return nil
		}

		s := storing.New([]volume.Volume{&v})
		err := s.DeleteFile(key)

		require.NoError(t, err)
		assert.True(t, v.DeleteFileInvoked)
		assert.True(t, v.HasFileInvoked)
	})
	t.Run("SuccessMultiVolume", func(t *testing.T) {
		var (
			v   mock.Volume
			v2  mock.Volume
			key = "expectedkey"
		)
		v.HasFileFn = func(k string) (bool, error) {
			assert.Equal(t, key, k)
			return false, nil
		}

		v2.HasFileFn = func(k string) (bool, error) {
			assert.Equal(t, key, k)
			return true, nil
		}

		v2.DeleteFileFn = func(k string) error {
			assert.Equal(t, key, k)
			return nil
		}

		s := storing.New([]volume.Volume{&v, &v2})
		err := s.DeleteFile(key)

		require.NoError(t, err)
		assert.True(t, v.HasFileInvoked)
		assert.True(t, v2.HasFileInvoked)
		assert.True(t, v2.DeleteFileInvoked)
	})
}
