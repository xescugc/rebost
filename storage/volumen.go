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
	return v
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
	f := &File{key: key, volume: v}

	// Save on disk and calculate signature
	err := f.store(reader)
	if err != nil {
		return f, err
	}

	// save on index and get old file if it need to be deleted
	oldSig, err := v.indexSetFile(f)
	if err != nil {
		return f, err
	}
	if oldSig != "" {
		oldFile := &File{key: key, Signature: oldSig, volume: v}
		_ = oldFile.remove()
	}

	return f, err
}
