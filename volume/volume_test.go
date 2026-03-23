package volume_test

import (
	"bytes"
	"context"
	"errors"
	"io"
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
	"github.com/xescugc/rebost/idxttl"
	"github.com/xescugc/rebost/idxvolume"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/state"
	"github.com/xescugc/rebost/uow"
	"github.com/xescugc/rebost/volume"
)

func TestNew(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var rootDir = "/"

		ctrl := gomock.NewController(t)

		fs := mock.NewFs(ctrl)
		sr := mock.NewStateRepository(ctrl)
		idPath := path.Join(rootDir, "id")
		fh := mem.NewFileHandle(mem.CreateFile(idPath))

		uowFn := func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn) error {
			uw := mock.NewUnitOfWork(ctrl)
			uw.EXPECT().State().Return(sr).AnyTimes()
			return uowFn(ctx, uw)
		}

		defer ctrl.Finish()

		fs.EXPECT().MkdirAll(path.Join(rootDir, "file"), os.ModePerm).Return(nil)
		fs.EXPECT().MkdirAll(path.Join(rootDir, "tmps"), os.ModePerm).Return(nil)

		fs.EXPECT().Stat(idPath).Return(nil, os.ErrNotExist)
		fs.EXPECT().Create(idPath).Return(fh, nil)

		sr.EXPECT().Find(gomock.Any()).Return(&state.State{}, nil)
		sr.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

		v, err := volume.New(rootDir, fs, nil, uowFn)
		require.NoError(t, err)
		assert.NotNil(t, v)
		defer v.Close()

		// As the FH is closed on the tests,
		// we have to open it again
		err = fh.Open()
		require.NoError(t, err)

		id, err := io.ReadAll(fh)
		require.NoError(t, err)

		_, err = uuid.FromString(string(id))
		require.NoError(t, err, "Validates that it's a UUID")
	})
	t.Run("SuccessWithSize", func(t *testing.T) {
		var (
			rootDirWithSize = "/:20G"
			rootDir         = "/"
		)

		ctrl := gomock.NewController(t)

		fs := mock.NewFs(ctrl)
		sr := mock.NewStateRepository(ctrl)
		idPath := path.Join(rootDir, "id")
		fh := mem.NewFileHandle(mem.CreateFile(idPath))

		uowFn := func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn) error {
			uw := mock.NewUnitOfWork(ctrl)
			uw.EXPECT().State().Return(sr).AnyTimes()
			return uowFn(ctx, uw)
		}

		defer ctrl.Finish()

		fs.EXPECT().MkdirAll(path.Join(rootDir, "file"), os.ModePerm).Return(nil)
		fs.EXPECT().MkdirAll(path.Join(rootDir, "tmps"), os.ModePerm).Return(nil)

		fs.EXPECT().Stat(idPath).Return(nil, os.ErrNotExist)
		fs.EXPECT().Create(idPath).Return(fh, nil)

		sr.EXPECT().Find(gomock.Any()).Return(&state.State{}, nil)
		sr.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, s *state.State) error {
			assert.Equal(t, 21474836480, s.VolumeTotalSize)
			return nil
		})

		v, err := volume.New(rootDirWithSize, fs, nil, uowFn)
		require.NoError(t, err)
		assert.NotNil(t, v)
		defer v.Close()

		// As the FH is closed on the tests,
		// we have to open it again
		err = fh.Open()
		require.NoError(t, err)

		id, err := io.ReadAll(fh)
		require.NoError(t, err)

		_, err = uuid.FromString(string(id))
		require.NoError(t, err, "Validates that it's a UUID")
	})
	t.Run("SuccessWithAlreadyID", func(t *testing.T) {
		var rootDir = "/"

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fs := mock.NewFs(ctrl)
		sr := mock.NewStateRepository(ctrl)
		idPath := path.Join(rootDir, "id")
		fh := mem.NewFileHandle(mem.CreateFile(idPath))
		id := uuid.NewV4().String()

		_, err := io.WriteString(fh, id)
		require.NoError(t, err)

		_, err = fh.Seek(0, 0)
		require.NoError(t, err)

		uowFn := func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn) error {
			uw := mock.NewUnitOfWork(ctrl)
			uw.EXPECT().State().Return(sr).AnyTimes()
			return uowFn(ctx, uw)
		}

		fs.EXPECT().MkdirAll(path.Join(rootDir, "file"), os.ModePerm).Return(nil)
		fs.EXPECT().MkdirAll(path.Join(rootDir, "tmps"), os.ModePerm).Return(nil)

		fs.EXPECT().Stat(idPath).Return(nil, nil)
		fs.EXPECT().Open(idPath).Return(fh, nil)

		sr.EXPECT().Find(gomock.Any()).Return(&state.State{}, nil)
		sr.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

		v, err := volume.New(rootDir, fs, nil, uowFn)
		require.NoError(t, err)
		assert.NotNil(t, v)
		defer v.Close()
		assert.Equal(t, id, v.ID())
	})
	t.Run("Invalid size", func(t *testing.T) {
		var rootDir = "/:20potato"

		ctrl := gomock.NewController(t)

		fs := mock.NewFs(ctrl)

		uowFn := func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn) error {
			uw := mock.NewUnitOfWork(ctrl)
			return uowFn(ctx, uw)
		}

		defer ctrl.Finish()

		v, err := volume.New(rootDir, fs, nil, uowFn)
		assert.Equal(t, "byte quantity must be a positive integer with a unit of measurement like M, MB, MiB, G, GiB, or GB", err.Error())
		assert.Empty(t, v)
	})
}

func TestCreateFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "/"
			mv       = newManageVolume(t, rootDir)
			rep      = 2
			ttl      = 2 * time.Minute
			ca       = time.Now()
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			buff     = io.NopCloser(bytes.NewBufferString("content of the file"))
			ef       = file.File{
				Keys:      []string{key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   2,
				VolumeIDs: []string{mv.V.ID()},
				Size:      19,
				TTL:       ttl,
				CreatedAt: ca,
			}
			eik = idxkey.IDXKey{
				Key:   key,
				Value: ef.Signature,
			}
			eittl = idxttl.New(ef.ExpiresAt(), "1", ef.Signature)

			ctx = context.Background()
		)

		defer mv.Finish()

		mv.IDXTTLs.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

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

		mv.Files.EXPECT().FindBySignature(ctx, ef.Signature).Return(nil, errors.New("not found")).Times(2)

		mv.Files.EXPECT().CreateOrReplace(ctx, &ef).Return(nil)

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(nil, errors.New("not found"))

		mv.IDXKeys.EXPECT().CreateOrReplace(ctx, &eik).Return(nil)

		mv.IDXTTLs.EXPECT().Find(ctx, ef.ExpiresAt()).Return(idxttl.New(ef.ExpiresAt(), eittl.Signatures[0]), nil)
		mv.IDXTTLs.EXPECT().CreateOrReplace(ctx, eittl).Return(nil)

		mv.Replicas.EXPECT().Create(ctx, gomock.Any()).Do(
			func(_ context.Context, rp *replica.Replica) error {
				assert.Equal(t, mv.V.ID(), rp.VolumeID)
				assert.Equal(t, key, rp.Key)
				assert.Equal(t, ef.Signature, rp.Signature)
				assert.Equal(t, rep-1, rp.Count)
				assert.Equal(t, rep, rp.OriginalCount)
				return nil
			},
		).Return(nil)

		expectUpdateState(t, mv, ctx, ef.Size)

		err := mv.V.CreateFile(ctx, key, buff, rep, ttl, ca)
		require.NoError(t, err)
	})
	t.Run("SuccessUpdateFileKey", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "/"
			mv       = newManageVolume(t, rootDir)
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			buff     = io.NopCloser(bytes.NewBufferString("content of the file"))
			rep      = 2
			ttl      = 2 * time.Minute
			ca       = time.Now()
			ef       = file.File{
				Keys:      []string{"b", key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   rep,
				VolumeIDs: []string{mv.V.ID()},
				TTL:       ttl,
				CreatedAt: ca,
			}
			eik = idxkey.IDXKey{
				Key:   key,
				Value: ef.Signature,
			}
			eittl = idxttl.New(ef.ExpiresAt(), "1", ef.Signature)

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
			TTL:       ttl,
			CreatedAt: ca,
		}, nil).Times(2)

		mv.Files.EXPECT().CreateOrReplace(ctx, &ef).Return(nil)

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(nil, errors.New("not found"))

		mv.IDXKeys.EXPECT().CreateOrReplace(ctx, &eik).Return(nil)

		mv.IDXTTLs.EXPECT().Find(ctx, ef.ExpiresAt()).Return(idxttl.New(ef.ExpiresAt(), eittl.Signatures[0]), nil)
		mv.IDXTTLs.EXPECT().CreateOrReplace(ctx, eittl).Return(nil)

		mv.Replicas.EXPECT().Create(ctx, gomock.Any()).Do(
			func(_ context.Context, rp *replica.Replica) error {
				assert.Equal(t, mv.V.ID(), rp.VolumeID)
				assert.Equal(t, key, rp.Key)
				assert.Equal(t, ef.Signature, rp.Signature)
				assert.Equal(t, rep-1, rp.Count)
				assert.Equal(t, rep, rp.OriginalCount)
				return nil
			},
		).Return(nil)

		err := mv.V.CreateFile(ctx, key, buff, rep, ttl, ca)
		require.NoError(t, err)
	})
	t.Run("SuccessSame", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "/"
			mv       = newManageVolume(t, rootDir)
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			rep      = 2
			ttl      = 2 * time.Minute
			ca       = time.Now()
			buff     = io.NopCloser(bytes.NewBufferString("content of the file"))
			ef       = file.File{
				Keys:      []string{key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   rep,
				VolumeIDs: []string{mv.V.ID()},
				TTL:       ttl,
				CreatedAt: ca,
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
			Keys:      ef.Keys,
			Signature: ef.Signature,
		}, nil).Times(2)

		err := mv.V.CreateFile(ctx, key, buff, rep, ttl, ca)
		require.NoError(t, err)
	})
	t.Run("SuccessRemoveFileKey", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "/"
			mv       = newManageVolume(t, rootDir)
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			buff     = io.NopCloser(bytes.NewBufferString("content of the file"))
			rep      = 2
			ttl      = 2 * time.Minute
			ca       = time.Now()
			ef       = file.File{
				Keys:      []string{key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   rep,
				VolumeIDs: []string{mv.V.ID()},
				Size:      19,
				TTL:       ttl,
				CreatedAt: ca,
			}
			eik = idxkey.IDXKey{
				Key:   key,
				Value: ef.Signature,
			}
			eittl     = idxttl.New(ef.ExpiresAt(), "1", ef.Signature)
			foundFile = file.File{
				Keys:      []string{key, "b"},
				Signature: "123123123",
				VolumeIDs: []string{mv.V.ID()},
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
				VolumeIDs: []string{mv.V.ID()},
			}, nil
		}).Times(3)

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

		mv.IDXTTLs.EXPECT().Find(ctx, ef.ExpiresAt()).Return(idxttl.New(ef.ExpiresAt(), eittl.Signatures[0]), nil)
		mv.IDXTTLs.EXPECT().CreateOrReplace(ctx, eittl).Return(nil)

		mv.Replicas.EXPECT().Create(ctx, gomock.Any()).Do(
			func(_ context.Context, rp *replica.Replica) error {
				assert.Equal(t, mv.V.ID(), rp.VolumeID)
				assert.Equal(t, key, rp.Key)
				assert.Equal(t, ef.Signature, rp.Signature)
				assert.Equal(t, rep-1, rp.Count)
				assert.Equal(t, rep, rp.OriginalCount)
				return nil
			},
		).Return(nil)

		expectUpdateState(t, mv, ctx, ef.Size)

		err := mv.V.CreateFile(ctx, key, buff, rep, ttl, ca)
		require.NoError(t, err)
	})
	t.Run("SuccessRemoveFileKeyAndFile", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "/"
			mv       = newManageVolume(t, rootDir)
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			buff     = io.NopCloser(bytes.NewBufferString("content of the file"))
			rep      = 2
			ttl      = 2 * time.Minute
			ca       = time.Now()
			ef       = file.File{
				Keys:      []string{key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   rep,
				VolumeIDs: []string{mv.V.ID()},
				Size:      19,
				TTL:       ttl,
				CreatedAt: ca,
			}
			eik = idxkey.IDXKey{
				Key:   key,
				Value: ef.Signature,
			}
			eittl     = idxttl.New(ef.ExpiresAt(), "1", ef.Signature)
			foundFile = file.File{
				Keys:      []string{key},
				Signature: "123123123",
				VolumeIDs: []string{mv.V.ID()},
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
		}).Times(3)

		mv.Files.EXPECT().CreateOrReplace(ctx, &ef).Return(nil)

		mv.Files.EXPECT().DeleteBySignature(ctx, foundFile.Signature).Return(nil)

		mv.Fs.EXPECT().Remove(foundFile.Path(fileDir)).Return(nil)

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(idxkey.New(key, foundFile.Signature), nil)

		mv.IDXKeys.EXPECT().DeleteByKey(ctx, key).Return(nil)

		mv.IDXKeys.EXPECT().CreateOrReplace(ctx, &eik).Return(nil)

		mv.IDXTTLs.EXPECT().Find(ctx, ef.ExpiresAt()).Return(idxttl.New(ef.ExpiresAt(), eittl.Signatures[0]), nil)
		mv.IDXTTLs.EXPECT().CreateOrReplace(ctx, eittl).Return(nil)

		mv.Replicas.EXPECT().Create(ctx, gomock.Any()).Do(
			func(_ context.Context, rp *replica.Replica) error {
				assert.Equal(t, mv.V.ID(), rp.VolumeID)
				assert.Equal(t, key, rp.Key)
				assert.Equal(t, ef.Signature, rp.Signature)
				assert.Equal(t, rep-1, rp.Count)
				assert.Equal(t, rep, rp.OriginalCount)
				return nil
			},
		).Return(nil)

		expectUpdateState(t, mv, ctx, ef.Size)

		err := mv.V.CreateFile(ctx, key, buff, rep, ttl, ca)
		require.NoError(t, err)
	})
	t.Run("SuccessWithNoReplica", func(t *testing.T) {
		var (
			tempuuid string
			rootDir  = "/"
			mv       = newManageVolume(t, rootDir)
			rep      = 1
			ttl      = 2 * time.Minute
			ca       = time.Now()
			tmpsDir  = path.Join(rootDir, "tmps")
			fileDir  = path.Join(rootDir, "file")
			key      = "expectedkey"
			buff     = io.NopCloser(bytes.NewBufferString("content of the file"))
			ef       = file.File{
				Keys:      []string{key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   1,
				VolumeIDs: []string{mv.V.ID()},
				Size:      19,
				TTL:       ttl,
				CreatedAt: ca,
			}
			eik = idxkey.IDXKey{
				Key:   key,
				Value: ef.Signature,
			}
			eittl = idxttl.New(ef.ExpiresAt(), "1", ef.Signature)

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

		mv.Files.EXPECT().FindBySignature(ctx, ef.Signature).Return(nil, errors.New("not found")).Times(2)

		mv.Files.EXPECT().CreateOrReplace(ctx, &ef).Return(nil)

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(nil, errors.New("not found"))

		mv.IDXKeys.EXPECT().CreateOrReplace(ctx, &eik).Return(nil)

		mv.IDXTTLs.EXPECT().Find(ctx, ef.ExpiresAt()).Return(idxttl.New(ef.ExpiresAt(), eittl.Signatures[0]), nil)
		mv.IDXTTLs.EXPECT().CreateOrReplace(ctx, eittl).Return(nil)

		expectUpdateState(t, mv, ctx, ef.Size)

		err := mv.V.CreateFile(ctx, key, buff, rep, ttl, ca)
		require.NoError(t, err)
	})
	t.Run("FailsForSize", func(t *testing.T) {
		var (
			rootDir = "/"
			mv      = newManageVolume(t, rootDir)
			rep     = 2
			ttl     = 2 * time.Minute
			ca      = time.Now()
			tmpsDir = path.Join(rootDir, "tmps")
			key     = "expectedkey"
			buff    = io.NopCloser(bytes.NewBufferString("content of the file"))
			ef      = file.File{
				Keys:      []string{key},
				Signature: "e7e8c72d1167454b76a610074fed244be0935298",
				Replica:   2,
				VolumeIDs: []string{mv.V.ID()},
				Size:      19,
			}

			ctx = context.Background()
		)

		defer mv.Finish()

		mv.Fs.EXPECT().Create(gomock.Any()).DoAndReturn(func(p string) (afero.File, error) {
			assert.True(t, strings.HasPrefix(p, tmpsDir))
			return mem.NewFileHandle(mem.CreateFile(p)), nil
		})

		dir, _ := path.Split(ef.Path(path.Join(rootDir, "file")))
		mv.Fs.EXPECT().MkdirAll(dir, os.ModePerm).Return(nil)

		mv.Files.EXPECT().FindBySignature(ctx, ef.Signature).Return(nil, errors.New("not found"))

		dbs := state.State{
			SystemTotalSize: 2000,
			SystemUsedSize:  100,
			VolumeTotalSize: 1,
			VolumeUsedSize:  0,
		}

		mv.State.EXPECT().Find(ctx).Return(&dbs, nil)

		mv.Fs.EXPECT().Remove(gomock.Any()).Times(1).Return(nil)

		err := mv.V.CreateFile(ctx, key, buff, rep, ttl, ca)
		assert.True(t, errors.Is(err, volume.ErrNoSpace))
	})
}

func TestCreateFile_CopyError(t *testing.T) {
	t.Run("ErrorPropagated", func(t *testing.T) {
		var (
			rootDir = "/"
			mv      = newManageVolume(t, rootDir)
			ctx     = context.Background()
			tmpsDir = path.Join(rootDir, "tmps")
			copyErr = errors.New("copy failed")
		)

		defer mv.Finish()

		// errReader returns an error mid-stream
		errRC := io.NopCloser(&errReader{err: copyErr})

		mv.Fs.EXPECT().Create(gomock.Any()).DoAndReturn(func(p string) (afero.File, error) {
			assert.True(t, strings.HasPrefix(p, tmpsDir))
			return mem.NewFileHandle(mem.CreateFile(p)), nil
		})

		err := mv.V.CreateFile(ctx, "key", errRC, 1, 0, time.Time{})
		require.Error(t, err)
	})
}

// errReader is an io.Reader that always returns an error.
type errReader struct {
	err error
}

func (e *errReader) Read(_ []byte) (int, error) {
	return 0, e.err
}

func TestGetFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir   = "/"
			key       = "expectedkey"
			signature = "123123123"
			content   = "expectedcontent"
			fileDir   = path.Join(rootDir, "file")

			mv  = newManageVolume(t, rootDir)
			ctx = context.Background()
		)

		defer mv.Finish()

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(idxkey.New(key, signature), nil)

		mv.Fs.EXPECT().Open(file.Path(fileDir, signature)).DoAndReturn(func(p string) (afero.File, error) {
			tf := mem.NewFileHandle(mem.CreateFile(p))
			tf.WriteString(content)
			// After finishing writing, the "cursor" points to the end of
			// the content, so to be able to read all of it, has to be
			// set to the beggining, that's what the Seek does
			tf.Seek(0, 0)
			return tf, nil
		})

		ior, _, err := mv.V.GetFile(ctx, key)
		require.NoError(t, err)
		require.NotNil(t, ior)
		b, err := io.ReadAll(ior)
		require.NoError(t, err)
		assert.Equal(t, content, string(b))
	})
	t.Run("NotFound", func(t *testing.T) {
		var (
			rootDir = "/"
			key     = "expectedkey"
			mv      = newManageVolume(t, rootDir)
			ctx     = context.Background()
		)

		defer mv.Finish()

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(nil, errors.New("not found"))

		_, _, err := mv.V.GetFile(ctx, key)
		assert.EqualError(t, err, errors.New("not found").Error())
	})
	t.Run("StatError", func(t *testing.T) {
		var (
			rootDir   = "/"
			key       = "expectedkey"
			signature = "123123123"
			fileDir   = path.Join(rootDir, "file")
			statErr   = errors.New("stat failed")

			mv  = newManageVolume(t, rootDir)
			ctx = context.Background()
		)

		defer mv.Finish()

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(idxkey.New(key, signature), nil)

		mv.Fs.EXPECT().Open(file.Path(fileDir, signature)).DoAndReturn(func(p string) (afero.File, error) {
			// Return a file handle backed by a statErrorFile so Stat() fails
			return &statErrorFile{File: mem.NewFileHandle(mem.CreateFile(p)), statErr: statErr}, nil
		})

		fh, size, err := mv.V.GetFile(ctx, key)
		require.Error(t, err)
		assert.Nil(t, fh)
		assert.Equal(t, int64(-1), size)
	})
}

