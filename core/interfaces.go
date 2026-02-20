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
// Return (false, nil) to respond with 403, or (false, err) for 500.
type Guard interface {
	CanActivate(r *http.Request) (bool, error)
}

// Interceptor hooks into the request lifecycle around a handler.
// Before runs before the handler; After receives the final status code.
type Interceptor interface {
	Before(ctx context.Context, r *http.Request) context.Context
	After(ctx context.Context, w http.ResponseWriter, statusCode int)
}

// MiddlewareFunc is the standard net/http middleware signature.
type MiddlewareFunc func(http.Handler) http.Handler
