// Package httputil provides shared HTTP utility types for the Zenqo framework.
package httputil

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

// StatusWriter wraps http.ResponseWriter to capture the HTTP status code written
// by downstream handlers. It correctly delegates Flush and Hijack so that
// streaming (SSE) and connection-upgrade (WebSocket) responses work through it.
type StatusWriter struct {
	http.ResponseWriter
	StatusCode  int
	WroteHeader bool
}

// WriteHeader captures the status code and delegates to the underlying writer.
// Subsequent calls are no-ops to prevent superfluous-WriteHeader log warnings.
func (sw *StatusWriter) WriteHeader(code int) {
	if sw.WroteHeader {
		return
	}
	sw.StatusCode = code
	sw.WroteHeader = true
	sw.ResponseWriter.WriteHeader(code)
}

// Write captures an implicit 200 when a handler writes without calling WriteHeader first.
func (sw *StatusWriter) Write(b []byte) (int, error) {
	if !sw.WroteHeader {
		sw.StatusCode = http.StatusOK
		sw.WroteHeader = true
	}
	return sw.ResponseWriter.Write(b)
}

// Flush implements http.Flusher, enabling streaming responses such as SSE.
// It is a no-op if the underlying ResponseWriter does not support flushing.
func (sw *StatusWriter) Flush() {
	if f, ok := sw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker, enabling WebSocket and other connection upgrades.
// Returns an error if the underlying ResponseWriter does not support hijacking.
func (sw *StatusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := sw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("zenqo: underlying ResponseWriter does not support hijacking")
	}
	return h.Hijack()
}