// statErrorFile wraps afero.File and returns an error from Stat.
type statErrorFile struct {
	afero.File
	statErr error
}

func (s *statErrorFile) Stat() (os.FileInfo, error) {
	return nil, s.statErr
}

func TestHasFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir = "/"
			key     = "expectedkey"
			mv      = newManageVolume(t, rootDir)
			ctx     = context.Background()
		)

		defer mv.Finish()

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(idxkey.New(key, "not needed"), nil)

		vid, ok, err := mv.V.HasFile(ctx, key)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, mv.V.ID(), vid)
	})
	t.Run("NotFound", func(t *testing.T) {
		var (
			rootDir = "/"
			key     = "expectedkey"
			ctx     = context.Background()
			mv      = newManageVolume(t, rootDir)
		)

		defer mv.Finish()

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(nil, errors.New("not found"))

		vid, ok, err := mv.V.HasFile(ctx, key)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, "", vid)
	})
}

func TestDeleteFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir   = "/"
			key       = "expectedkey"
			signature = "123123123"
			ef        = file.File{
				Keys:      []string{key},
				Signature: signature,
				Size:      19,
			}
			fileDir = path.Join(rootDir, "file")

			ctx = context.Background()
			mv  = newManageVolume(t, rootDir)
		)

		defer mv.Finish()

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(idxkey.New(key, signature), nil)

		mv.Files.EXPECT().FindBySignature(ctx, signature).DoAndReturn(func(_ context.Context, sig string) (*file.File, error) {
			aux := file.File(ef)
			return &aux, nil
		})

		mv.Files.EXPECT().DeleteBySignature(ctx, signature).Return(nil)
		mv.Replicas.EXPECT().Delete(ctx, &replica.Replica{VolumeReplicaID: []byte(signature)}).Return(nil)

		mv.IDXKeys.EXPECT().DeleteByKey(ctx, key).Return(nil)

		mv.Fs.EXPECT().Remove(file.Path(fileDir, signature)).Return(nil)

		expectUpdateState(t, mv, ctx, -ef.Size)

		err := mv.V.DeleteFile(ctx, key)
		require.NoError(t, err)
	})
	t.Run("SuccessWithMultipleKeys", func(t *testing.T) {
		var (
			rootDir   = "/"
			key       = "expectedkey"
			signature = "123123123"
			ef        = file.File{
				Keys:      []string{key, "b"},
				Signature: signature,
			}

			ctx = context.Background()
			mv  = newManageVolume(t, rootDir)
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

func TestNextReplica(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir = "/"
			ctx     = context.Background()
			mv      = newManageVolume(t, rootDir)
			rp      = &replica.Replica{
				ID: "1",
			}
		)
		defer mv.Finish()

		mv.Replicas.EXPECT().First(ctx).Return(rp, nil)

		dbrep, err := mv.V.NextReplica(ctx)
		require.NoError(t, err)
		assert.Equal(t, rp, dbrep)
	})
}

