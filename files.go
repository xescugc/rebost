package main

import (
	"crypto/sha1"
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

func NewFileIn(key, signature string) *FileIn {
	return &FileIn{
		key:       key,
		tmp:       path.Join(tempDir, uuid.NewV4().String()),
		signature: signature,
	}
}

func (f *FileIn) store(b io.ReadCloser) error {
	fh, err := os.OpenFile(f.tmp, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
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

func getFile(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]

	var v []byte
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("files"))
		v = b.Get([]byte(key))
		return nil
	})

	fi := NewFileIn(key, string(v))
	http.ServeFile(w, r, fi.filePath())
	logger.info("OUT " + key + " <- " + fi.filePath())
}

func putFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	key := mux.Vars(r)["key"]

	fi := NewFileIn(key, "")
	err := fi.store(r.Body)
	if err != nil {
		logger.error("IN " + key + " " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":%q}`, err)
		return
	}

	db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket([]byte("files"))
		err = b.Put([]byte(key), []byte(fi.signature))
		return err
	})

	//w.Write([]byte(fmt.Sprintf(`{"checksum":"%s"}`, fi.signature)))
	fmt.Fprintf(w, `{"id":"%s"}`, fi.signature)
	logger.info("PUT " + key + " -> " + fi.filePath())
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
}

func headFile(w http.ResponseWriter, r *http.Request) {
}
