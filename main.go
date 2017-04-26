package main

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

var (
	//tmpDb = &db{m: make(map[string]string)}
	//repDb = &db{m: make(map[string]string)}
	port = "8001"
)

func init() {
	if p := os.Getenv("PORT"); len(p) != 0 {
		port = p
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
	s.HandleFunc("", getStatus).Methods("GET")

	r := h.PathPrefix("/replica").Subrouter()
	r.HandleFunc("/in", getReplicaIn).Methods("GET")
	r.HandleFunc("/out", getReplicaOut).Methods("GET")

	http.ListenAndServe(":"+port, h)
}