func TestDeleteReplica(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir = "/"
			ctx     = context.Background()
			mv      = newManageVolume(t, rootDir)
			rp      = &replica.Replica{
				ID: "1",
			}
		)
		defer mv.Finish()

		mv.Replicas.EXPECT().Delete(ctx, rp).Return(nil)

		err := mv.V.DeleteReplica(ctx, rp)
		require.NoError(t, err)
	})
}

func TestUpdateReplica(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir = "/"
			ctx     = context.Background()
			mv      = newManageVolume(t, rootDir)
			rp      = &replica.Replica{
				ID:            "1",
				Count:         2,
				OriginalCount: 3,
				Signature:     "sig",
				VolumeID:      mv.V.ID(),
			}
			createRP = &replica.Replica{
				ID:            "1",
				Count:         1,
				OriginalCount: 3,
				Signature:     "sig",
				VolumeID:      mv.V.ID(),
			}
			findFile = &file.File{
				Signature: "sig",
			}
			updateFile = &file.File{
				Signature: "sig",
				Replica:   3,
				VolumeIDs: []string{"1"},
			}
		)
		defer mv.Finish()

		mv.Files.EXPECT().FindBySignature(ctx, rp.Signature).Return(findFile, nil)
		mv.Files.EXPECT().CreateOrReplace(ctx, updateFile).Return(nil)
		mv.IDXVolumes.EXPECT().FindByVolumeID(ctx, "1").Return(nil, errors.New("not found"))
		mv.IDXVolumes.EXPECT().CreateOrReplace(ctx, idxvolume.New("1", []string{findFile.Signature})).Return(nil)
		mv.Replicas.EXPECT().Delete(ctx, rp).Return(nil)
		mv.Replicas.EXPECT().Create(ctx, createRP).Return(nil)

		err := mv.V.UpdateReplica(ctx, rp, "1")
		require.NoError(t, err)
	})
	t.Run("SuccessWithNoMoreReplicas", func(t *testing.T) {
		var (
			rootDir = "/"
			ctx     = context.Background()
			mv      = newManageVolume(t, rootDir)
			rp      = &replica.Replica{
				ID:            "1",
				Count:         1,
				OriginalCount: 2,
				Signature:     "sig",
				VolumeID:      mv.V.ID(),
			}
			findFile = &file.File{
				Signature: "sig",
			}
			updateFile = &file.File{
				Signature: "sig",
				Replica:   2,
				VolumeIDs: []string{"1"},
			}
		)
		defer mv.Finish()

		mv.Files.EXPECT().FindBySignature(ctx, rp.Signature).Return(findFile, nil)
		mv.Files.EXPECT().CreateOrReplace(ctx, updateFile).Return(nil)
		mv.IDXVolumes.EXPECT().FindByVolumeID(ctx, "1").Return(nil, errors.New("not found"))
		mv.IDXVolumes.EXPECT().CreateOrReplace(ctx, idxvolume.New("1", []string{findFile.Signature})).Return(nil)
		mv.Replicas.EXPECT().Delete(ctx, rp).Return(nil)

		err := mv.V.UpdateReplica(ctx, rp, "1")
		require.NoError(t, err)
	})
	t.Run("ErrorWithNoReplica", func(t *testing.T) {
		var (
			rootDir = "/"
			ctx     = context.Background()
			mv      = newManageVolume(t, rootDir)
		)
		defer mv.Finish()

		err := mv.V.UpdateReplica(ctx, nil, "1")
		assert.EqualError(t, err, "the replica is required")
	})
	t.Run("ErrorWithNoSignatureOrKey", func(t *testing.T) {
		var (
			rootDir = "/"
			ctx     = context.Background()
			mv      = newManageVolume(t, rootDir)
		)
		defer mv.Finish()

		err := mv.V.UpdateReplica(ctx, &replica.Replica{}, "1")
		assert.EqualError(t, err, "the replica Signature is required")
	})
	t.Run("ErrorWithNoOriginalCount", func(t *testing.T) {
		var (
			rootDir = "/"
			ctx     = context.Background()
			mv      = newManageVolume(t, rootDir)
		)
		defer mv.Finish()

		err := mv.V.UpdateReplica(ctx, &replica.Replica{Signature: "key"}, "1")
		assert.EqualError(t, err, "the replica OriginalCount is required")
	})
}

