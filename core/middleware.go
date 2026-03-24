package core

import (
	"errors"
	"net/http"

	"github.com/zenqos/zenqo/internal/httputil"
	zlog "github.com/zenqos/zenqo/internal/log"
)

// GuardToMiddleware converts a Guard into a standard MiddlewareFunc
// so it can be applied at the router or controller level.
//
// Deprecated: Prefer UseControllerGuard on a BaseController or UseGuard on a
// RouteDefinition. Those APIs integrate directly with the framework's error
// handler and do not require wiring ErrorHandlerFunc manually.
//
// An optional ErrorHandlerFunc can be passed to route rejections through
// a custom error handler (e.g. for RFC 9457 problem+json responses).
// When omitted, the default HTTP error format is used.
//
// Status code logic:
//   - (true, _)            → next handler runs
//   - (false, *HTTPError)  → responds with the HTTPError's Status and Message
//   - (false, nil)         → 403 Forbidden
//   - (false, other error) → 500 Internal Server Error
func GuardToMiddleware(g Guard, errHandler ...ErrorHandlerFunc) MiddlewareFunc {
	if len(errHandler) > 0 && errHandler[0] != nil {
		eh := errHandler[0]
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				allowed, err := g.CanActivate(r)
				if !allowed {
					guardRejectWith(w, r, err, eh)
					return
				}
				next.ServeHTTP(w, r)
			})
		}
	}
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
			sw := &httputil.StatusWriter{ResponseWriter: w, StatusCode: http.StatusOK}
			next.ServeHTTP(sw, r)
			i.After(ctx, w, sw.StatusCode)
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
		var he *HTTPError
		if errors.As(err, &he) {
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
// Delegates to GuardToMiddleware to avoid duplicating rejection logic.
func guardToMiddleware(g Guard, errHandler ErrorHandlerFunc) MiddlewareFunc {
	return GuardToMiddleware(g, errHandler)
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
		sw := &httputil.StatusWriter{ResponseWriter: w, StatusCode: http.StatusOK}
		next(sw, r)
		i.After(ctx, w, sw.StatusCode)
	}
}

// statusWriter is a package-level alias for httputil.StatusWriter.
// It is used by Interceptors and the panic recoverer to capture the HTTP status code
// written by downstream handlers.
type statusWriter = httputil.StatusWriter
