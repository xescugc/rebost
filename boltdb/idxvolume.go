package boltdb

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/xescugc/rebost/idxvolume"
	bolt "go.etcd.io/bbolt"
)

type idxvolumeRepository struct {
	client     *bolt.DB
	bucketName []byte
	bucket     *bolt.Bucket
}

// NewIDXVolumeRepository returns an implementation of the interface idxvolume.Repository
func NewIDXVolumeRepository(c *bolt.DB) (idxvolume.Repository, error) {
	bn := []byte("idxvolumes")
	if err := createBucket(c, bn); err != nil {
		return nil, err
	}
	return &idxvolumeRepository{
		client:     c,
		bucketName: bn,
	}, nil
}

func (r *idxvolumeRepository) CreateOrReplace(ctx context.Context, iv *idxvolume.IDXVolume) error {
	b, err := json.Marshal(iv.Signatures)
	if err != nil {
		return err
	}
	return r.bucket.Put([]byte(iv.VolumeID), b)
}

func (r *idxvolumeRepository) FindByVolumeID(ctx context.Context, vid string) (*idxvolume.IDXVolume, error) {
	b := r.bucket.Get([]byte(vid))
	if b == nil {
		return nil, errors.New("not found")
	}

	var sigs []string
	err := json.Unmarshal(b, &sigs)
	if err != nil {
		return nil, err
	}

	return idxvolume.New(vid, sigs), nil
}

func (r *idxvolumeRepository) DeleteByKey(ctx context.Context, k string) error {
	return r.bucket.Delete([]byte(k))
}
