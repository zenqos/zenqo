package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	zlog "github.com/zenqos/zenqo/internal/log"
)

// Logger is a middleware that logs each request's method, path, status code,
// and duration using Zenqo's internal logger.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &logStatusWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(sw, r)
		elapsed := time.Since(start)
		zlog.Log("HTTP", fmt.Sprintf("%-6s %s  %d  %s",
			r.Method, sanitizeLogValue(r.URL.Path), sw.statusCode, elapsed.Round(time.Microsecond)))
	})
}

// sanitizeLogValue replaces newline and carriage-return characters with their
// escape sequences to prevent log injection attacks via crafted URL paths.
func sanitizeLogValue(s string) string {
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}

// logStatusWriter captures the status code written by downstream handlers.
type logStatusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (sw *logStatusWriter) WriteHeader(code int) {
	sw.statusCode = code
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *logStatusWriter) Flush() {
	if f, ok := sw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (sw *logStatusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := sw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("zenqo: underlying ResponseWriter does not support hijacking")
	}
	return h.Hijack()
}
