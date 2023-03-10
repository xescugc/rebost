// Package uow provides an interface in which the "repositories" that participate on it
// are asure that the functions/actions that are called will be rollback if the Unit of Work
// fails at some point.
// So it's not necessary to care about removing the already created data if an error raises
// on the middle of the Unit of Work. It's basically an interface to emulate a Transaction
// which is a more common word for it.
package uow

import (
	"context"

	"github.com/spf13/afero"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/idxkey"
	"github.com/xescugc/rebost/idxvolume"
	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/state"
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
	IDXVolumes() idxvolume.Repository
	Fs() afero.Fs
	Replicas() replica.Repository
	State() state.Repository
}

// StartUnitOfWork it's the way to initialize a typed UoW, it has a uowFn
// which is the callback where all the work should be done, it also has the
// repositories, which are all the Repositories that belong to this UoW
type StartUnitOfWork func(ctx context.Context, t Type, uowFn UnitOfWorkFn, repositories ...interface{}) error

// UnitOfWorkFn is the signature of the function
// that is the callback of the StartUnitOfWork
type UnitOfWorkFn func(ctx context.Context, uw UnitOfWork) error
