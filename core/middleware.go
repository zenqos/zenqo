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
//
// Status code logic:
//   - (true, _)            → next handler runs
//   - (false, *HTTPError)  → responds with the HTTPError's Status and Message
//   - (false, nil)         → 403 Forbidden
//   - (false, other error) → 500 Internal Server Error
func GuardToMiddleware(g Guard) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, err := g.CanActivate(r)
			if !allowed {
				guardReject(w, err)
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
// When errHandler is non-nil, guard rejections are routed through it
// (e.g. for RFC 9457 problem+json responses).
func applyGuard(g Guard, next http.HandlerFunc, errHandler ErrorHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed, err := g.CanActivate(r)
		if !allowed {
			if errHandler != nil {
				guardRejectWith(w, r, err, errHandler)
			} else {
				guardReject(w, err)
			}
			return
		}
		next(w, r)
	}
}

// guardReject writes the appropriate error response when a Guard denies access.
func guardReject(w http.ResponseWriter, err error) {
	if err != nil {
		if he, ok := err.(*HTTPError); ok {
			JSON(w, he.Status, ErrorResponse{Code: he.Status, Message: he.Message})
			return
		}
		zlog.Err("Guard", err.Error())
		JSON(w, 500, ErrorResponse{Code: 500, Message: "internal server error"})
		return
	}
	JSON(w, 403, ErrorResponse{Code: 403, Message: "access denied"})
}

// guardToMiddleware converts a Guard into a MiddlewareFunc that routes
// rejections through the given ErrorHandlerFunc.
func guardToMiddleware(g Guard, errHandler ErrorHandlerFunc) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, err := g.CanActivate(r)
			if !allowed {
				guardRejectWith(w, r, err, errHandler)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// guardRejectWith routes a guard rejection through the given error handler.
func guardRejectWith(w http.ResponseWriter, r *http.Request, err error, errHandler ErrorHandlerFunc) {
	if err != nil {
		errHandler(w, r, err)
		return
	}
	errHandler(w, r, ErrForbidden("access denied"))
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
