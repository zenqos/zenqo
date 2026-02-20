package core

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

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
	RegisterRoutes(r chi.Router)
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
