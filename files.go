package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"

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

	s, err := dbGetFileSignature(key)
	if err != nil {
		logger.warn("OUT " + key + " " + err.Error())
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"warn":%q}`, err)
		return
	}

	var fo *FileIn
	fo, err = NewFileOut(key, s)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":%q}`, err)
		return
	}

	http.ServeFile(w, r, fo.filePath())
	logger.info("OUT " + key + " <- " + fo.filePath())
}

func putFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	fi, err := NewFileIn(mux.Vars(r)["key"], r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":%q}`, err)
		return
	}

	var oldSig string
	oldSig, err = dbSetFileSignature(fi.key, fi.signature)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":%q}`, err)
		return
	}

	fmt.Fprintf(w, `{"id":"%s"}`, fi.signature)
	logger.info("IN " + fi.key + " -> " + fi.filePath())

	// Remove old file if it exists
	if oldSig != "" {
		fo, err2 := NewFileOut("old", oldSig)
		if err2 == nil {
			err = fo.remove()
			logger.info("FS:RM" + fo.filePath())
		}
	}
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	key := mux.Vars(r)["key"]

	s, err := dbGetFileSignature(key)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"warn":%q}`, err)
		return
	}

	var fo *FileIn
	fo, err = NewFileOut(key, s)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":%q}`, err)
		return
	}

	err = dbDelFile(key)
	err = fo.remove() // BUG: need to reverse check if other keys point to the same file!

	w.WriteHeader(http.StatusGone)
	logger.info("DEL(" + key + ") " + fo.filePath())
}

func headFile(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	s, err := dbGetFileSignature(key)
	if err != nil {
		logger.warn("OUT " + key + " " + err.Error())
		w.WriteHeader(http.StatusNotFound)
		return
	}

	logger.info("HEAD " + key)
	w.Header().Set("Signature", s)
}
