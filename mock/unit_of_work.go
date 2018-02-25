package mock

import (
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/idxkey"
)

type UnitOfWork struct {
	FilesFn      func() file.Repository
	FilesInvoked bool

	IDXKeysFn      func() idxkey.Repository
	IDXKeysInvoked bool
}

func (uw *UnitOfWork) Files() file.Repository {
	uw.FilesInvoked = true
	return uw.FilesFn()
}

func (uw *UnitOfWork) IDXKeys() idxkey.Repository {
	uw.IDXKeysInvoked = true
	return uw.IDXKeysFn()
}
