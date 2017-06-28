package storage

import (
	"errors"
	"log"

	"github.com/boltdb/bolt"
)

func NewIndex(path string) *bolt.DB {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range []string{"files"} {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return db
}

// indexSetFile will add the given File to this volume index
// and it will return a string having the previous signature
// pointed by the given File key in case it existed.
func (v *volume) indexSetFile(file *File) (string, error) {
	var oldSignature string
	err := v.index.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("files"))

		// look if we know that key
		oldSignature = string(b.Get([]byte(file.key)))
		// skip if we know it and it's the same
		if oldSignature != "" && oldSignature == file.Signature {
			oldSignature = ""
			return nil
		}
		//TODO: check here if this signature is referred by other key?, probably not...

		err := b.Put([]byte(file.key), []byte(file.Signature))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return oldSignature, nil
}

func (v *volume) indexGetFileSignature(key string) (string, error) {
	var sig []byte
	err := v.index.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("files"))
		sig = b.Get([]byte(key))
		if sig == nil {
			return errors.New("Missing file")
		}
		return nil
	})

	return string(sig), err
}

func (v *volume) indexDeleteFile(key string) (string, error) {
	var sig string
	err := v.index.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("files"))
		sig = string(b.Get([]byte(key)))
		return b.Delete([]byte(key))
	})

	return sig, err
}
