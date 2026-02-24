package core

import (
	"bufio"
	"fmt"
	"net"
	"net/http"

	zlog "github.com/zenqos/zenqo/internal/log"
)

// GuardToMiddleware converts a Guard into a standard MiddlewareFunc
// so it can be applied at the router or controller level.
func GuardToMiddleware(g Guard) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, err := g.CanActivate(r)
			if err != nil {
				zlog.Err("Guard", err.Error())
				JSON(w, 500, ErrorResponse{Code: 500, Message: "internal server error"})
				return
			}
			if !allowed {
				JSON(w, 403, ErrorResponse{Code: 403, Message: "access denied"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// InterceptorToMiddleware converts an Interceptor into a standard MiddlewareFunc
// so it can be applied at the router or controller level.
func InterceptorToMiddleware(i Interceptor) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := i.Before(r.Context(), r)
			r = r.WithContext(ctx)
			sw := &statusWriter{ResponseWriter: w, statusCode: 200}
			next.ServeHTTP(sw, r)
			i.After(ctx, w, sw.statusCode)
		})
	}
}

// applyGuard wraps a single HandlerFunc with Guard logic at the route level.
func applyGuard(g Guard, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed, err := g.CanActivate(r)
		if err != nil {
			zlog.Err("Guard", err.Error())
			JSON(w, 500, ErrorResponse{Code: 500, Message: "internal server error"})
			return
		}
		if !allowed {
			JSON(w, 403, ErrorResponse{Code: 403, Message: "access denied"})
			return
		}
		next(w, r)
	}
}

// applyInterceptor wraps a single HandlerFunc with Interceptor logic at the route level.
func applyInterceptor(i Interceptor, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := i.Before(r.Context(), r)
		r = r.WithContext(ctx)
		sw := &statusWriter{ResponseWriter: w, statusCode: 200}
		next(sw, r)
		i.After(ctx, w, sw.statusCode)
	}
}

// statusWriter wraps http.ResponseWriter to capture the written status code
// so Interceptors can observe it in their After hook.
// It also delegates http.Flusher and http.Hijacker so that streaming
// responses (SSE) and connection upgrades (WebSocket) work correctly.
type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.statusCode = code
	sw.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher, enabling streaming responses such as SSE.
// It is a no-op if the underlying ResponseWriter does not support flushing.
func (sw *statusWriter) Flush() {
	if f, ok := sw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker, enabling WebSocket and other connection upgrades.
// Returns an error if the underlying ResponseWriter does not support hijacking.
func (sw *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := sw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("zenqo: underlying ResponseWriter does not support hijacking")
	}
	return h.Hijack()
}
