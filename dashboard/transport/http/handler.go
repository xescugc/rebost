package http

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/xescugc/rebost/dashboard"
	"github.com/xescugc/rebost/dashboard/templates"
	"github.com/xescugc/rebost/dashboard/transport"
)

// MakeHandler initializes the router for the Dashboard
func MakeHandler(s dashboard.Service, l *slog.Logger) http.Handler {
	r := mux.NewRouter()

	r.Methods(http.MethodGet).Path("/").HandlerFunc(homeHandler(s))

	return r
}

func homeHandler(s dashboard.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ns, err := s.ListNodes(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		t, _ := templates.Templates["views/dashboard/index.tmpl"]
		t.Execute(w, transport.HomeResponse{Nodes: ns})
	}
}