func TestUpdateFileReplica(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir  = "/"
			ctx      = context.Background()
			mv       = newManageVolume(t, rootDir)
			findFile = &file.File{
				Keys:      []string{"file-key"},
				Signature: "sig",
			}
			kv = &idxkey.IDXKey{
				Key:   findFile.Keys[0],
				Value: findFile.Signature,
			}
			rep        = 3
			vids       = []string{mv.V.ID(), "2"}
			updateFile = &file.File{
				Keys:      findFile.Keys,
				Signature: findFile.Signature,
				Replica:   rep,
				VolumeIDs: vids,
			}
		)
		defer mv.Finish()

		mv.IDXKeys.EXPECT().FindByKey(ctx, kv.Key).Return(kv, nil)
		mv.Files.EXPECT().FindBySignature(ctx, kv.Value).Return(findFile, nil)
		mv.IDXVolumes.EXPECT().FindByVolumeID(ctx, vids[1]).Return(idxvolume.New(vids[1], []string{"other"}), nil)
		mv.IDXVolumes.EXPECT().CreateOrReplace(ctx, idxvolume.New(vids[1], []string{"other", findFile.Signature})).Return(nil)
		mv.Files.EXPECT().CreateOrReplace(ctx, updateFile).Return(nil)

		err := mv.V.UpdateFileReplica(ctx, findFile.Keys[0], vids, rep)
		require.NoError(t, err)
	})
	t.Run("ErrorRequireVolumeIDOnList", func(t *testing.T) {
		var (
			rootDir  = "/"
			ctx      = context.Background()
			mv       = newManageVolume(t, rootDir)
			findFile = &file.File{
				Keys:      []string{"file-key"},
				Signature: "sig",
			}
			rep  = 3
			vids = []string{"1", "2"}
		)
		defer mv.Finish()

		err := mv.V.UpdateFileReplica(ctx, findFile.Keys[0], vids, rep)
		assert.EqualError(t, err, "the volume ID has to be on the list of volume")
	})
}

