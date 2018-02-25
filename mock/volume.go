package mock

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/idxkey"
	"github.com/xescugc/rebost/uow"
	"github.com/xescugc/rebost/volume"
)

type Volume struct {
	CreateFileFn      func(k string, reader io.Reader) (*file.File, error)
	CreateFileInvoked bool

	GetFileFn      func(k string) (io.Reader, error)
	GetFileInvoked bool

	HasFileFn      func(k string) (bool, error)
	HasFileInvoked bool

	DeleteFileFn      func(k string) error
	DeleteFileInvoked bool
}

func (v *Volume) CreateFile(k string, r io.Reader) (*file.File, error) {
	v.CreateFileInvoked = true
	return v.CreateFileFn(k, r)
}

func (v *Volume) GetFile(k string) (io.Reader, error) {
	v.GetFileInvoked = true
	return v.GetFileFn(k)
}

func (v *Volume) HasFile(k string) (bool, error) {
	v.HasFileInvoked = true
	return v.HasFileFn(k)
}

func (v *Volume) DeleteFile(k string) error {
	v.DeleteFileInvoked = true
	return v.DeleteFileFn(k)
}

type ManageVolume struct {
	Files   *FileRepository
	IDXKeys *IDXKeyRepository
	Fs      *Fs

	V volume.Volume
}

func NewManageVolume(t *testing.T, root string) ManageVolume {
	var files FileRepository
	var fs Fs
	var idxkeys IDXKeyRepository

	var uowFn = func(t uow.Type, uowFn uow.UnitOfWorkFn, repositories ...interface{}) error {
		var uw UnitOfWork
		uw.FilesFn = func() file.Repository {
			return &files
		}
		uw.IDXKeysFn = func() idxkey.Repository {
			return &idxkeys
		}
		return uowFn(&uw)
	}

	fs.MkdirAllFn = func(p string, fm os.FileMode) error {
		// This first implementation is already tested
		// so we do not need it
		return nil
	}

	v, err := volume.New(root, &files, &idxkeys, &fs, uowFn)
	require.NoError(t, err)

	return ManageVolume{
		Files:   &files,
		IDXKeys: &idxkeys,
		Fs:      &fs,

		V: v,
	}

}
