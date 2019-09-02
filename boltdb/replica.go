package boltdb

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/boltdb/bolt"
	"github.com/xescugc/rebost/replica"
)

type replicaRepository struct {
	client     *bolt.DB
	bucketName []byte
	bucket     *bolt.Bucket
}

// NewReplicaRepository returns an implementation of the interface replica.PendentRepository
func NewReplicaRepository(c *bolt.DB) (replica.Repository, error) {
	bn := []byte("replica")
	if err := createBucket(c, bn); err != nil {
		return nil, err
	}
	return &replicaRepository{
		client:     c,
		bucketName: bn,
	}, nil
}

func (r *replicaRepository) Create(ctx context.Context, rp *replica.Replica) error {
	rp.VolumeReplicaID = newKey()
	b, err := json.Marshal(rp)
	if err != nil {
		return err
	}
	return r.bucket.Put(rp.VolumeReplicaID, b)
}

func (r *replicaRepository) First(ctx context.Context) (*replica.Replica, error) {
	var p replica.Replica
	_, b := r.bucket.Cursor().First()
	if b == nil {
		return nil, errors.New("not found")
	}
	err := json.Unmarshal(b, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *replicaRepository) Delete(ctx context.Context, rp *replica.Replica) error {
	return r.bucket.Delete(rp.VolumeReplicaID)
}
