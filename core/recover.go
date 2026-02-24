package core

import (
	"fmt"
	"net/http"
	"runtime/debug"

	chimw "github.com/go-chi/chi/v5/middleware"
	zlog "github.com/zenqos/zenqo/internal/log"
)

// zenqoRecoverer recovers from panics, logs the panic value and stack trace,
// and returns a JSON 500 response so the server keeps running.
func zenqoRecoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if rvr == http.ErrAbortHandler {
					panic(rvr)
				}
				reqID := chimw.GetReqID(r.Context())
				zlog.Err("Panic", fmt.Sprintf("[%s] %s %s — %v\n%s",
					reqID, r.Method, r.URL.Path, rvr, debug.Stack()))
				InternalError(w, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