func TestSynchronizeReplicas(t *testing.T) {
	t.Run("SuccessBeingOwner", func(t *testing.T) {
		var (
			rootDir  = "/"
			ctx      = context.Background()
			mv       = newManageVolume(t, rootDir)
			findFile = &file.File{
				Keys:      []string{"file-key", "file-key-2"},
				Signature: "sig",
				VolumeIDs: []string{mv.V.ID(), "b"},
				Replica:   4,
			}
			rep = &replica.Replica{
				Key:           findFile.Keys[0],
				Count:         3,
				OriginalCount: 4,
				Signature:     findFile.Signature,
				VolumeID:      mv.V.ID(),
				VolumeIDs:     []string{mv.V.ID()},
			}
			vid = "b"
		)
		defer mv.Finish()

		mv.IDXVolumes.EXPECT().
			FindByVolumeID(ctx, vid).Return(&idxvolume.IDXVolume{VolumeID: vid, Signatures: []string{findFile.Signature}}, nil)
		mv.Files.EXPECT().FindBySignature(ctx, findFile.Signature).Return(findFile, nil)
		mv.Replicas.EXPECT().Create(ctx, gomock.Any()).Do(
			func(_ context.Context, rp *replica.Replica) error {
				assert.NotEmpty(t, rp.ID)
				rp.ID = rep.ID

				assert.Equal(t, rep, rp)
				return nil
			},
		).Return(nil)

		err := mv.V.SynchronizeReplicas(ctx, vid)
		require.NoError(t, err)
	})
	t.Run("SuccessNotBeingOwner", func(t *testing.T) {
		var (
			rootDir  = "/"
			ctx      = context.Background()
			mv       = newManageVolume(t, rootDir)
			findFile = &file.File{
				Keys:      []string{"file-key", "file-key-2"},
				Signature: "sig",
				VolumeIDs: []string{"b", mv.V.ID(), "c"},
				Replica:   4,
			}
			vid = "c"
		)
		defer mv.Finish()

		mv.IDXVolumes.EXPECT().
			FindByVolumeID(ctx, vid).Return(&idxvolume.IDXVolume{VolumeID: vid, Signatures: []string{findFile.Signature}}, nil)
		mv.Files.EXPECT().FindBySignature(ctx, findFile.Signature).Return(findFile, nil)

		err := mv.V.SynchronizeReplicas(ctx, vid)
		require.NoError(t, err)
	})
	t.Run("NoPanicWhenVolumeIDsBecomesEmpty", func(t *testing.T) {
		var (
			rootDir  = "/"
			ctx      = context.Background()
			mv       = newManageVolume(t, rootDir)
			vid      = "only-vid"
			findFile = &file.File{
				Keys:      []string{"file-key"},
				Signature: "sig",
				VolumeIDs: []string{vid},
				Replica:   3,
			}
		)
		defer mv.Finish()

		mv.IDXVolumes.EXPECT().
			FindByVolumeID(ctx, vid).Return(&idxvolume.IDXVolume{VolumeID: vid, Signatures: []string{findFile.Signature}}, nil)
		mv.Files.EXPECT().FindBySignature(ctx, findFile.Signature).Return(findFile, nil)

		// Should not panic even though VolumeIDs becomes empty after DeleteVolumeID
		err := mv.V.SynchronizeReplicas(ctx, vid)
		require.NoError(t, err)
	})
}

func TestGetState(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir = "/"

			dbs = state.State{
				VolumeUsedSize: 10,
			}

			mv  = newManageVolume(t, rootDir)
			ctx = context.Background()
		)

		defer mv.Finish()

		mv.State.EXPECT().Find(ctx).Return(&dbs, nil)

		rs, err := mv.V.GetState(ctx)
		require.NoError(t, err)
		assert.Equal(t, rs, &dbs)
	})
	t.Run("NotFound", func(t *testing.T) {
		var (
			rootDir = "/"
			key     = "expectedkey"
			mv      = newManageVolume(t, rootDir)
			ctx     = context.Background()
		)

		defer mv.Finish()

		mv.IDXKeys.EXPECT().FindByKey(ctx, key).Return(nil, errors.New("not found"))

		_, _, err := mv.V.GetFile(ctx, key)
		assert.EqualError(t, err, errors.New("not found").Error())
	})
}

