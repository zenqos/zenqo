package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zenqos/zenqo/internal/httputil"
	zlog "github.com/zenqos/zenqo/internal/log"
)

// Logger is a middleware that logs each request's method, path, status code,
// and duration using Zenqo's internal logger.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &httputil.StatusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		elapsed := time.Since(start)
		status := sw.StatusCode
		if status == 0 {
			status = http.StatusOK
		}
		zlog.Log("HTTP", fmt.Sprintf("%-6s %s  %d  %s",
			r.Method, sanitizeLogValue(r.URL.Path), status, elapsed.Round(time.Microsecond)))
	})
}

// sanitizeLogValue replaces newline and carriage-return characters with their
// escape sequences to prevent log injection attacks via crafted URL paths.
func sanitizeLogValue(s string) string {
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}

