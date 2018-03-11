package boltdb

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/spf13/afero"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/idxkey"
	"github.com/xescugc/rebost/uow"
)

type unitOfWork struct {
	tx *bolt.Tx
	t  uow.Type

	fileRepository   file.Repository
	idxkeyRepository idxkey.Repository
	fs               afero.Fs
}

// NewUOW returns an implementation of the interface uow.StartUnitOfWork
func NewUOW(db *bolt.DB) uow.StartUnitOfWork {
	return func(t uow.Type, uowFn uow.UnitOfWorkFn, repos ...interface{}) (err error) {
		uw := newUnitOfWork(t)

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

		return uowFn(uw)
	}
}

func (uw *unitOfWork) Files() file.Repository {
	return uw.fileRepository
}

func (uw *unitOfWork) IDXKeys() idxkey.Repository {
	return uw.idxkeyRepository
}

func (uw *unitOfWork) Fs() afero.Fs {
	return uw.fs
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
		r := *rep
		b := uw.tx.Bucket(r.bucketName)
		if b == nil {
			return fmt.Errorf("bucker for %q not found", r.bucketName)
		}
		r.bucket = b
		uw.fileRepository = &r
		return nil
	case *idxkeyRepository:
		r := *rep
		b := uw.tx.Bucket(r.bucketName)
		if b == nil {
			return fmt.Errorf("bucker for %q not found", r.bucketName)
		}
		r.bucket = b
		uw.idxkeyRepository = &r
		return nil
	default:
		if v, ok := r.(afero.Fs); ok {
			uw.fs = v
			return nil
		}
		return fmt.Errorf("inalid respository of type: %T", rep)
	}
}
