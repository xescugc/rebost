package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

var (
	tmpDb = &db{m: make(map[string]string)}
	repDb = &db{m: make(map[string]string)}
)

func mvToStore(h string) {
	p := path.Join(".", "data")
	currentDir := []byte{}
	for _, b := range []byte(h) {
		currentDir = append(currentDir, b)
		if len(currentDir) == 2 {
			p = path.Join(p, string(currentDir))
			currentDir = []byte{}
		}
	}
	dir, _ := path.Split(p)
	os.MkdirAll(dir, 0755)
	p += ".jpeg"
	tmpFile, _ := tmpDb.get(h)
	os.Rename(tmpFile, p)
	tmpDb.del(h)
	repDb.set(p, p)
}

func mvToTmp(r *http.Request) (string, error) {
	p := path.Join("./tmp", strconv.FormatInt(time.Now().UnixNano(), 10)+".jpg")
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sh1 := sha1.New()
	w := io.MultiWriter(f, sh1)
	io.Copy(w, r.Body)
	hash := fmt.Sprintf("%x", sh1.Sum(nil))
	tmpDb.set(hash, p)
	mvToStore(hash)
	return fmt.Sprintf("%s-%s", p, hash), nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	p, err := mvToTmp(r)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("{'error': '%v'}", err)))
	} else {
		w.Write([]byte(fmt.Sprintf("{'test': '%s'}", p)))
	}

}

func main() {
	h := mux.NewRouter()

	f := h.PathPrefix("/files").Subrouter()
	f.HandleFunc("/{key:.*}", getFile).Methods("GET")
	f.HandleFunc("/{key:.*}", putFile).Methods("PUT")
	f.HandleFunc("/{key:.*}", deleteFile).Methods("DELETE")
	f.HandleFunc("/{key:.*}", headFile).Methods("HEAD")

	s := h.PathPrefix("/status").Subrouter()
	s.HandleFunc("/", getStatus).Methods("GET")

	r := h.PathPrefix("/replica").Subrouter()
	r.HandleFunc("/in", getReplicaIn).Methods("GET")
	r.HandleFunc("/out", getReplicaOut).Methods("GET")

	http.ListenAndServe(":8000", h)
}
