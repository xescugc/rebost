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