func TestReset(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			rootDir = "/"
			mv      = newManageVolume(t, rootDir)
			ctx     = context.Background()
			fileDir = path.Join(rootDir, "file")
			tempDir = path.Join(rootDir, "tmps")
			idPath  = path.Join(rootDir, "id")
			fh      = mem.NewFileHandle(mem.CreateFile(idPath))
		)
		defer mv.Finish()

		mv.Files.EXPECT().DeleteAll(ctx).Return(nil)
		mv.IDXKeys.EXPECT().DeleteAll(ctx).Return(nil)
		mv.Replicas.EXPECT().DeleteAll(ctx).Return(nil)
		mv.IDXVolumes.EXPECT().DeleteAll(ctx).Return(nil)
		mv.Fs.EXPECT().RemoveAll(fileDir).Return(nil)
		mv.Fs.EXPECT().RemoveAll(tempDir).Return(nil)
		mv.State.EXPECT().DeleteAll(ctx).Return(nil)
		mv.Fs.EXPECT().Remove(idPath).Return(nil)

		mv.Fs.EXPECT().Create(idPath).Return(fh, nil)

		mv.State.EXPECT().Find(ctx).Return(&state.State{}, nil)
		mv.State.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		err := mv.V.Reset(ctx)
		require.NoError(t, err)

		// As the FH is closed on the tests,
		// we have to open it again
		err = fh.Open()
		require.NoError(t, err)

		id, err := io.ReadAll(fh)
		require.NoError(t, err)

		_, err = uuid.FromString(string(id))
		require.NoError(t, err, "Validates that it's a UUID")
	})
}

func TestLocal_PrepareForDrain(t *testing.T) {
	t.Run("CreatesReplicaJobWhenNotEnoughExternalReplicas", func(t *testing.T) {
		var (
			rootDir = "/"
			ctx     = context.Background()
			mv      = newManageVolume(t, rootDir)
			// File with only the local volume ID (no external replicas) and replica=3
			f = &file.File{
				Keys:      []string{"mykey"},
				Signature: "abc123",
				Replica:   3,
				VolumeIDs: []string{mv.V.ID()},
			}
		)
		defer mv.Finish()

		mv.Files.EXPECT().All(ctx).Return([]*file.File{f}, nil)

		mv.Replicas.EXPECT().Create(ctx, gomock.Any()).Do(
			func(_ context.Context, rp *replica.Replica) error {
				assert.Equal(t, f.Keys[0], rp.Key)
				assert.Equal(t, f.Signature, rp.Signature)
				assert.Equal(t, mv.V.ID(), rp.VolumeID)
				// VolumeIDs must exclude self so downstream UpdateFileReplica
				// never propagates the draining node's ID to other holders.
				assert.Empty(t, rp.VolumeIDs)
				// externalCount=0, Replica=3 → Count=3
				assert.Equal(t, 3, rp.Count)
				assert.Equal(t, 3, rp.OriginalCount)
				assert.Equal(t, f.TTL, rp.TTL)
				assert.Equal(t, f.CreatedAt, rp.CreatedAt)
				return nil
			},
		).Return(nil)

		err := mv.V.PrepareForDrain(ctx)
		require.NoError(t, err)
	})

	t.Run("NoReplicaJobWhenEnoughExternalReplicas", func(t *testing.T) {
		var (
			rootDir = "/"
			ctx     = context.Background()
			mv      = newManageVolume(t, rootDir)
			// File with local + 3 external replicas, replica=3 → 3 external copies already cover the goal
			f = &file.File{
				Keys:      []string{"mykey"},
				Signature: "abc123",
				Replica:   3,
				VolumeIDs: []string{mv.V.ID(), "remote1", "remote2", "remote3"},
			}
		)
		defer mv.Finish()

		mv.Files.EXPECT().All(ctx).Return([]*file.File{f}, nil)
		// No Replicas.Create call expected: externalCount(3) >= Replica(3)

		err := mv.V.PrepareForDrain(ctx)
		require.NoError(t, err)
	})

	t.Run("CreatesReplicaJobForNonOwnedFileWhenNeeded", func(t *testing.T) {
		var (
			rootDir = "/"
			ctx     = context.Background()
			mv      = newManageVolume(t, rootDir)
			// File where VolumeIDs[0] is a different volume (this node is just a replica).
			// Drain must still create a Replica job because externalCount(1) < Replica(3).
			f = &file.File{
				Keys:      []string{"mykey"},
				Signature: "abc123",
				Replica:   3,
				VolumeIDs: []string{"other-owner-id", mv.V.ID()},
			}
		)
		defer mv.Finish()

		mv.Files.EXPECT().All(ctx).Return([]*file.File{f}, nil)

		mv.Replicas.EXPECT().Create(ctx, gomock.Any()).Do(
			func(_ context.Context, rp *replica.Replica) error {
				assert.Equal(t, f.Keys[0], rp.Key)
				assert.Equal(t, f.Signature, rp.Signature)
				assert.Equal(t, mv.V.ID(), rp.VolumeID)
				// VolumeIDs excludes self; "other-owner-id" is the only external holder.
				assert.Equal(t, []string{"other-owner-id"}, rp.VolumeIDs)
				// externalCount=1, Replica=3 → Count=2
				assert.Equal(t, 2, rp.Count)
				assert.Equal(t, 3, rp.OriginalCount)
				return nil
			},
		).Return(nil)

		err := mv.V.PrepareForDrain(ctx)
		require.NoError(t, err)
	})

	t.Run("NoReplicaJobWhenNonOwnedFileHasEnoughExternalReplicas", func(t *testing.T) {
		var (
			rootDir = "/"
			ctx     = context.Background()
			mv      = newManageVolume(t, rootDir)
			// Non-owned file that already has enough external copies — no job needed.
			f = &file.File{
				Keys:      []string{"mykey"},
				Signature: "abc123",
				Replica:   3,
				VolumeIDs: []string{"other-owner-id", mv.V.ID(), "r2", "r3"},
			}
		)
		defer mv.Finish()

		mv.Files.EXPECT().All(ctx).Return([]*file.File{f}, nil)
		// externalCount=3 >= Replica=3: no Replicas.Create call expected.

		err := mv.V.PrepareForDrain(ctx)
		require.NoError(t, err)
	})
}

