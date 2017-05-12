package main

import (
	"net/http"
	"os"

	"github.com/xescugc/rebost/api"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/storage"
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
	c := config.New()
	s := storage.New(c)

	h := api.New(c, s)

	logger.info("Listening on http://*:" + port)
	http.ListenAndServe(":"+port, h)
}
