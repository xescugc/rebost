package boltdb

import (
	"context"
	"encoding/json"

	"github.com/xescugc/rebost/state"
	bolt "go.etcd.io/bbolt"
)

type stateRepository struct {
	client     *bolt.DB
	bucketName []byte
	bucket     *bolt.Bucket
	key        []byte
}

// NewStateRepository returns an implementation of the interface state.Repository
func NewStateRepository(c *bolt.DB) (state.Repository, error) {
	bn := []byte("state")
	if err := createBucket(c, bn); err != nil {
		return nil, err
	}
	return &stateRepository{
		client:     c,
		bucketName: bn,
		key:        []byte("state"),
	}, nil
}

func (r *stateRepository) Find(ctx context.Context) (*state.State, error) {
	var s state.State
	b := r.bucket.Get(r.key)
	if b == nil {
		return &state.State{}, nil
	}
	err := json.Unmarshal(b, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *stateRepository) Update(ctx context.Context, s *state.State) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return r.bucket.Put(r.key, b)
}

func (r *stateRepository) DeleteAll(ctx context.Context) error {
	bk, err := recreateBucket(r.bucket, r.bucketName)
	if err != nil {
		return err
	}
	r.bucket = bk
	return nil
}
