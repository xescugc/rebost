GO_BIN = $(shell which go)

# HACK: This is a trick because when deploying the GO_BIN is undefined (empty)
# so we harcode it to the default Debian installation to be able to use it
ifeq ($(GO_BIN),)
	GO_BIN = /usr/local/go/bin/go
endif

serve:
	gin -p 8000 -a 8001 -b rebost

deps:
	$(GO_BIN) get -u github.com/boltdb/bolt/... \
									 github.com/gorilla/mux \
									 github.com/satori/go.uuid \
									 github.com/shirou/gopsutil \
									 github.com/codegangsta/gin

devDeps:
	$(GO_BIN) get -u github.com/codegangsta/gin
