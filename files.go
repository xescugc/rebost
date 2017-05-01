package main

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

var (
	rootDir string = "./"
	tempDir string = path.Join(rootDir, "data", "tmp")
	dataDir string = path.Join(rootDir, "data", "files")
)

type FileIn struct {
	tmp       string
	signature string
	key       string
}

// Ensure dirs exists
func init() {
	err := os.MkdirAll(tempDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(dataDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
}

func NewFileIn(key string, data io.ReadCloser) (*FileIn, error) {
	f := &FileIn{
		key: key,
		tmp: path.Join(tempDir, uuid.NewV4().String()),
	}

	err := f.store(data)
	return f, err
}

func NewFileOut(key, signature string) (*FileIn, error) {
	f := &FileIn{
		key:       key,
		signature: signature,
	}

	_, err := os.Stat(f.filePath())

	return f, err
}

func (f *FileIn) store(b io.ReadCloser) error {
	fh, err := os.OpenFile(f.tmp, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		logger.error("FS:STORE " + f.key + " " + err.Error())
		return err
	}
	defer fh.Close()
	sh1 := sha1.New()
	w := io.MultiWriter(fh, sh1)
	io.Copy(w, b)
	f.signature = fmt.Sprintf("%x", sh1.Sum(nil))

	p := f.filePath()
	dir, _ := path.Split(p)
	os.MkdirAll(dir, 0755)
	os.Rename(f.tmp, p)
	f.tmp = ""
	return nil
}

func (f *FileIn) filePath() string {
	p := dataDir
	currentDir := []byte{}
	for _, b := range []byte(f.signature) {
		currentDir = append(currentDir, b)
		if len(currentDir) == 3 {
			p = path.Join(p, string(currentDir))
			currentDir = []byte{}
		}
	}
	return p
}

func (f *FileIn) remove() error {
	if f.tmp != "" {
		return os.Remove(f.tmp)
	}

	return os.Remove(f.filePath())
}

func getFile(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]

	var v []byte
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("files"))
		v = b.Get([]byte(key))
		if v == nil {
			return errors.New("Missing file")
		}
		return nil
	})
	if err != nil {
		logger.warn("OUT " + key + " " + err.Error())
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"warn":%q}`, err)
		return
	}

	var fo *FileIn
	fo, err = NewFileOut(key, string(v))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":%q}`, err)
		return
	}

	http.ServeFile(w, r, fo.filePath())
	logger.info("OUT " + key + " <- " + fo.filePath())
}

func putFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	key := mux.Vars(r)["key"]

	fi, err := NewFileIn(key, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":%q}`, err)
		return
	}

	var old string
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("files"))
		old = string(b.Get([]byte(key)))
		err := b.Put([]byte(key), []byte(fi.signature))
		if err != nil {
			logger.error("DB:PUT " + err.Error())
			return err
		}

		return nil
	})
	// Remove old file if it exists
	if err == nil && old != "" && old != key {
		fo, err2 := NewFileOut(key, old)
		if err2 == nil {
			err = fo.remove()
		}
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":%q}`, err)
		return
	}

	fmt.Fprintf(w, `{"id":"%s"}`, fi.signature)
	logger.info("IN " + key + " -> " + fi.filePath())
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
}

func headFile(w http.ResponseWriter, r *http.Request) {
}
