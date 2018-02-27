package uow

import (
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/idxkey"
)

//go:generate mockgen -destination=../mock/unit_of_work.go -mock_names=UnitOfWork=UnitOfWork -package mock github.com/xescugc/rebost/uow UnitOfWork

// Type is the type of the UniteOfWork
type Type int

const (
	// Read is the type of UoW that only reads data
	Read Type = iota

	// Write is the type of UoW that Reads and Writes data
	Write
)

// UnitOfWork is the interface that any UnitOfWork has to follow
// the only methods it as are to return Repositories that work
// together to achive a common purpose/work.
type UnitOfWork interface {
	Files() file.Repository
	IDXKeys() idxkey.Repository
}

// StartUnitOfWork it's the way to initialize a typed UoW, it has a uowFn
// which is the callback where all the work should be done, it also has the
// repositories, which are all the Repositories that belong to this UoW
type StartUnitOfWork func(t Type, uowFn UnitOfWorkFn, repositories ...interface{}) error

// UnitOfWorkFn is the signature of the function
// that is the callback of the StartUnitOfWork
type UnitOfWorkFn func(uw UnitOfWork) error
