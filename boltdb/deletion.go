package boltdb

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/xescugc/rebost/deletion"
	bolt "go.etcd.io/bbolt"
)

type deletionRepository struct {
	client     *bolt.DB
	bucketName []byte
	bucket     *bolt.Bucket
}

// NewDeletionRepository returns an implementation of deletion.Repository backed by BoltDB.
func NewDeletionRepository(c *bolt.DB) (deletion.Repository, error) {
	bn := []byte("deletion")
	if err := createBucket(c, bn); err != nil {
		return nil, err
	}
	return &deletionRepository{
		client:     c,
		bucketName: bn,
	}, nil
}

func (r *deletionRepository) Create(ctx context.Context, d *deletion.Deletion) error {
	d.VolumeDeletionID = newKey()
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return r.bucket.Put(d.VolumeDeletionID, b)
}

func (r *deletionRepository) First(ctx context.Context) (*deletion.Deletion, error) {
	var d deletion.Deletion
	_, b := r.bucket.Cursor().First()
	if b == nil {
		return nil, errors.New("not found")
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *deletionRepository) Delete(ctx context.Context, d *deletion.Deletion) error {
	return r.bucket.Delete(d.VolumeDeletionID)
}
