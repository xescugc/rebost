package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

var (
	rootDir string = "./"

	tmpDir  string = path.Join(rootDir, "tmp")
	dataDir string = path.Join(rootDir, "data")
)

type FileIn struct {
	tmp       string
	signature string
	key       string
}

func NewFileIn(key string) *FileIn {

	return &FileIn{
		key: key,
		tmp: path.Join(tmpDir, uuid.NewV4().String()),
	}
}

func (f *FileIn) store(b io.ReadCloser) {
	fh, err := os.OpenFile(f.tmp, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
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
	vars := mux.Vars(r)

	fmt.Fprintf(w, "%#v\n", vars)
}

func putFile(w http.ResponseWriter, r *http.Request) {
	fi := NewFileIn(mux.Vars(r)["key"])
	fi.store(r.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf("{'checksum': '%s', 'status': 'ok'}", fi.signature)))
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
}

func headFile(w http.ResponseWriter, r *http.Request) {
}
