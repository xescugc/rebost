package storage

import (
	"io"
	"log"
	"os"
	"path"

	"github.com/boltdb/bolt"
)

type volume struct {
	rootDir string
	tempDir string
	fileDir string
	index   *bolt.DB
}

func NewVolume(root string) *volume {
	v := &volume{
		root,
		path.Join(root, "tmps"),
		path.Join(root, "file"),
		NewIndex(path.Join(root, "volume.index")),
	}

	v.initialize()
	return s
}

func (v *volume) initialize() {
	err := os.MkdirAll(v.tempDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(v.fileDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
}

func (v *volume) Add(key string, reader io.Reader) (*File, error) {
	f := File{key: key, volume: v}
	err := f.Store(reader)

	return f, err
}
