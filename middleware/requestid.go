package middleware

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net/http"
	"regexp"
	"sync/atomic"
)

// validRequestID matches safe request ID values: alphanumeric and a small set of
// punctuation characters, capped at 64 bytes to prevent log bloat.
var validRequestID = regexp.MustCompile(`^[a-zA-Z0-9+/=_:@\-]{1,64}$`)

// requestIDKey is the context key for the request ID.
type requestIDKey struct{}

var (
	prefix  string
	counter uint64
)

func init() {
	var buf [4]byte
	_, _ = rand.Read(buf[:])
	prefix = fmt.Sprintf("%x", buf)
}

func nextID() string {
	n := atomic.AddUint64(&counter, 1)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], n)
	return fmt.Sprintf("%s-%x", prefix, buf[2:]) // 6-byte suffix
}

// RequestID is a middleware that injects a unique request ID into each request.
// If the incoming request has an X-Request-Id header, that value is reused;
// otherwise a new ID is generated from an atomic counter with a random prefix.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" || !validRequestID.MatchString(id) {
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
