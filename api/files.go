package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func putFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	file, err := n.storage.Add(mux.Vars(r)["key"], r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":%q}`, err)
		return
	}
	fmt.Fprintf(w, `{"id":"%s"}`, file.Signature)
}

func getFile(w http.ResponseWriter, r *http.Request) {
	f, err := n.storage.Get(mux.Vars(r)["key"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":%q}`, err)
		return
	}
	http.ServeFile(w, r, f.Path())
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	err := n.storage.Delete(mux.Vars(r)["key"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":%q}`, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
