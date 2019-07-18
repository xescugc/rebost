package boltdb

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/boltdb/bolt"
	"github.com/xescugc/rebost/replica"
)

type replicaPendentRepository struct {
	client     *bolt.DB
	bucketName []byte
	bucket     *bolt.Bucket
	key        keyGenerator
}

// NewReplicaPendentRepository returns an implementation of the interface replica.PendentRepository
func NewReplicaPendentRepository(c *bolt.DB) (replica.PendentRepository, error) {
	bn := []byte("replica-pendent")
	if err := createBucket(c, bn); err != nil {
		return nil, err
	}
	return &replicaPendentRepository{
		client:     c,
		bucketName: bn,
		key:        keyGenerator{},
	}, nil
}

func (r *replicaPendentRepository) Create(ctx context.Context, rp *replica.Pendent) error {
	rp.VolumeReplicaID = r.key.new()
	b, err := json.Marshal(rp)
	if err != nil {
		return err
	}
	return r.bucket.Put(rp.VolumeReplicaID, b)
}

func (r *replicaPendentRepository) First(ctx context.Context) (*replica.Pendent, error) {
	var p replica.Pendent
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

func (r *replicaPendentRepository) Delete(ctx context.Context, rp *replica.Pendent) error {
	return r.bucket.Delete(rp.VolumeReplicaID)
}
