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
		sw := &logStatusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		elapsed := time.Since(start)
		status := sw.statusCode
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

// logStatusWriter captures the status code written by downstream handlers.
// statusCode is initialised to 0; the first call to WriteHeader or Write sets it.
type logStatusWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (sw *logStatusWriter) WriteHeader(code int) {
	if !sw.wroteHeader {
		sw.statusCode = code
		sw.wroteHeader = true
	}
	sw.ResponseWriter.WriteHeader(code)
}

// Write captures the implicit 200 that net/http sends when a handler calls
// Write without first calling WriteHeader.
func (sw *logStatusWriter) Write(b []byte) (int, error) {
	if !sw.wroteHeader {
		sw.statusCode = http.StatusOK
		sw.wroteHeader = true
	}
	return sw.ResponseWriter.Write(b)
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
