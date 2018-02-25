package volume_test

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/afero/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/idxkey"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/uow"
	"github.com/xescugc/rebost/volume"
)

func TestNew(t *testing.T) {
	var files mock.FileRepository
	var fs mock.Fs
	var idxkeys mock.IDXKeyRepository
	var suow uow.StartUnitOfWork

	fs.MkdirAllFn = func(p string, fm os.FileMode) error {
		assert.Equal(t, os.ModePerm, fm)
		switch p {
		case path.Join("root", "file"):
			return nil
		case path.Join("root", "tmps"):
			return nil
		default:
			return fmt.Errorf("test error for path: %q", p)
		}
	}

	v, err := volume.New("root", &files, &idxkeys, &fs, suow)
	require.NoError(t, err)
	assert.NotNil(t, v)

	assert.True(t, fs.MkdirAllInvoked)
}

func TestCreateFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir  = "root"
			mv       = mock.NewManageVolume(t, rootDir)
			key      = "expectedkey"
			tempuuid string
		)
		buff := bytes.NewBufferString("content of the file")
		ef := file.File{
			Keys:      []string{key},
			Signature: "e7e8c72d1167454b76a610074fed244be0935298",
		}
		eik := idxkey.IDXKey{
			Key:   key,
			Value: ef.Signature,
		}

		mv.Fs.OpenFileFn = func(p string, flag int, perm os.FileMode) (afero.File, error) {
			assert.True(t, strings.HasPrefix(p, path.Join(rootDir, "tmps")))

			_, tempuuid = path.Split(p)
			tf := mem.NewFileHandle(mem.CreateFile(p))
			return tf, nil
		}

		mv.Fs.MkdirAllFn = func(p string, fm os.FileMode) error {
			assert.Equal(t, os.ModePerm, fm)
			dir, _ := path.Split(ef.Path(path.Join(rootDir, "file")))
			switch p {
			case path.Join(rootDir, "tmps", tempuuid):
				return nil
			case dir:
				return nil
			default:
				return fmt.Errorf("test error for path: %q", p)
			}
		}

		mv.Fs.RenameFn = func(oldpath, newpath string) error {
			assert.Equal(t, path.Join(rootDir, "tmps", tempuuid), oldpath)
			assert.Equal(t, ef.Path(path.Join(rootDir, "file")), newpath)
			return nil
		}

		mv.Files.FindBySignatureFn = func(sig string) (*file.File, error) {
			assert.Equal(t, ef.Signature, sig)
			return nil, errors.New("not found")
		}

		mv.Files.CreateOrReplaceFn = func(fl *file.File) error {
			assert.Equal(t, &ef, fl)
			return nil
		}

		mv.IDXKeys.FindByKeyFn = func(k string) (*idxkey.IDXKey, error) {
			assert.Equal(t, ef.Keys[0], k)
			return nil, errors.New("not found")
		}

		mv.IDXKeys.CreateOrReplaceFn = func(ik *idxkey.IDXKey) error {
			assert.Equal(t, &eik, ik)
			return nil
		}

		f, err := mv.V.CreateFile(ef.Keys[0], buff)
		require.NoError(t, err)

		assert.Equal(t, &ef, f)
		assert.True(t, mv.Fs.MkdirAllInvoked)
		assert.True(t, mv.Fs.OpenFileInvoked)
		assert.True(t, mv.Fs.RenameInvoked)
		assert.True(t, mv.Files.CreateOrReplaceInvoked)
		assert.True(t, mv.Files.FindBySignatureInvoked)
		assert.True(t, mv.IDXKeys.CreateOrReplaceInvoked)
		assert.True(t, mv.IDXKeys.FindByKeyInvoked)
	})
	t.Run("SuccessUpdateFileKey", func(t *testing.T) {
		var (
			rootDir  = "root"
			mv       = mock.NewManageVolume(t, rootDir)
			key      = "expectedkey"
			tempuuid string
			buff     = bytes.NewBufferString("content of the file")
		)
		ef := file.File{
			Keys:      []string{"b", key},
			Signature: "e7e8c72d1167454b76a610074fed244be0935298",
		}
		eik := idxkey.IDXKey{
			Key:   "expectedkey",
			Value: ef.Signature,
		}

		mv.Fs.OpenFileFn = func(p string, flag int, perm os.FileMode) (afero.File, error) {
			assert.True(t, strings.HasPrefix(p, path.Join(rootDir, "tmps")))

			_, tempuuid = path.Split(p)
			tf := mem.NewFileHandle(mem.CreateFile(p))
			return tf, nil
		}

		mv.Fs.MkdirAllFn = func(p string, fm os.FileMode) error {
			assert.Equal(t, os.ModePerm, fm)
			dir, _ := path.Split(ef.Path(path.Join(rootDir, "file")))
			switch p {
			case path.Join("root", "tmps", tempuuid):
				return nil
			case dir:
				return nil
			default:
				return fmt.Errorf("test error for path: %q", p)
			}
		}

		mv.Fs.RenameFn = func(oldpath, newpath string) error {
			assert.Equal(t, path.Join(rootDir, "tmps", tempuuid), oldpath)
			assert.Equal(t, ef.Path(path.Join(rootDir, "file")), newpath)
			return nil
		}

		mv.Files.FindBySignatureFn = func(sig string) (*file.File, error) {
			return &file.File{
				Keys:      []string{"b"},
				Signature: sig,
			}, nil
		}

		mv.Files.CreateOrReplaceFn = func(fl *file.File) error {
			assert.Equal(t, &ef, fl)
			return nil
		}

		mv.IDXKeys.FindByKeyFn = func(k string) (*idxkey.IDXKey, error) {
			assert.Equal(t, key, k)
			return nil, errors.New("not found")
		}

		mv.IDXKeys.CreateOrReplaceFn = func(ik *idxkey.IDXKey) error {
			assert.Equal(t, &eik, ik)
			return nil
		}

		f, err := mv.V.CreateFile(key, buff)
		require.NoError(t, err)

		assert.Equal(t, &ef, f)
		assert.True(t, mv.Fs.MkdirAllInvoked)
		assert.True(t, mv.Fs.OpenFileInvoked)
		assert.True(t, mv.Fs.RenameInvoked)
		assert.True(t, mv.Files.CreateOrReplaceInvoked)
		assert.True(t, mv.Files.FindBySignatureInvoked)
		assert.True(t, mv.IDXKeys.CreateOrReplaceInvoked)
		assert.True(t, mv.IDXKeys.FindByKeyInvoked)
	})
	t.Run("SuccessSame", func(t *testing.T) {
		var (
			rootDir  = "root"
			mv       = mock.NewManageVolume(t, rootDir)
			key      = "expectedkey"
			tempuuid string
		)
		buff := bytes.NewBufferString("content of the file")
		ef := file.File{
			Keys:      []string{key},
			Signature: "e7e8c72d1167454b76a610074fed244be0935298",
		}

		mv.Fs.OpenFileFn = func(p string, flag int, perm os.FileMode) (afero.File, error) {
			assert.True(t, strings.HasPrefix(p, path.Join(rootDir, "tmps")))

			_, tempuuid = path.Split(p)
			tf := mem.NewFileHandle(mem.CreateFile(p))
			return tf, nil
		}

		mv.Fs.MkdirAllFn = func(p string, fm os.FileMode) error {
			assert.Equal(t, os.ModePerm, fm)
			dir, _ := path.Split(ef.Path(path.Join(rootDir, "file")))
			switch p {
			case path.Join(rootDir, "tmps", tempuuid):
				return nil
			case dir:
				return nil
			default:
				return fmt.Errorf("test error for path: %q", p)
			}
		}

		mv.Fs.RenameFn = func(oldpath, newpath string) error {
			assert.Equal(t, path.Join(rootDir, "tmps", tempuuid), oldpath)
			assert.Equal(t, ef.Path(path.Join(rootDir, "file")), newpath)
			return nil
		}

		mv.Files.FindBySignatureFn = func(sig string) (*file.File, error) {
			assert.Equal(t, ef.Signature, sig)
			return &ef, nil
		}

		f, err := mv.V.CreateFile(ef.Keys[0], buff)
		require.NoError(t, err)

		assert.Equal(t, &ef, f)
		assert.True(t, mv.Fs.MkdirAllInvoked)
		assert.True(t, mv.Fs.OpenFileInvoked)
		assert.True(t, mv.Fs.RenameInvoked)
		assert.True(t, mv.Files.FindBySignatureInvoked)
	})
	t.Run("SuccessRemoveFileKey", func(t *testing.T) {
		var (
			rootDir  = "root"
			mv       = mock.NewManageVolume(t, rootDir)
			key      = "expectedkey"
			tempuuid string
			buff     = bytes.NewBufferString("content of the file")
		)
		ef := file.File{
			Keys:      []string{key},
			Signature: "e7e8c72d1167454b76a610074fed244be0935298",
		}
		eik := idxkey.IDXKey{
			Key:   key,
			Value: ef.Signature,
		}

		mv.Fs.OpenFileFn = func(p string, flag int, perm os.FileMode) (afero.File, error) {
			assert.True(t, strings.HasPrefix(p, path.Join(rootDir, "tmps")))

			_, tempuuid = path.Split(p)
			tf := mem.NewFileHandle(mem.CreateFile(p))
			return tf, nil
		}

		mv.Fs.MkdirAllFn = func(p string, fm os.FileMode) error {
			assert.Equal(t, os.ModePerm, fm)
			dir, _ := path.Split(ef.Path(path.Join(rootDir, "file")))
			switch p {
			case path.Join("root", "tmps", tempuuid):
				return nil
			case dir:
				return nil
			default:
				return fmt.Errorf("test error for path: %q", p)
			}
		}

		mv.Fs.RenameFn = func(oldpath, newpath string) error {
			assert.Equal(t, path.Join(rootDir, "tmps", tempuuid), oldpath)
			assert.Equal(t, ef.Path(path.Join(rootDir, "file")), newpath)
			return nil
		}

		foundFile := file.File{
			Keys:      []string{key, "b"},
			Signature: "123123123",
		}

		mv.Files.FindBySignatureFn = func(sig string) (*file.File, error) {
			if sig == ef.Signature {
				return nil, errors.New("not found")
			}
			return &file.File{
				Keys:      foundFile.Keys,
				Signature: foundFile.Signature,
			}, nil
		}

		mv.Files.CreateOrReplaceFn = func(fl *file.File) error {
			if fl.Signature == ef.Signature {
				assert.Equal(t, &ef, fl)
				return nil
			}
			foundFile.Keys = []string{"b"}
			assert.Equal(t, &foundFile, fl)
			return nil
		}

		mv.IDXKeys.FindByKeyFn = func(k string) (*idxkey.IDXKey, error) {
			assert.Equal(t, key, k)
			return idxkey.New(k, foundFile.Signature), nil
		}

		mv.IDXKeys.CreateOrReplaceFn = func(ik *idxkey.IDXKey) error {
			assert.Equal(t, &eik, ik)
			return nil
		}

		f, err := mv.V.CreateFile(key, buff)
		require.NoError(t, err)

		assert.Equal(t, &ef, f)
		assert.Equal(t, 2, mv.Files.CreateOrReplaceTimes)
		assert.Equal(t, 2, mv.Files.FindBySignatureTimes)
		assert.True(t, mv.Fs.MkdirAllInvoked)
		assert.True(t, mv.Fs.OpenFileInvoked)
		assert.True(t, mv.Fs.RenameInvoked)
		assert.True(t, mv.Files.CreateOrReplaceInvoked)
		assert.True(t, mv.Files.FindBySignatureInvoked)
		assert.True(t, mv.IDXKeys.CreateOrReplaceInvoked)
		assert.True(t, mv.IDXKeys.FindByKeyInvoked)
	})
	t.Run("SuccessRemoveFileKeyAndFile", func(t *testing.T) {
		var (
			rootDir  = "root"
			mv       = mock.NewManageVolume(t, rootDir)
			key      = "expectedkey"
			tempuuid string
			buff     = bytes.NewBufferString("content of the file")
		)
		sh1 := sha1.New()
		sh1.Write(buff.Bytes())
		ef := file.File{
			Keys:      []string{key},
			Signature: "e7e8c72d1167454b76a610074fed244be0935298",
		}
		eik := idxkey.IDXKey{
			Key:   key,
			Value: ef.Signature,
		}

		mv.Fs.OpenFileFn = func(p string, flag int, perm os.FileMode) (afero.File, error) {
			assert.True(t, strings.HasPrefix(p, path.Join(rootDir, "tmps")))

			_, tempuuid = path.Split(p)
			tf := mem.NewFileHandle(mem.CreateFile(p))
			return tf, nil
		}

		mv.Fs.MkdirAllFn = func(p string, fm os.FileMode) error {
			assert.Equal(t, os.ModePerm, fm)
			dir, _ := path.Split(ef.Path(path.Join(rootDir, "file")))
			switch p {
			case path.Join("root", "tmps", tempuuid):
				return nil
			case dir:
				return nil
			default:
				return fmt.Errorf("test error for path: %q", p)
			}
		}

		mv.Fs.RenameFn = func(oldpath, newpath string) error {
			assert.Equal(t, path.Join(rootDir, "tmps", tempuuid), oldpath)
			assert.Equal(t, ef.Path(path.Join(rootDir, "file")), newpath)
			return nil
		}

		foundFile := file.File{
			Keys:      []string{key},
			Signature: "123123123",
		}

		mv.Files.FindBySignatureFn = func(sig string) (*file.File, error) {
			if sig == ef.Signature {
				return nil, errors.New("not found")
			}
			return &file.File{
				Keys:      foundFile.Keys,
				Signature: foundFile.Signature,
			}, nil
		}

		mv.Files.CreateOrReplaceFn = func(fl *file.File) error {
			assert.Equal(t, &ef, fl)
			return nil
		}

		mv.Files.DeleteBySignatureFn = func(sig string) error {
			assert.Equal(t, foundFile.Signature, sig)
			return nil
		}

		mv.Fs.RemoveFn = func(p string) error {
			assert.Equal(t, foundFile.Path(path.Join(rootDir, "file")), p)
			return nil
		}

		mv.IDXKeys.FindByKeyFn = func(k string) (*idxkey.IDXKey, error) {
			assert.Equal(t, key, k)
			return idxkey.New(k, foundFile.Signature), nil
		}

		mv.IDXKeys.DeleteByKeyFn = func(k string) error {
			assert.Equal(t, key, k)
			return nil
		}

		mv.IDXKeys.CreateOrReplaceFn = func(ik *idxkey.IDXKey) error {
			assert.Equal(t, &eik, ik)
			return nil
		}

		f, err := mv.V.CreateFile(key, buff)
		require.NoError(t, err)

		assert.Equal(t, &ef, f)
		assert.Equal(t, 1, mv.Files.CreateOrReplaceTimes)
		assert.Equal(t, 2, mv.Files.FindBySignatureTimes)
		assert.True(t, mv.Fs.MkdirAllInvoked)
		assert.True(t, mv.Fs.OpenFileInvoked)
		assert.True(t, mv.Fs.RenameInvoked)
		assert.True(t, mv.Fs.RemoveInvoked)
		assert.True(t, mv.Files.CreateOrReplaceInvoked)
		assert.True(t, mv.Files.FindBySignatureInvoked)
		assert.True(t, mv.Files.DeleteBySignatureInvoked)
		assert.True(t, mv.IDXKeys.CreateOrReplaceInvoked)
		assert.True(t, mv.IDXKeys.FindByKeyInvoked)
		assert.True(t, mv.IDXKeys.DeleteByKeyInvoked)
	})
}

func TestGetFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir   = "root"
			mv        = mock.NewManageVolume(t, rootDir)
			key       = "expectedkey"
			signature = "123123123"
			content   = "expectedcontent"
		)

		mv.IDXKeys.FindByKeyFn = func(k string) (*idxkey.IDXKey, error) {
			assert.Equal(t, key, k)
			return idxkey.New(k, signature), nil
		}

		mv.Fs.OpenFileFn = func(p string, flag int, perm os.FileMode) (afero.File, error) {
			assert.Equal(t, file.Path(path.Join(rootDir, "file"), signature), p)

			tf := mem.NewFileHandle(mem.CreateFile(p))
			tf.WriteString(content)
			// After finishing writting, the "cursor" points to the end of
			// the content, so to be able to read all of it, has to be
			// set to the beggining, that's what the Seek does
			tf.Seek(0, 0)
			return tf, nil
		}

		ior, err := mv.V.GetFile(key)
		require.NoError(t, err)
		require.NotNil(t, ior)
		b, err := ioutil.ReadAll(ior)
		require.NoError(t, err)

		assert.True(t, mv.Fs.MkdirAllInvoked)
		assert.True(t, mv.Fs.OpenFileInvoked)
		assert.True(t, mv.IDXKeys.FindByKeyInvoked)
		assert.Equal(t, content, string(b))
	})
	t.Run("NotFound", func(t *testing.T) {
		var (
			rootDir = "root"
			mv      = mock.NewManageVolume(t, rootDir)
			key     = "expectedkey"
		)

		mv.IDXKeys.FindByKeyFn = func(k string) (*idxkey.IDXKey, error) {
			assert.Equal(t, key, k)
			return nil, errors.New("not found")
		}

		_, err := mv.V.GetFile(key)
		assert.EqualError(t, err, errors.New("not found").Error())

		assert.True(t, mv.Fs.MkdirAllInvoked)
		assert.False(t, mv.Fs.OpenFileInvoked)
		assert.True(t, mv.IDXKeys.FindByKeyInvoked)
	})
}

func TestHasFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir = "root"
			mv      = mock.NewManageVolume(t, rootDir)
			key     = "expectedkey"
		)

		mv.IDXKeys.FindByKeyFn = func(k string) (*idxkey.IDXKey, error) {
			assert.Equal(t, key, k)
			return idxkey.New(k, "not needed"), nil
		}

		ok, err := mv.V.HasFile(key)
		require.NoError(t, err)

		assert.True(t, ok)
		assert.True(t, mv.Fs.MkdirAllInvoked)
		assert.True(t, mv.IDXKeys.FindByKeyInvoked)
	})
	t.Run("NotFound", func(t *testing.T) {
		var (
			rootDir = "root"
			mv      = mock.NewManageVolume(t, rootDir)
			key     = "expectedkey"
		)

		mv.IDXKeys.FindByKeyFn = func(k string) (*idxkey.IDXKey, error) {
			assert.Equal(t, key, k)
			return nil, errors.New("not found")
		}

		ok, err := mv.V.HasFile(key)
		require.NoError(t, err)

		assert.False(t, ok)
		assert.True(t, mv.Fs.MkdirAllInvoked)
		assert.True(t, mv.IDXKeys.FindByKeyInvoked)
	})
}

func TestDeleteFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir   = "root"
			mv        = mock.NewManageVolume(t, rootDir)
			key       = "expectedkey"
			signature = "123123123"
			ef        = file.File{
				Keys:      []string{key},
				Signature: signature,
			}
		)

		mv.IDXKeys.FindByKeyFn = func(k string) (*idxkey.IDXKey, error) {
			assert.Equal(t, key, k)
			return idxkey.New(k, signature), nil
		}

		mv.Files.FindBySignatureFn = func(sig string) (*file.File, error) {
			assert.Equal(t, signature, sig)
			aux := file.File(ef)
			return &aux, nil
		}

		mv.Files.DeleteBySignatureFn = func(sig string) error {
			assert.Equal(t, signature, sig)
			return nil
		}

		mv.IDXKeys.DeleteByKeyFn = func(k string) error {
			assert.Equal(t, key, k)
			return nil
		}

		mv.Fs.RemoveFn = func(p string) error {
			assert.Equal(t, file.Path(path.Join(rootDir, "file"), signature), p)
			return nil
		}

		err := mv.V.DeleteFile(key)
		require.NoError(t, err)

		assert.True(t, mv.Fs.MkdirAllInvoked)
		assert.True(t, mv.Fs.RemoveInvoked)
		assert.True(t, mv.IDXKeys.FindByKeyInvoked)
		assert.True(t, mv.IDXKeys.DeleteByKeyInvoked)
		assert.True(t, mv.Files.DeleteBySignatureInvoked)
		assert.True(t, mv.Files.FindBySignatureInvoked)
	})
	t.Run("SuccessWithMultipleKeys", func(t *testing.T) {
		var (
			rootDir   = "root"
			mv        = mock.NewManageVolume(t, rootDir)
			key       = "expectedkey"
			signature = "123123123"
			ef        = file.File{
				Keys:      []string{key, "b"},
				Signature: signature,
			}
		)

		mv.IDXKeys.FindByKeyFn = func(k string) (*idxkey.IDXKey, error) {
			assert.Equal(t, key, k)
			return idxkey.New(k, signature), nil
		}

		mv.Files.FindBySignatureFn = func(sig string) (*file.File, error) {
			assert.Equal(t, signature, sig)
			aux := file.File(ef)
			return &aux, nil
		}

		mv.Files.CreateOrReplaceFn = func(f *file.File) error {
			assert.Equal(t, []string{"b"}, f.Keys)
			assert.Equal(t, signature, f.Signature)
			return nil
		}

		mv.IDXKeys.DeleteByKeyFn = func(k string) error {
			assert.Equal(t, key, k)
			return nil
		}

		err := mv.V.DeleteFile(key)
		require.NoError(t, err)

		assert.True(t, mv.Fs.MkdirAllInvoked)
		assert.False(t, mv.Fs.RemoveInvoked)
		assert.True(t, mv.IDXKeys.FindByKeyInvoked)
		assert.True(t, mv.IDXKeys.DeleteByKeyInvoked)
		assert.True(t, mv.Files.FindBySignatureInvoked)
		assert.False(t, mv.Files.DeleteBySignatureInvoked)
	})
}
