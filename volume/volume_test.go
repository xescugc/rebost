package volume_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
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
	var suow uow.StartUnitOfWork
	var rootDir = "root"

	filesCtrl := gomock.NewController(t)
	idxKeysCtrl := gomock.NewController(t)
	fsCtrl := gomock.NewController(t)

	files := mock.NewFileRepository(filesCtrl)
	idxkeys := mock.NewIDXKeyRepository(idxKeysCtrl)
	fs := mock.NewFs(fsCtrl)

	defer filesCtrl.Finish()
	defer idxKeysCtrl.Finish()
	defer fsCtrl.Finish()

	fs.EXPECT().MkdirAll(gomock.Any(), os.ModePerm).DoAndReturn(func(p string, _ os.FileMode) error {
		switch p {
		case path.Join(rootDir, "file"):
			return nil
		case path.Join(rootDir, "tmps"):
			return nil
		default:
			return fmt.Errorf("test error for path: %q", p)
		}
	}).Times(2)

	v, err := volume.New(rootDir, files, idxkeys, fs, suow)
	require.NoError(t, err)
	assert.NotNil(t, v)
}

func TestCreateFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "root"
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			buff     = bytes.NewBufferString("content of the file")
			ef       = file.File{
				Keys:      []string{key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
			}
			eik = idxkey.IDXKey{
				Key:   key,
				Value: ef.Signature,
			}

			mv  = mock.NewManageVolume(t, rootDir)
			ctx = context.Background()
		)

		defer mv.Finish()

		mv.Fs.EXPECT().Create(gomock.Any()).DoAndReturn(func(p string) (afero.File, error) {
			assert.True(t, strings.HasPrefix(p, tmpsDir))
			_, tempuuid = path.Split(p)
			return mem.NewFileHandle(mem.CreateFile(p)), nil
		})

		dir, _ := path.Split(ef.Path(fileDir))
		mv.Fs.EXPECT().MkdirAll(dir, os.ModePerm).Return(nil)

		mv.Fs.EXPECT().Rename(gomock.Any(), ef.Path(fileDir)).Do(func(p string, _ string) {
			assert.Equal(t, path.Join(tmpsDir, tempuuid), p)
		}).Return(nil)

		mv.Files.EXPECT().FindBySignature(ctx, ef.Signature).Return(nil, errors.New("not found"))

		mv.Files.EXPECT().CreateOrReplace(ctx, &ef).Return(nil)

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(nil, errors.New("not found"))

		mv.IDXKeys.EXPECT().CreateOrReplace(ctx, &eik).Return(nil)

		f, err := mv.V.CreateFile(ctx, key, buff)
		require.NoError(t, err)
		assert.Equal(t, &ef, f)
	})
	t.Run("SuccessUpdateFileKey", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "root"
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			buff     = bytes.NewBufferString("content of the file")
			ef       = file.File{
				Keys:      []string{"b", key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
			}
			eik = idxkey.IDXKey{
				Key:   key,
				Value: ef.Signature,
			}

			mv  = mock.NewManageVolume(t, rootDir)
			ctx = context.Background()
		)

		defer mv.Finish()

		mv.Fs.EXPECT().Create(gomock.Any()).DoAndReturn(func(p string) (afero.File, error) {
			assert.True(t, strings.HasPrefix(p, tmpsDir))
			_, tempuuid = path.Split(p)
			return mem.NewFileHandle(mem.CreateFile(p)), nil
		})

		dir, _ := path.Split(ef.Path(fileDir))
		mv.Fs.EXPECT().MkdirAll(dir, os.ModePerm).Return(nil)

		mv.Fs.EXPECT().Rename(gomock.Any(), ef.Path(fileDir)).Do(func(p string, _ string) {
			assert.Equal(t, path.Join(tmpsDir, tempuuid), p)
		}).Return(nil)

		mv.Files.EXPECT().FindBySignature(ctx, ef.Signature).Return(&file.File{
			Keys:      []string{"b"},
			Signature: ef.Signature,
		}, nil)

		mv.Files.EXPECT().CreateOrReplace(ctx, &ef).Return(nil)

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(nil, errors.New("not found"))

		mv.IDXKeys.EXPECT().CreateOrReplace(ctx, &eik).Return(nil)

		f, err := mv.V.CreateFile(ctx, key, buff)
		require.NoError(t, err)

		assert.Equal(t, &ef, f)
	})
	t.Run("SuccessSame", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "root"
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			buff     = bytes.NewBufferString("content of the file")
			ef       = file.File{
				Keys:      []string{key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
			}

			mv  = mock.NewManageVolume(t, rootDir)
			ctx = context.Background()
		)

		defer mv.Finish()

		mv.Fs.EXPECT().Create(gomock.Any()).DoAndReturn(func(p string) (afero.File, error) {
			assert.True(t, strings.HasPrefix(p, tmpsDir))
			_, tempuuid = path.Split(p)
			return mem.NewFileHandle(mem.CreateFile(p)), nil
		})

		dir, _ := path.Split(ef.Path(fileDir))
		mv.Fs.EXPECT().MkdirAll(dir, os.ModePerm).Return(nil)

		mv.Fs.EXPECT().Rename(gomock.Any(), ef.Path(fileDir)).Do(func(p string, _ string) {
			assert.Equal(t, path.Join(tmpsDir, tempuuid), p)
		}).Return(nil)

		mv.Files.EXPECT().FindBySignature(ctx, ef.Signature).Return(&file.File{
			Keys:      ef.Keys,
			Signature: ef.Signature,
		}, nil)

		f, err := mv.V.CreateFile(ctx, key, buff)
		require.NoError(t, err)

		assert.Equal(t, &ef, f)
	})
	t.Run("SuccessRemoveFileKey", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "root"
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			buff     = bytes.NewBufferString("content of the file")
			ef       = file.File{
				Keys:      []string{key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
			}
			eik = idxkey.IDXKey{
				Key:   key,
				Value: ef.Signature,
			}
			foundFile = file.File{
				Keys:      []string{key, "b"},
				Signature: "123123123",
			}

			mv  = mock.NewManageVolume(t, rootDir)
			ctx = context.Background()
		)

		defer mv.Finish()

		mv.Fs.EXPECT().Create(gomock.Any()).DoAndReturn(func(p string) (afero.File, error) {
			assert.True(t, strings.HasPrefix(p, tmpsDir))
			_, tempuuid = path.Split(p)
			return mem.NewFileHandle(mem.CreateFile(p)), nil
		})

		dir, _ := path.Split(ef.Path(fileDir))
		mv.Fs.EXPECT().MkdirAll(dir, os.ModePerm).Return(nil)

		mv.Fs.EXPECT().Rename(gomock.Any(), ef.Path(fileDir)).Do(func(p string, _ string) {
			assert.Equal(t, path.Join(tmpsDir, tempuuid), p)
		}).Return(nil)

		mv.Files.EXPECT().FindBySignature(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, sig string) (*file.File, error) {
			if sig == ef.Signature {
				return nil, errors.New("not found")
			}
			return &file.File{
				Keys:      foundFile.Keys,
				Signature: foundFile.Signature,
			}, nil
		}).Times(2)

		mv.Files.EXPECT().CreateOrReplace(ctx, gomock.Any()).Do(func(_ context.Context, fl *file.File) {
			if fl.Signature == ef.Signature {
				assert.Equal(t, &ef, fl)
			} else {
				foundFile.Keys = []string{"b"}
				assert.Equal(t, &foundFile, fl)
			}
		}).Return(nil).Times(2)

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(idxkey.New(key, foundFile.Signature), nil)

		mv.IDXKeys.EXPECT().CreateOrReplace(ctx, &eik).Return(nil)

		f, err := mv.V.CreateFile(ctx, key, buff)
		require.NoError(t, err)

		assert.Equal(t, &ef, f)
	})
	t.Run("SuccessRemoveFileKeyAndFile", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "root"
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			buff     = bytes.NewBufferString("content of the file")
			ef       = file.File{
				Keys:      []string{key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
			}
			eik = idxkey.IDXKey{
				Key:   key,
				Value: ef.Signature,
			}
			foundFile = file.File{
				Keys:      []string{key},
				Signature: "123123123",
			}

			mv  = mock.NewManageVolume(t, rootDir)
			ctx = context.Background()
		)

		defer mv.Finish()

		mv.Fs.EXPECT().Create(gomock.Any()).DoAndReturn(func(p string) (afero.File, error) {
			assert.True(t, strings.HasPrefix(p, tmpsDir))
			_, tempuuid = path.Split(p)
			return mem.NewFileHandle(mem.CreateFile(p)), nil
		})

		dir, _ := path.Split(ef.Path(fileDir))
		mv.Fs.EXPECT().MkdirAll(dir, os.ModePerm).Return(nil)

		mv.Fs.EXPECT().Rename(gomock.Any(), ef.Path(fileDir)).Do(func(p string, _ string) {
			assert.Equal(t, path.Join(tmpsDir, tempuuid), p)
		}).Return(nil)

		mv.Files.EXPECT().FindBySignature(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, sig string) (*file.File, error) {
			if sig == ef.Signature {
				return nil, errors.New("not found")
			}
			return &file.File{
				Keys:      foundFile.Keys,
				Signature: foundFile.Signature,
			}, nil
		}).Times(2)

		mv.Files.EXPECT().CreateOrReplace(ctx, &ef).Return(nil)

		mv.Files.EXPECT().DeleteBySignature(ctx, foundFile.Signature).Return(nil)

		mv.Fs.EXPECT().Remove(foundFile.Path(fileDir)).Return(nil)

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(idxkey.New(key, foundFile.Signature), nil)

		mv.IDXKeys.EXPECT().DeleteByKey(ctx, key).Return(nil)

		mv.IDXKeys.EXPECT().CreateOrReplace(ctx, &eik).Return(nil)

		f, err := mv.V.CreateFile(ctx, key, buff)
		require.NoError(t, err)

		assert.Equal(t, &ef, f)
	})
}

func TestGetFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir   = "root"
			key       = "expectedkey"
			signature = "123123123"
			content   = "expectedcontent"
			fileDir   = path.Join(rootDir, "file")

			mv  = mock.NewManageVolume(t, rootDir)
			ctx = context.Background()
		)

		defer mv.Finish()

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(idxkey.New(key, signature), nil)

		mv.Fs.EXPECT().Open(file.Path(fileDir, signature)).DoAndReturn(func(p string) (afero.File, error) {
			tf := mem.NewFileHandle(mem.CreateFile(p))
			tf.WriteString(content)
			// After finishing writting, the "cursor" points to the end of
			// the content, so to be able to read all of it, has to be
			// set to the beggining, that's what the Seek does
			tf.Seek(0, 0)
			return tf, nil
		})

		ior, err := mv.V.GetFile(ctx, key)
		require.NoError(t, err)
		require.NotNil(t, ior)
		b, err := ioutil.ReadAll(ior)
		require.NoError(t, err)
		assert.Equal(t, content, string(b))
	})
	t.Run("NotFound", func(t *testing.T) {
		var (
			rootDir = "root"
			key     = "expectedkey"
			mv      = mock.NewManageVolume(t, rootDir)
			ctx     = context.Background()
		)

		defer mv.Finish()

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(nil, errors.New("not found"))

		_, err := mv.V.GetFile(ctx, key)
		assert.EqualError(t, err, errors.New("not found").Error())
	})
}

func TestHasFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir = "root"
			key     = "expectedkey"
			mv      = mock.NewManageVolume(t, rootDir)
			ctx     = context.Background()
		)

		defer mv.Finish()

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(idxkey.New(key, "not needed"), nil)

		ok, err := mv.V.HasFile(ctx, key)
		require.NoError(t, err)
		assert.True(t, ok)
	})
	t.Run("NotFound", func(t *testing.T) {
		var (
			rootDir = "root"
			key     = "expectedkey"
			ctx     = context.Background()
			mv      = mock.NewManageVolume(t, rootDir)
		)

		defer mv.Finish()

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(nil, errors.New("not found"))

		ok, err := mv.V.HasFile(ctx, key)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestDeleteFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir   = "root"
			key       = "expectedkey"
			signature = "123123123"
			ef        = file.File{
				Keys:      []string{key},
				Signature: signature,
			}
			fileDir = path.Join(rootDir, "file")

			ctx = context.Background()
			mv  = mock.NewManageVolume(t, rootDir)
		)

		defer mv.Finish()

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(idxkey.New(key, signature), nil)

		mv.Files.EXPECT().FindBySignature(ctx, signature).DoAndReturn(func(_ context.Context, sig string) (*file.File, error) {
			aux := file.File(ef)
			return &aux, nil
		})

		mv.Files.EXPECT().DeleteBySignature(ctx, signature).Return(nil)

		mv.IDXKeys.EXPECT().DeleteByKey(ctx, key).Return(nil)

		mv.Fs.EXPECT().Remove(file.Path(fileDir, signature)).Return(nil)

		err := mv.V.DeleteFile(ctx, key)
		require.NoError(t, err)
	})
	t.Run("SuccessWithMultipleKeys", func(t *testing.T) {
		var (
			rootDir   = "root"
			key       = "expectedkey"
			signature = "123123123"
			ef        = file.File{
				Keys:      []string{key, "b"},
				Signature: signature,
			}

			ctx = context.Background()
			mv  = mock.NewManageVolume(t, rootDir)
		)

		defer mv.Finish()

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(idxkey.New(key, signature), nil)

		mv.Files.EXPECT().FindBySignature(ctx, signature).DoAndReturn(func(_ context.Context, sig string) (*file.File, error) {
			aux := file.File(ef)
			return &aux, nil
		})

		mv.Files.EXPECT().CreateOrReplace(ctx, &file.File{Keys: []string{"b"}, Signature: signature}).Return(nil)

		mv.IDXKeys.EXPECT().DeleteByKey(ctx, key).Return(nil)

		err := mv.V.DeleteFile(ctx, key)
		require.NoError(t, err)
	})
}
