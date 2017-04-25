package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/shirou/gopsutil/disk"
)

func getStatus(w http.ResponseWriter, r *http.Request) {
	u, err := disk.Usage(dataDir)
	if err != nil {
		fmt.Fprint(w, fmt.Sprintf("%#v", err))
	}
	b, err := json.Marshal(u)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)

}
