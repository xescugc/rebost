package boltdb

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/xescugc/rebost/file"
	bolt "go.etcd.io/bbolt"
)

type fileRepository struct {
	client     *bolt.DB
	bucketName []byte
	bucket     *bolt.Bucket
}

// NewFileRepository returns an implementation of the interface file.Repository
func NewFileRepository(c *bolt.DB) (file.Repository, error) {
	bn := []byte("files")
	if err := createBucket(c, bn); err != nil {
		return nil, err
	}
	return &fileRepository{
		client:     c,
		bucketName: bn,
	}, nil
}

func (r *fileRepository) CreateOrReplace(ctx context.Context, f *file.File) error {
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	return r.bucket.Put([]byte(f.Signature), b)
}

func (r *fileRepository) FindBySignature(ctx context.Context, sig string) (*file.File, error) {
	var f file.File
	b := r.bucket.Get([]byte(sig))
	if b == nil {
		return nil, errors.New("not found")
	}
	err := json.Unmarshal(b, &f)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *fileRepository) DeleteBySignature(ctx context.Context, sig string) error {
	return r.bucket.Delete([]byte(sig))
}

func (r *fileRepository) DeleteAll(ctx context.Context) error {
	bk, err := recreateBucket(r.bucket, r.bucketName)
	if err != nil {
		return err
	}
	r.bucket = bk
	return nil
}

func (r *fileRepository) All(ctx context.Context) ([]*file.File, error) {
	var files []*file.File
	err := r.bucket.ForEach(func(k, v []byte) error {
		var f file.File
		if err := json.Unmarshal(v, &f); err != nil {
			return err
		}
		files = append(files, &f)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
