package boltdb

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/boltdb/bolt"
	"github.com/xescugc/rebost/replica"
)

type replicaRetryRepository struct {
	client     *bolt.DB
	bucketName []byte
	bucket     *bolt.Bucket
	key        keyGenerator
}

// NewReplicaRetryRepository returns an implementation of the interface replica.RetryRepository
func NewReplicaRetryRepository(c *bolt.DB) (replica.RetryRepository, error) {
	bn := []byte("replica-retry")
	if err := createBucket(c, bn); err != nil {
		return nil, err
	}
	return &replicaRetryRepository{
		client:     c,
		bucketName: bn,
		key:        keyGenerator{},
	}, nil
}

func (r *replicaRetryRepository) Create(ctx context.Context, rr *replica.Retry) error {
	rr.VolumeReplicaID = r.key.new()
	b, err := json.Marshal(rr)
	if err != nil {
		return err
	}
	return r.bucket.Put(rr.VolumeReplicaID, b)
}

func (r *replicaRetryRepository) First(ctx context.Context) (*replica.Retry, error) {
	var r replica.Retry
	_, b := r.bucket.Cursor().First()
	if b == nil {
		return nil, errors.New("not found")
	}
	err := json.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *replicaRetryRepository) Delete(ctx context.Context, rr *replica.Retry) error {
	return r.bucket.Delete(rr.VolumeReplicaID)
}
