package middleware

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net/http"
	"sync/atomic"
)

// requestIDKey is the context key for the request ID.
type requestIDKey struct{}

var (
	prefix  string
	counter uint64
)

func init() {
	// Use 8 bytes (64 bits) of random prefix for stronger uniqueness guarantees.
	// This makes the prefix space large enough that prefix collisions across
	// process restarts or multiple instances are astronomically unlikely.
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	prefix = fmt.Sprintf("%x", buf)
}

func nextID() string {
	n := atomic.AddUint64(&counter, 1)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], n)
	return fmt.Sprintf("%s-%x", prefix, buf[2:]) // 6-byte suffix
}

// isValidRequestID reports whether id consists solely of safe ASCII characters
// (alphanumeric plus +/=_:@-) and is between 1 and 64 bytes long.
//
// This replaces a compiled regexp to avoid the regex engine entirely —
// simpler, faster, and carries no regex-overhead risk for adversarial inputs.
func isValidRequestID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if (c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '+' || c == '/' || c == '=' ||
			c == '_' || c == ':' || c == '@' || c == '-' {
			continue
		}
		return false
	}
	return true
}

// RequestID is a middleware that injects a unique request ID into each request.
// If the incoming request has an X-Request-Id header with safe characters, that
// value is reused; otherwise a new ID is generated from an atomic counter with
// a random 8-byte prefix.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if !isValidRequestID(id) {
			id = nextID()
		}
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetReqID returns the request ID from the context.
// Returns an empty string if no request ID is present.
func GetReqID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}
