package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/storage"
)

type Node struct {
	config  *config.Config
	storage *storage.Storage
}

var n *Node

func New(c *config.Config, s *storage.Storage) http.Handler {
	n = &Node{c, s}

	h := mux.NewRouter()

	f := h.PathPrefix("/files").Subrouter()
	//f.HandleFunc("/{key:.*}", getFile).Methods("GET")
	f.HandleFunc("/{key:.*}", putFile).Methods("PUT")
	//f.HandleFunc("/{key:.*}", deleteFile).Methods("DELETE")
	//f.HandleFunc("/{key:.*}", headFile).Methods("HEAD")

	//s := h.PathPrefix("/status").Subrouter()
	//s.HandleFunc("", getStatus).Methods("GET")

	//r := h.PathPrefix("/replica").Subrouter()
	//r.HandleFunc("/in", getReplicaIn).Methods("GET")
	//r.HandleFunc("/out", getReplicaOut).Methods("GET")

	return h
}
