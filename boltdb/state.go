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
	}, nil
}

func (r *stateRepository) Find(ctx context.Context, vid string) (*state.State, error) {
	var s state.State
	b := r.bucket.Get([]byte(vid))
	if b == nil {
		return &state.State{}, nil
	}
	err := json.Unmarshal(b, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *stateRepository) Update(ctx context.Context, vid string, s *state.State) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return r.bucket.Put([]byte(vid), b)
}
