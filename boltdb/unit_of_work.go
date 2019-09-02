package boltdb

import (
	"context"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/spf13/afero"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/idxkey"
	"github.com/xescugc/rebost/idxvolume"
	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/uow"
)

type unitOfWork struct {
	tx *bolt.Tx
	t  uow.Type

	fileRepository      file.Repository
	idxkeyRepository    idxkey.Repository
	idxvolumeRepository idxvolume.Repository
	fs                  afero.Fs
	replicaRepository   replica.Repository
}

type key int

var uowKey key

// NewUOW returns an implementation of the interface uow.StartUnitOfWork
// that will track all the boltDB repositories
func NewUOW(db *bolt.DB) uow.StartUnitOfWork {
	return func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn, repos ...interface{}) (err error) {
		uw := newUnitOfWork(t)

		if ctxUOW, ok := ctx.Value(uowKey).(*unitOfWork); ok {
			for _, r := range repos {
				if err = ctxUOW.add(r); err != nil {
					return fmt.Errorf("could not add repository: %s", err)
				}
			}
			ctx = context.WithValue(ctx, uowKey, ctxUOW)
			return uowFn(ctx, ctxUOW)
		} else {
			ctx = context.WithValue(ctx, uowKey, uw)
		}

		err = uw.begin(db)
		if err != nil {
			return fmt.Errorf("could not initialize TX: %s", err)
		}

		defer func() {
			rerr := uw.rollback()
			if rerr != nil && rerr != bolt.ErrTxClosed {
				err = fmt.Errorf("failed to rollback TX: %s", rerr)
			}
			return
		}()

		for _, r := range repos {
			if err = uw.add(r); err != nil {
				return fmt.Errorf("could not add repository: %s", err)
			}
		}

		defer func() {
			// Only commit if no error found
			if err == nil {
				cerr := uw.commit()
				if cerr != nil {
					err = fmt.Errorf("failed to commit TX: %s", cerr)
				}
			}
			return
		}()

		return uowFn(ctx, uw)
	}
}

func (uw *unitOfWork) Files() file.Repository {
	return uw.fileRepository
}

func (uw *unitOfWork) IDXKeys() idxkey.Repository {
	return uw.idxkeyRepository
}

func (uw *unitOfWork) IDXVolumes() idxvolume.Repository {
	return uw.idxvolumeRepository
}

func (uw *unitOfWork) Fs() afero.Fs {
	return uw.fs
}

func (uw *unitOfWork) Replicas() replica.Repository {
	return uw.replicaRepository
}

func newUnitOfWork(t uow.Type) *unitOfWork {
	return &unitOfWork{
		t: t,
	}
}

func (uw *unitOfWork) begin(db *bolt.DB) error {
	var (
		tx  *bolt.Tx
		err error
	)

	if uw.t == uow.Read {
		tx, err = db.Begin(false)
	} else if uw.t == uow.Write {
		tx, err = db.Begin(true)
	} else {
		err = fmt.Errorf("unsoported uow.Type: %d", uw.t)
	}
	if err != nil {
		return err
	}

	uw.tx = tx

	return nil
}

func (uw *unitOfWork) rollback() error {
	return uw.tx.Rollback()
}

func (uw *unitOfWork) commit() error {
	if uw.t == uow.Read {
		return nil
	} else if uw.t == uow.Write {
		return uw.tx.Commit()
	} else {
		return fmt.Errorf("unsoported uow.Type: %d", uw.t)
	}
}

func (uw *unitOfWork) add(r interface{}) error {
	switch rep := r.(type) {
	case *fileRepository:
		if uw.fileRepository == nil {
			r := *rep
			b := uw.tx.Bucket(r.bucketName)
			if b == nil {
				return fmt.Errorf("bucker for %q not found", r.bucketName)
			}
			r.bucket = b
			uw.fileRepository = &r
		}
		return nil
	case *idxkeyRepository:
		if uw.idxkeyRepository == nil {
			r := *rep
			b := uw.tx.Bucket(r.bucketName)
			if b == nil {
				return fmt.Errorf("bucker for %q not found", r.bucketName)
			}
			r.bucket = b
			uw.idxkeyRepository = &r
		}
		return nil
	case *idxvolumeRepository:
		if uw.idxvolumeRepository == nil {
			r := *rep
			b := uw.tx.Bucket(r.bucketName)
			if b == nil {
				return fmt.Errorf("bucker for %q not found", r.bucketName)
			}
			r.bucket = b
			uw.idxvolumeRepository = &r
		}
		return nil
	case *replicaRepository:
		if uw.replicaRepository == nil {
			r := *rep
			b := uw.tx.Bucket(r.bucketName)
			if b == nil {
				return fmt.Errorf("bucker for %q not found", r.bucketName)
			}
			r.bucket = b
			uw.replicaRepository = &r
		}
		return nil
	default:
		if v, ok := r.(afero.Fs); ok {
			uw.fs = v
			return nil
		}
		return fmt.Errorf("inalid respository of type: %T", rep)
	}
}
