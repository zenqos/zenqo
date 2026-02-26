package middleware

import (
  	"fmt"
  	"net/http"
  	"time"

  	zlog "github.com/zenqos/zenqo/internal/log"
  )

// responseRecorder wraps http.ResponseWriter to capture the HTTP status code
// written by the downstream handler.
type responseRecorder struct {
  	http.ResponseWriter
  	status int
  }

func (rr *responseRecorder) WriteHeader(code int) {
  	rr.status = code
  	rr.ResponseWriter.WriteHeader(code)
  }

// Logger returns a middleware that logs each request with its HTTP method,
// request path, response status code, elapsed time, and client IP address.
//
// Example:
//
//	app.Use(middleware.Logger())
func Logger() func(http.Handler) http.Handler {
  	return func(next http.Handler) http.Handler {
      		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            			start := time.Now()
            			rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
            			next.ServeHTTP(rec, r)

            			duration := time.Since(start)
            			msg := fmt.Sprintf("%d | %-7s | %-30s | %s | %v",
                                     				rec.status, r.Method, r.URL.Path, r.RemoteAddr, duration)
            			zlog.Log("Logger", msg)
            		})
      	}
  }
