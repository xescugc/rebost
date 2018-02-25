package boltdb

import (
	"encoding/json"
	"errors"

	"github.com/boltdb/bolt"
	"github.com/xescugc/rebost/file"
)

type fileRepository struct {
	client     *bolt.DB
	bucketName []byte
	bucket     *bolt.Bucket
}

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

func (r *fileRepository) CreateOrReplace(f *file.File) error {
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	return r.bucket.Put([]byte(f.Signature), b)
}

func (r *fileRepository) FindBySignature(sig string) (*file.File, error) {
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

func (r *fileRepository) DeleteBySignature(sig string) error {
	return r.bucket.Delete([]byte(sig))
}
