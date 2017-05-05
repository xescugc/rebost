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

	//var oldSig string
	//oldSig, err = dbSetFileSignature(fi.key, fi.signature)
	//if err != nil {
	//w.WriteHeader(http.StatusInternalServerError)
	//fmt.Fprintf(w, `{"error":%q}`, err)
	//return
	//}

	fmt.Fprintf(w, `{"id":"%s"}`, file.signature)
	//logger.info("IN " + fi.key + " -> " + fi.filePath())

	// Remove old file if it exists
	//if oldSig != "" {
	//fo, err2 := NewFileOut("old", oldSig)
	//if err2 == nil {
	//err = fo.remove()
	//logger.info("FS:RM" + fo.filePath())
	//}
	//}
}
