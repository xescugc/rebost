package http

import (
	"context"
	"net/http"

	"github.com/go-kit/kit/log"
	kittransport "github.com/go-kit/kit/transport"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/xescugc/rebost/dashboard"
	"github.com/xescugc/rebost/dashboard/templates"
	"github.com/xescugc/rebost/dashboard/transport"
)

// MakeHandler initializes the router for the Dashboard
func MakeHandler(s dashboard.Service, l log.Logger) http.Handler {
	r := mux.NewRouter()
	e := transport.MakeServerEndpoints(s)

	options := []kithttp.ServerOption{
		kithttp.ServerErrorHandler(kittransport.NewLogErrorHandler(l)),
	}

	r.Methods(http.MethodGet).Path("/").Handler(kithttp.NewServer(
		e.Home,
		decodeHomeRequest,
		encodeHomeResponse,
		options...,
	))
	return r
}

func decodeHomeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return transport.HomeRequest{}, nil
}

func encodeHomeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	res := response.(transport.HomeResponse)
	t, _ := templates.Templates["views/dashboard/index.tmpl"]
	return t.Execute(w, res)
}
