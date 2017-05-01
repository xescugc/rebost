package main

import (
	"errors"
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

func dbGetFileSignature(key string) (string, error) {
	var v []byte
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("files"))
		v = b.Get([]byte(key))
		if v == nil {
			return errors.New("Missing file")
		}
		return nil
	})

	return string(v), err
}

// Set a file-key signature and get the signature replaced if it exists
func dbSetFileSignature(key, signature string) (string, error) {
	var old string
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("files"))

		// look if we know that key
		old = string(b.Get([]byte(key)))
		// skip if we know it and it's the same
		if old != "" && old == signature {
			old = ""
			return nil
		}

		err := b.Put([]byte(key), []byte(signature))
		if err != nil {
			logger.error("DB:PUT " + err.Error())
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return old, nil
}

func dbDelFile(key string) error {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("files"))
		return b.Delete([]byte(key))
	})

	return err
}
