package core

import (
	"fmt"
	"net/http"
	"runtime/debug"

	zlog "github.com/zenqos/zenqo/internal/log"
	"github.com/zenqos/zenqo/middleware"
)

// zenqoRecoverer recovers from panics, logs the panic value and stack trace,
// and returns a JSON 500 response so the server keeps running.
//
// If a handler has already begun writing a response (headers sent or body
// partially written), the 500 error response is skipped to avoid sending a
// malformed/mixed response to the client.
func zenqoRecoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, StatusCode: http.StatusOK}
		defer func() {
			if rvr := recover(); rvr != nil {
				if rvr == http.ErrAbortHandler {
					panic(rvr)
				}
				reqID := middleware.GetReqID(r.Context())
				zlog.Err("Panic", fmt.Sprintf("[%s] %s %s — %v\n%s",
					reqID, r.Method, r.URL.Path, rvr, debug.Stack()))
				// Only write the error response if headers have not been sent yet.
				// Writing a second response after a partial write produces a
				// malformed body and a "superfluous response.WriteHeader" log warning.
				if !sw.WroteHeader {
					InternalError(w, "internal server error")
				}
			}
		}()
		next.ServeHTTP(sw, r)
	})
}

// zenqoRecovererWith returns a panic recovery middleware that routes the
// error through the given ErrorHandlerFunc (e.g. RFC9457ErrorHandler).
func zenqoRecovererWith(errHandler ErrorHandlerFunc) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusWriter{ResponseWriter: w, StatusCode: http.StatusOK}
			defer func() {
				if rvr := recover(); rvr != nil {
					if rvr == http.ErrAbortHandler {
						panic(rvr)
					}
					reqID := middleware.GetReqID(r.Context())
					zlog.Err("Panic", fmt.Sprintf("[%s] %s %s — %v\n%s",
						reqID, r.Method, r.URL.Path, rvr, debug.Stack()))
					if !sw.WroteHeader {
						errHandler(w, r, ErrInternal("internal server error"))
					}
				}
			}()
			next.ServeHTTP(sw, r)
		})
	}
}