func TestLocal_HasPendingReplicas(t *testing.T) {
	t.Run("ReturnsFalseWhenQueueIsEmpty", func(t *testing.T) {
		var (
			rootDir = "/"
			ctx     = context.Background()
			mv      = newManageVolume(t, rootDir)
		)
		defer mv.Finish()

		mv.Replicas.EXPECT().HasAny(ctx).Return(false, nil)

		has, err := mv.V.HasPendingReplicas(ctx)
		require.NoError(t, err)
		assert.False(t, has)
	})

	t.Run("ReturnsTrueWhenQueueHasReplica", func(t *testing.T) {
		var (
			rootDir = "/"
			ctx     = context.Background()
			mv      = newManageVolume(t, rootDir)
		)
		defer mv.Finish()

		mv.Replicas.EXPECT().HasAny(ctx).Return(true, nil)

		has, err := mv.V.HasPendingReplicas(ctx)
		require.NoError(t, err)
		assert.True(t, has)
	})
}

func TestLocal_PurgeAllFiles(t *testing.T) {
	t.Run("RemovesFilesAndKeysWithoutDeletionJobs", func(t *testing.T) {
		var (
			rootDir   = "/"
			ctx       = context.Background()
			mv        = newManageVolume(t, rootDir)
			fileDir   = path.Join(rootDir, "file")
			signature = "abc123def456"
			key       = "mykey"
			f         = &file.File{
				Keys:      []string{key},
				Signature: signature,
				Replica:   3,
				VolumeIDs: []string{mv.V.ID(), "remote1", "remote2"},
				Size:      42,
			}
		)
		defer mv.Finish()

		// First UoW: read all files
		mv.Files.EXPECT().All(ctx).Return([]*file.File{f}, nil)

		// Second UoW (per file): remove idxkey, purge file record/fs/state
		mv.IDXKeys.EXPECT().DeleteByKey(ctx, key).Return(nil)
		mv.Files.EXPECT().DeleteBySignature(ctx, signature).Return(nil)
		mv.Replicas.EXPECT().Delete(ctx, &replica.Replica{VolumeReplicaID: []byte(signature)}).Return(nil)
		mv.Fs.EXPECT().Remove(file.Path(fileDir, signature)).Return(nil)
		expectUpdateState(t, mv, ctx, -f.Size)

		// Deletions.Create must NOT be called

		err := mv.V.PurgeAllFiles(ctx)
		require.NoError(t, err)
	})

	t.Run("RemovesIDXTTLEntryWhenOnlySignature", func(t *testing.T) {
		var (
			rootDir   = "/"
			ctx       = context.Background()
			mv        = newManageVolume(t, rootDir)
			fileDir   = path.Join(rootDir, "file")
			signature = "abc123def456"
			key       = "mykey"
			ttl       = 2 * time.Minute
			ca        = time.Now()
			f         = &file.File{
				Keys:      []string{key},
				Signature: signature,
				Replica:   3,
				VolumeIDs: []string{mv.V.ID(), "remote1", "remote2"},
				Size:      42,
				TTL:       ttl,
				CreatedAt: ca,
			}
		)
		defer mv.Finish()

		// First UoW: read all files
		mv.Files.EXPECT().All(ctx).Return([]*file.File{f}, nil)

		// Second UoW: remove idxkey, update idxttl (only sig → delete), purge file
		mv.IDXKeys.EXPECT().DeleteByKey(ctx, key).Return(nil)

		// idxttl entry has only this file's signature — expect Delete
		mv.IDXTTLs.EXPECT().Find(ctx, f.ExpiresAt()).Return(idxttl.New(f.ExpiresAt(), signature), nil)
		mv.IDXTTLs.EXPECT().Delete(ctx, f.ExpiresAt()).Return(nil)

		mv.Files.EXPECT().DeleteBySignature(ctx, signature).Return(nil)
		mv.Replicas.EXPECT().Delete(ctx, &replica.Replica{VolumeReplicaID: []byte(signature)}).Return(nil)
		mv.Fs.EXPECT().Remove(file.Path(fileDir, signature)).Return(nil)
		expectUpdateState(t, mv, ctx, -f.Size)

		err := mv.V.PurgeAllFiles(ctx)
		require.NoError(t, err)
	})

	t.Run("UpdatesIDXTTLEntryWhenOtherSignaturesExist", func(t *testing.T) {
		var (
			rootDir       = "/"
			ctx           = context.Background()
			mv            = newManageVolume(t, rootDir)
			fileDir       = path.Join(rootDir, "file")
			signature     = "abc123def456"
			otherSig      = "other999sig000"
			key           = "mykey"
			ttl           = 2 * time.Minute
			ca            = time.Now()
			f             = &file.File{
				Keys:      []string{key},
				Signature: signature,
				Replica:   3,
				VolumeIDs: []string{mv.V.ID(), "remote1", "remote2"},
				Size:      42,
				TTL:       ttl,
				CreatedAt: ca,
			}
			// idxttl entry shared with another file
			sharedIDXTTL    = idxttl.New(f.ExpiresAt(), signature, otherSig)
			remainingIDXTTL = idxttl.New(f.ExpiresAt(), otherSig)
		)
		defer mv.Finish()

		// First UoW: read all files
		mv.Files.EXPECT().All(ctx).Return([]*file.File{f}, nil)

		// Second UoW: remove idxkey, update idxttl (other sig remains → CreateOrReplace), purge file
		mv.IDXKeys.EXPECT().DeleteByKey(ctx, key).Return(nil)

		// idxttl entry has two signatures — expect CreateOrReplace with only otherSig remaining
		mv.IDXTTLs.EXPECT().Find(ctx, f.ExpiresAt()).Return(sharedIDXTTL, nil)
		mv.IDXTTLs.EXPECT().CreateOrReplace(ctx, remainingIDXTTL).Return(nil)

		mv.Files.EXPECT().DeleteBySignature(ctx, signature).Return(nil)
		mv.Replicas.EXPECT().Delete(ctx, &replica.Replica{VolumeReplicaID: []byte(signature)}).Return(nil)
		mv.Fs.EXPECT().Remove(file.Path(fileDir, signature)).Return(nil)
		expectUpdateState(t, mv, ctx, -f.Size)

		err := mv.V.PurgeAllFiles(ctx)
		require.NoError(t, err)
	})
}
