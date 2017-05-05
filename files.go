package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func NewFileOut(key, signature string) (*FileIn, error) {
	f := &FileIn{
		key:       key,
		signature: signature,
	}

	_, err := os.Stat(f.filePath())

	return f, err
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
