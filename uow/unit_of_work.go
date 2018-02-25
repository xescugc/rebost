package uow

import (
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/idxkey"
)

type Type int

const (
	Read Type = iota
	Write
)

type UnitOfWork interface {
	Files() file.Repository
	IDXKeys() idxkey.Repository
}

type StartUnitOfWork func(t Type, uowFn UnitOfWorkFn, repositories ...interface{}) error

type UnitOfWorkFn func(uw UnitOfWork) error
