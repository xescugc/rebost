package httptransport

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/xescugc/rebost/logevent"
)

// AuditMiddleware logs S3 object create/access/stat/delete events with the
// parsed key and caller IP. Routes that do not have both a "bucket" and "key"
// mux variable (e.g. /replicas/, /config) are silently skipped.
func AuditMiddleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)

			vars := mux.Vars(r)
			bucket, hasBucket := vars["bucket"]
			key, hasKey := vars["key"]
			if !hasBucket || !hasKey {
				return
			}

			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
				host = strings.SplitN(fwd, ",", 2)[0]
			}

			logger.Info("audit",
				"event", auditEventForMethod(r.Method),
				"method", r.Method,
				"key", bucket+"/"+key,
				"caller_ip", host,
				"status", rw.status,
				"time", time.Now().UTC().Format(time.RFC3339),
			)
		})
	}
}

func auditEventForMethod(method string) string {
	switch method {
	case http.MethodPut:
		return logevent.AuditCreate
	case http.MethodDelete:
		return logevent.AuditDelete
	case http.MethodGet:
		return logevent.AuditAccess
	case http.MethodHead:
		return logevent.AuditStat
	default:
		return "audit.unknown"
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
