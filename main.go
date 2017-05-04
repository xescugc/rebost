package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

var (
	port = "8001"
)

func init() {
	if p := os.Getenv("PORT"); len(p) != 0 {
		port = p
	}
}

func main() {
	c := GetConfig()

	fmt.Printf("%#v\n", c)

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

	logger.info("Listening on http://*:" + port)
	http.ListenAndServe(":"+port, h)
}
