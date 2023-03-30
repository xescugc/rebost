package boltdb

import (
	"context"
	"errors"

	"github.com/xescugc/rebost/idxkey"
	bolt "go.etcd.io/bbolt"
)

type idxkeyRepository struct {
	client     *bolt.DB
	bucketName []byte
	bucket     *bolt.Bucket
}

// NewIDXKeyRepository returns an implementation of the interface idxkey.Repository
func NewIDXKeyRepository(c *bolt.DB) (idxkey.Repository, error) {
	bn := []byte("idxkeys")
	if err := createBucket(c, bn); err != nil {
		return nil, err
	}
	return &idxkeyRepository{
		client:     c,
		bucketName: bn,
	}, nil
}

func (r *idxkeyRepository) CreateOrReplace(ctx context.Context, ik *idxkey.IDXKey) error {
	return r.bucket.Put([]byte(ik.Key), []byte(ik.Value))
}

func (r *idxkeyRepository) FindByKey(ctx context.Context, k string) (*idxkey.IDXKey, error) {
	v := r.bucket.Get([]byte(k))
	if v == nil {
		return nil, errors.New("not found")
	}
	return idxkey.New(k, string(v)), nil
}

func (r *idxkeyRepository) DeleteByKey(ctx context.Context, k string) error {
	return r.bucket.Delete([]byte(k))
}

func (r *idxkeyRepository) DeleteAll(ctx context.Context) error {
	bk, err := recreateBucket(r.bucket, r.bucketName)
	if err != nil {
		return err
	}
	r.bucket = bk
	return nil
}
