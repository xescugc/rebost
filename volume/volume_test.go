package volume_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/afero"
	"github.com/spf13/afero/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/idxkey"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/uow"
	"github.com/xescugc/rebost/volume"
)

func TestNew(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var suow uow.StartUnitOfWork
		var rootDir = "root"

		ctrl := gomock.NewController(t)

		files := mock.NewFileRepository(ctrl)
		idxkeys := mock.NewIDXKeyRepository(ctrl)
		fs := mock.NewFs(ctrl)
		rp := mock.NewReplicaPendentRepository(ctrl)
		idPath := path.Join(rootDir, "id")
		fh := mem.NewFileHandle(mem.CreateFile(idPath))

		defer ctrl.Finish()

		fs.EXPECT().MkdirAll(path.Join(rootDir, "file"), os.ModePerm).Return(nil)
		fs.EXPECT().MkdirAll(path.Join(rootDir, "tmps"), os.ModePerm).Return(nil)

		fs.EXPECT().Stat(idPath).Return(nil, os.ErrNotExist)
		fs.EXPECT().Create(idPath).Return(fh, nil)

		v, err := volume.New(rootDir, files, idxkeys, rp, fs, suow)
		require.NoError(t, err)
		assert.NotNil(t, v)
		defer v.Close()

		// As the FH is closed on the tests,
		// we have to open it again
		err = fh.Open()
		require.NoError(t, err)

		id, err := ioutil.ReadAll(fh)
		require.NoError(t, err)

		_, err = uuid.FromString(string(id))
		require.NoError(t, err, "Validates that it's a UUID")
	})
	t.Run("SuccessWithAlreadyID", func(t *testing.T) {
		var suow uow.StartUnitOfWork
		var rootDir = "root"

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		files := mock.NewFileRepository(ctrl)
		idxkeys := mock.NewIDXKeyRepository(ctrl)
		fs := mock.NewFs(ctrl)
		rp := mock.NewReplicaPendentRepository(ctrl)
		idPath := path.Join(rootDir, "id")
		fh := mem.NewFileHandle(mem.CreateFile(idPath))
		id := uuid.NewV4().String()
		_, err := io.WriteString(fh, id)
		require.NoError(t, err)
		_, err = fh.Seek(0, 0)
		require.NoError(t, err)

		fs.EXPECT().MkdirAll(path.Join(rootDir, "file"), os.ModePerm).Return(nil)
		fs.EXPECT().MkdirAll(path.Join(rootDir, "tmps"), os.ModePerm).Return(nil)

		fs.EXPECT().Stat(idPath).Return(nil, nil)
		fs.EXPECT().Open(idPath).Return(fh, nil)

		v, err := volume.New(rootDir, files, idxkeys, rp, fs, suow)
		require.NoError(t, err)
		assert.NotNil(t, v)
		defer v.Close()
		assert.Equal(t, id, v.ID())
	})
}

func TestCreateFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "root"
			rep      = 2
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			buff     = ioutil.NopCloser(bytes.NewBufferString("content of the file"))
			ef       = file.File{
				Keys:      []string{key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   2,
			}
			eik = idxkey.IDXKey{
				Key:   key,
				Value: ef.Signature,
			}
			mv          = mock.NewManageVolume(t, rootDir)
			eRepPendent = replica.Pendent{
				Key:       key,
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   2,
				VolumeID:  mv.V.ID(),
			}

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

		mv.ReplicaPendent.EXPECT().Create(ctx, gomock.Any()).Do(func(_ context.Context, rp *replica.Pendent) {
			_, err := uuid.FromString(string(rp.ID))
			require.NoError(t, err, "Validates that it's a UUID")
			rp.ID = ""
			assert.NotNil(t, rp.VolumeReplicaID)
			rp.VolumeReplicaID = nil
			assert.Equal(t, rp, &eRepPendent)
		}).Return(nil)

		err := mv.V.CreateFile(ctx, key, buff, rep)
		require.NoError(t, err)
	})
	t.Run("SuccessUpdateFileKey", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "root"
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			buff     = ioutil.NopCloser(bytes.NewBufferString("content of the file"))
			rep      = 2
			ef       = file.File{
				Keys:      []string{"b", key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   rep,
			}
			eik = idxkey.IDXKey{
				Key:   key,
				Value: ef.Signature,
			}
			mv          = mock.NewManageVolume(t, rootDir)
			eRepPendent = replica.Pendent{
				Key:       key,
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   2,
				VolumeID:  mv.V.ID(),
			}

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
			Replica:   rep,
		}, nil)

		mv.Files.EXPECT().CreateOrReplace(ctx, &ef).Return(nil)

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(nil, errors.New("not found"))

		mv.IDXKeys.EXPECT().CreateOrReplace(ctx, &eik).Return(nil)

		mv.ReplicaPendent.EXPECT().Create(ctx, gomock.Any()).Do(func(_ context.Context, rp *replica.Pendent) {
			_, err := uuid.FromString(string(rp.ID))
			require.NoError(t, err, "Validates that it's a UUID")
			rp.ID = ""
			assert.NotNil(t, rp.VolumeReplicaID)
			rp.VolumeReplicaID = nil
			assert.Equal(t, rp, &eRepPendent)
		}).Return(nil)

		err := mv.V.CreateFile(ctx, key, buff, rep)
		require.NoError(t, err)
	})
	t.Run("SuccessSame", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "root"
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			rep      = 2
			buff     = ioutil.NopCloser(bytes.NewBufferString("content of the file"))
			ef       = file.File{
				Keys:      []string{key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   rep,
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

		err := mv.V.CreateFile(ctx, key, buff, rep)
		require.NoError(t, err)
	})
	t.Run("SuccessRemoveFileKey", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "root"
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			buff     = ioutil.NopCloser(bytes.NewBufferString("content of the file"))
			rep      = 2
			ef       = file.File{
				Keys:      []string{key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   rep,
			}
			eik = idxkey.IDXKey{
				Key:   key,
				Value: ef.Signature,
			}
			foundFile = file.File{
				Keys:      []string{key, "b"},
				Signature: "123123123",
			}
			mv          = mock.NewManageVolume(t, rootDir)
			eRepPendent = replica.Pendent{
				Key:       key,
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   2,
				VolumeID:  mv.V.ID(),
			}

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

		mv.ReplicaPendent.EXPECT().Create(ctx, gomock.Any()).Do(func(_ context.Context, rp *replica.Pendent) {
			_, err := uuid.FromString(string(rp.ID))
			require.NoError(t, err, "Validates that it's a UUID")
			rp.ID = ""
			assert.NotNil(t, rp.VolumeReplicaID)
			rp.VolumeReplicaID = nil
			assert.Equal(t, rp, &eRepPendent)
		}).Return(nil)

		err := mv.V.CreateFile(ctx, key, buff, rep)
		require.NoError(t, err)
	})
	t.Run("SuccessRemoveFileKeyAndFile", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "root"
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			buff     = ioutil.NopCloser(bytes.NewBufferString("content of the file"))
			rep      = 2
			ef       = file.File{
				Keys:      []string{key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   rep,
			}
			eik = idxkey.IDXKey{
				Key:   key,
				Value: ef.Signature,
			}
			foundFile = file.File{
				Keys:      []string{key},
				Signature: "123123123",
			}
			mv          = mock.NewManageVolume(t, rootDir)
			eRepPendent = replica.Pendent{
				Key:       key,
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   2,
				VolumeID:  mv.V.ID(),
			}

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

		mv.ReplicaPendent.EXPECT().Create(ctx, gomock.Any()).Do(func(_ context.Context, rp *replica.Pendent) {
			_, err := uuid.FromString(string(rp.ID))
			require.NoError(t, err, "Validates that it's a UUID")
			rp.ID = ""
			assert.NotNil(t, rp.VolumeReplicaID)
			rp.VolumeReplicaID = nil
			assert.Equal(t, rp, &eRepPendent)
		}).Return(nil)

		err := mv.V.CreateFile(ctx, key, buff, rep)
		require.NoError(t, err)
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

func expectVolumeReplicaPendent(t *testing.T, v volume.Local, eRepPendent replica.Pendent) {
	t.Helper()

	toctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	select {
	case rp := <-v.ReplicaPendent():
		cancel()
		assert.Equal(t, rp, eRepPendent)
	case <-toctx.Done():
		assert.Fail(t, "The ReplicaPendent queue did not recieve any event in 2s")
	}
}
