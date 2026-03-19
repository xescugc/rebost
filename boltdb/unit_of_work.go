package boltdb

import (
	"context"
	"fmt"

	"github.com/xescugc/rebost/deletion"
	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/idxkey"
	"github.com/xescugc/rebost/idxttl"
	"github.com/xescugc/rebost/idxvolume"
	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/state"
	"github.com/xescugc/rebost/uow"
	bolt "go.etcd.io/bbolt"

	"github.com/spf13/afero"
)

type unitOfWork struct {
	tx *bolt.Tx
	t  uow.Type

	fileRepository      file.Repository
	idxkeyRepository    idxkey.Repository
	idxttlRepository    idxttl.Repository
	idxvolumeRepository idxvolume.Repository
	replicaRepository   replica.Repository
	deletionRepository  deletion.Repository
	stateRepository     state.Repository
}

type key int

var uowKey key

// NewUOW returns an implementation of the interface uow.StartUnitOfWork
// that will lazily initialize all boltDB repositories on first access.
// It creates all required buckets as part of initialization.
func NewUOW(db *bolt.DB) (uow.StartUnitOfWork, error) {
	buckets := [][]byte{
		[]byte("files"), []byte("idxkeys"), []byte("idxttls"),
		[]byte("idxvolumes"), []byte("replica"), []byte("deletion"), []byte("state"),
	}
	for _, bn := range buckets {
		if err := createBucket(db, bn); err != nil {
			return nil, fmt.Errorf("could not create bucket %q: %s", bn, err)
		}
	}

	return func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn) (err error) {
		if ctxUOW, ok := ctx.Value(uowKey).(*unitOfWork); ok {
			return uowFn(ctx, ctxUOW)
		}

		uw := newUnitOfWork(t)
		ctx = context.WithValue(ctx, uowKey, uw)

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
	}, nil
}

func (uw *unitOfWork) Files() file.Repository {
	if uw.fileRepository == nil {
		bn := []byte("files")
		b := uw.tx.Bucket(bn)
		uw.fileRepository = &fileRepository{bucketName: bn, bucket: b}
	}
	return uw.fileRepository
}

func (uw *unitOfWork) IDXKeys() idxkey.Repository {
	if uw.idxkeyRepository == nil {
		bn := []byte("idxkeys")
		b := uw.tx.Bucket(bn)
		uw.idxkeyRepository = &idxkeyRepository{bucketName: bn, bucket: b}
	}
	return uw.idxkeyRepository
}

func (uw *unitOfWork) IDXTTLs() idxttl.Repository {
	if uw.idxttlRepository == nil {
		bn := []byte("idxttls")
		b := uw.tx.Bucket(bn)
		uw.idxttlRepository = &idxttlRepository{bucketName: bn, bucket: b}
	}
	return uw.idxttlRepository
}

func (uw *unitOfWork) IDXVolumes() idxvolume.Repository {
	if uw.idxvolumeRepository == nil {
		bn := []byte("idxvolumes")
		b := uw.tx.Bucket(bn)
		uw.idxvolumeRepository = &idxvolumeRepository{bucketName: bn, bucket: b}
	}
	return uw.idxvolumeRepository
}

func (uw *unitOfWork) Fs() afero.Fs {
	return nil
}

func (uw *unitOfWork) Replicas() replica.Repository {
	if uw.replicaRepository == nil {
		bn := []byte("replica")
		b := uw.tx.Bucket(bn)
		uw.replicaRepository = &replicaRepository{bucketName: bn, bucket: b}
	}
	return uw.replicaRepository
}

func (uw *unitOfWork) Deletions() deletion.Repository {
	if uw.deletionRepository == nil {
		bn := []byte("deletion")
		b := uw.tx.Bucket(bn)
		uw.deletionRepository = &deletionRepository{bucketName: bn, bucket: b}
	}
	return uw.deletionRepository
}

func (uw *unitOfWork) State() state.Repository {
	if uw.stateRepository == nil {
		bn := []byte("state")
		b := uw.tx.Bucket(bn)
		uw.stateRepository = &stateRepository{bucketName: bn, bucket: b, key: []byte("state")}
	}
	return uw.stateRepository
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
		err = fmt.Errorf("unsupported uw.Type: %d", uw.t)
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
		return fmt.Errorf("unsupported uw.Type: %d", uw.t)
	}
}
