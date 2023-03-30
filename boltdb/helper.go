package boltdb

import (
	"fmt"

	bolt "go.etcd.io/bbolt"
)

// createBucket tries to create a new bucket with the given name
func createBucket(db *bolt.DB, name []byte) error {
	return db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(name)
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
}

// recreateBucket will delete and create a new bucket
func recreateBucket(bk *bolt.Bucket, name []byte) (*bolt.Bucket, error) {
	err := bk.Tx().DeleteBucket(name)
	if err != nil {
		return nil, fmt.Errorf("delete bucket: %s", err)
	}
	nbk, err := bk.Tx().CreateBucketIfNotExists(name)
	if err != nil {
		return nil, fmt.Errorf("create bucket: %s", err)
	}
	return nbk, nil
}
