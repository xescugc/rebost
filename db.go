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
	db, err = bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
}
