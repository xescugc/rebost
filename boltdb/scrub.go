package boltdb

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/xescugc/rebost/scrub"
	bolt "go.etcd.io/bbolt"
)

type scrubRepository struct {
	client     *bolt.DB
	bucketName []byte
	bucket     *bolt.Bucket
}

// NewScrubRepository returns an implementation of scrub.Repository backed by BoltDB.
func NewScrubRepository(c *bolt.DB) (scrub.Repository, error) {
	bn := []byte("scrub")
	if err := createBucket(c, bn); err != nil {
		return nil, err
	}
	return &scrubRepository{
		client:     c,
		bucketName: bn,
	}, nil
}

func (r *scrubRepository) Create(ctx context.Context, s scrub.Scrub) error {
	s.ScrubID = newKey()
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return r.bucket.Put(s.ScrubID, b)
}

func (r *scrubRepository) First(ctx context.Context) (*scrub.Scrub, error) {
	var s scrub.Scrub
	_, b := r.bucket.Cursor().First()
	if b == nil {
		return nil, errors.New("not found")
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *scrubRepository) Delete(ctx context.Context, s *scrub.Scrub) error {
	return r.bucket.Delete(s.ScrubID)
}
