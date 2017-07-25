package storage

import (
	"io"
	"log"
	"os"
	"path"

	"github.com/boltdb/bolt"
)

// Volume is an interface to deal with local or remote volumes
type Volume interface {
	AddFile(key string, reader io.Reader) (*File, error)

	GetFile(key string) (*File, error)

	Exists(key string) (*File, error)

	DeleteFile(key string) error
}

type volume struct {
	rootDir string
	tempDir string
	fileDir string
	index   *bolt.DB
}

func NewVolume(root string) Volume {
	v := &volume{
		rootDir: root,
		tempDir: path.Join(root, "tmps"),
		fileDir: path.Join(root, "file"),
	}

	v.initialize()

	// Wait to Initialize the volume before connecting the DB
	// so the creation of the data directories is done
	v.index = NewIndex(path.Join(root, "volume.index"))

	return v
}

func (v *volume) initialize() {
	err := os.MkdirAll(v.rootDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(v.tempDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(v.fileDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
}

// Clean removes all the data from the Volume and closes the DB connection
func (v volume) Clean() {
	err := v.index.Close()
	if err != nil {
		log.Fatalf("error while closing the DB: %q", err)
	}
	err = os.RemoveAll(v.rootDir)
	if err != nil {
		log.Fatalf("error while cleaning the volume: %q", err)
	}
}

// AddFile adds a new File to the Volume
func (v *volume) AddFile(key string, reader io.Reader) (*File, error) {
	f := v.newFileFromKey(key)

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
		oldFile := v.newFile(key, oldSig)
		_ = oldFile.remove()
	}

	return f, err
}

// Get searches for a file with the given key
func (v *volume) GetFile(key string) (*File, error) {
	sig, err := v.indexGetFileSignature(key)

	if err != nil {
		return nil, err
	}

	return v.newFileFromSignature(sig), nil
}

// Delete removes a File with the matching key
func (v *volume) DeleteFile(key string) error {
	sig, err := v.indexDeleteFile(key)
	if err != nil {
		return err
	}
	if sig != "" {
		f := v.newFileFromSignature(sig)
		return f.remove()
	} else {
		return nil
	}
}

func (v *volume) Exists(key string) (*File, error) { return nil, nil }

func (v *volume) newFile(k, s string) *File           { return &File{key: k, Signature: s, volume: v} }
func (v *volume) newFileFromKey(k string) *File       { return &File{key: k, volume: v} }
func (v *volume) newFileFromSignature(s string) *File { return &File{Signature: s, volume: v} }
