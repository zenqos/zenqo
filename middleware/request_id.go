package middleware

import (
  	"context"
  	"crypto/rand"
  	"encoding/hex"
  	"net/http"
  )

type requestIDContextKey struct{}

// RequestID returns a middleware that assigns a unique, random 32-character hex ID
// to each incoming request. The ID is:
//   - Set as the X-Request-ID response header.
//   - Stored in the request context for retrieval via GetRequestID.
//
// Example:
//
//	app.Use(middleware.RequestID())
//
//	func handler(r *http.Request) (any, error) {
//	    id := middleware.GetRequestID(r.Context())
//	    return map[string]string{"requestId": id}, nil
//	}
func RequestID() func(http.Handler) http.Handler {
  	return func(next http.Handler) http.Handler {
      		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            			id := generateRequestID()
            			w.Header().Set("X-Request-ID", id)
            			ctx := context.WithValue(r.Context(), requestIDContextKey{}, id)
            			next.ServeHTTP(w, r.WithContext(ctx))
            		})
      	}
  }

// GetRequestID retrieves the request ID set by the RequestID middleware from ctx.
// Returns an empty string if no request ID is present in the context.
func GetRequestID(ctx context.Context) string {
  	id, _ := ctx.Value(requestIDContextKey{}).(string)
  	return id
  }

// generateRequestID creates a cryptographically random 128-bit (16-byte) hex string.
func generateRequestID() string {
  	b := make([]byte, 16)
  	_, _ = rand.Read(b)
  	return hex.EncodeToString(b)
  }
