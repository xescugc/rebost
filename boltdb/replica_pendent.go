package boltdb

import (
	"context"
	"encoding/json"

	"github.com/boltdb/bolt"
	"github.com/xescugc/rebost/replica"
)

type replicaPendentRepository struct {
	client     *bolt.DB
	bucketName []byte
	bucket     *bolt.Bucket
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
	}, nil
}

func (r *replicaPendentRepository) Create(ctx context.Context, rp *replica.Pendent) error {
	b, err := json.Marshal(rp)
	if err != nil {
		return err
	}
	return r.bucket.Put([]byte(rp.ID), b)
}
