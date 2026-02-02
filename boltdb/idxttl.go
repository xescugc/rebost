package boltdb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/xescugc/duration"
	"github.com/xescugc/rebost/idxttl"
	bolt "go.etcd.io/bbolt"
)

type idxttlRepository struct {
	client     *bolt.DB
	bucketName []byte
	bucket     *bolt.Bucket
}

// NewIDXTTLRepository returns an implementation of the interface idxttl.Repository
func NewIDXTTLRepository(c *bolt.DB) (idxttl.Repository, error) {
	bn := []byte("idxttls")
	if err := createBucket(c, bn); err != nil {
		return nil, err
	}
	return &idxttlRepository{
		client:     c,
		bucketName: bn,
	}, nil
}

func (r *idxttlRepository) CreateOrReplace(ctx context.Context, ittl *idxttl.IDXTTL) error {
	b, err := json.Marshal(ittl.Signatures)
	if err != nil {
		return err
	}
	return r.bucket.Put(formatTime(ittl.ExpiresAt), b)
}

// Filter will return all the IDXTTL older than ea
func (r *idxttlRepository) Filter(ctx context.Context, ea time.Time) ([]*idxttl.IDXTTL, error) {
	min := formatTime(time.Now().Add(-duration.Day))
	max := formatTime(ea)

	c := r.bucket.Cursor()
	idxttls := make([]*idxttl.IDXTTL, 0, 0)
	for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
		ittl := newIDXTTLFromDB(k, v)
		idxttls = append(idxttls, ittl)
	}
	return idxttls, nil
}

func (r *idxttlRepository) Find(ctx context.Context, ea time.Time) (*idxttl.IDXTTL, error) {
	k := formatTime(ea)
	b := r.bucket.Get(k)
	if b == nil {
		return nil, errors.New("not found")
	}
	ittl := newIDXTTLFromDB(k, b)
	return ittl, nil
}

func (r *idxttlRepository) Delete(ctx context.Context, ea time.Time) error {
	return r.bucket.Delete(formatTime(ea))
}

func newIDXTTLFromDB(k, v []byte) *idxttl.IDXTTL {
	ittl := &idxttl.IDXTTL{
		ExpiresAt: parseTime(k),
	}
	_ = json.Unmarshal(v, &ittl.Signatures)
	return ittl
}

func formatTime(t time.Time) []byte {
	return []byte(t.Format(time.RFC3339))
}

func parseTime(b []byte) time.Time {
	t, _ := time.Parse(string(b), time.RFC3339)
	return t
}
