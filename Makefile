GO_BIN = $(shell which go)

# HACK: This is a trick because when deploying the GO_BIN is undefined (empty)
# so we harcode it to the default Debian installation to be able to use it
ifeq ($(GO_BIN),)
	GO_BIN = /usr/local/go/bin/go
endif

deps:
	$(GO_BIN) get -u github.com/boltdb/bolt/... \
									 github.com/gorilla/mux \
									 github.com/satori/go.uuid
