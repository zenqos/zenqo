package core

import (
	"context"
	"net/http"
)

// Router is Zenqo's route registration interface.
// Controllers use this to declare routes without depending on any specific router library.
// The underlying implementation is an internal detail of the framework.
type Router interface {
	Get(path string, h http.HandlerFunc)
	Post(path string, h http.HandlerFunc)
	Put(path string, h http.HandlerFunc)
	Patch(path string, h http.HandlerFunc)
	Delete(path string, h http.HandlerFunc)
	Use(mw ...MiddlewareFunc)
	Group(path string, fn func(r Router))
}

// Module groups related Controllers into a single functional unit.
// Each module is self-contained and declares its own dependencies.
type Module interface {
	Name() string
	Controllers() []Controller
}

// Controller defines HTTP routes under a specific base path.
// Embed BaseController to get a builder-style route registration API.
type Controller interface {
	BasePath() string
	RegisterRoutes(r Router)
}

// Guard controls access to a route before the handler is invoked.
//
// Return values:
//   - (true, _)            → request proceeds to the handler
//   - (false, *HTTPError)  → responds with the HTTPError's Status and Message (e.g. 401, 429)
//   - (false, nil)         → 403 Forbidden
//   - (false, other error) → 500 Internal Server Error
type Guard interface {
	CanActivate(r *http.Request) (bool, error)
}

// Interceptor hooks into the request lifecycle around a handler.
// Before runs before the handler; After receives the final status code.
type Interceptor interface {
	Before(ctx context.Context, r *http.Request) context.Context
	After(ctx context.Context, w http.ResponseWriter, statusCode int)
}

// HandlerFunc is the Zenqo return-value handler signature.
// Return (data, nil) for a successful response — the framework serializes it as JSON automatically.
// Return (nil, core.ErrNotFound("msg")) to send an HTTP error with the correct status code.
// Return (nil, nil) to send 204 No Content (useful for DELETE).
//
// Status code rules (applied automatically):
//
//	POST with data  → 201 Created
//	data == nil     → 204 No Content
//	everything else → 200 OK
type HandlerFunc func(*http.Request) (any, error)

// MiddlewareFunc is the standard net/http middleware signature.
type MiddlewareFunc func(http.Handler) http.Handler

// RouterAdapter is Zenqo's router backend interface.
// The default implementation wraps chi. Pass a custom adapter to [NewAppWith]
// to use a different router (e.g. standard library mux, gin, etc.).
type RouterAdapter interface {
	Handler() http.Handler
	URLParam(r *http.Request, key string) string
	Use(mw ...MiddlewareFunc)
	Route(pattern string, fn func(r Router))
	Mount(fn func(r Router))
	Handle(pattern string, h http.Handler)
	NotFound(h http.HandlerFunc)
	MethodNotAllowed(h http.HandlerFunc)
}

// Walker is an optional interface that a [RouterAdapter] can implement
// to support route introspection (e.g. printing registered routes at startup).
type Walker interface {
	Walk(fn func(method, route string, handler http.Handler) error) error
}
