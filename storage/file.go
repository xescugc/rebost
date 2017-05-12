package storage

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path"

	uuid "github.com/satori/go.uuid"
)

type File struct {
	Signature string
	key       string
	volume    *volume
}

func (f *File) Path() string {
	p := f.volume.fileDir
	currentDir := []byte{}
	for _, b := range []byte(f.Signature) {
		currentDir = append(currentDir, b)
		if len(currentDir) == 3 {
			p = path.Join(p, string(currentDir))
			currentDir = []byte{}
		}
	}
	return p
}

func (f *File) store(reader io.Reader) error {
	tmp := path.Join(f.volume.tempDir, uuid.NewV4().String())

	fh, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer fh.Close()

	sh1 := sha1.New()
	w := io.MultiWriter(fh, sh1)
	io.Copy(w, reader)
	f.Signature = fmt.Sprintf("%x", sh1.Sum(nil))

	p, err := f.ensurePath()
	if err != nil {
		return err
	}

	err = os.Rename(tmp, p)
	if err != nil {
		return err
	}

	return nil
}

func (f *File) ensurePath() (string, error) {
	p := f.Path()
	dir, _ := path.Split(p)
	err := os.MkdirAll(dir, os.ModePerm)
	return p, err
}

func (f *File) remove() error {
	return os.Remove(f.Path())
}
