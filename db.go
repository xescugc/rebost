package main

import (
	"log"

	"github.com/boltdb/bolt"
)

var (
	db *bolt.DB
)

func init() {
	var err error
	db, err = bolt.Open("rebost.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("files"))
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
}
