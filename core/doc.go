// Package core provides the foundation of the Zenqo web framework.
//
// Zenqo uses return-value handlers instead of direct ResponseWriter manipulation.
// Handlers return (data, error) and the framework handles JSON serialization,
// status codes, and error responses automatically.
//
//	app.GET("/users/{id}", func(r *http.Request) (any, error) {
//	    id, err := core.Param[int64](r, "id")
//	    if err != nil {
//	        return nil, err
//	    }
//	    return svc.FindByID(id)
//	})
//
// Key types:
//   - [App] — application instance; use [NewApp] to create
//   - [BaseController] — embed in your controller structs
//   - [Guard] — access control before handler execution
//   - [Interceptor] — lifecycle hooks (before/after handler)
//   - [HandlerFunc] — the (any, error) handler signature
//
// Request helpers:
//   - [Bind] — decode JSON body + validate struct tags
//   - [Param] — type-safe URL path parameters
//   - [BindQuery] / [BindHeader] — read query params and headers
//
// Error types:
//   - [HTTPError] — return specific HTTP status codes
//   - [ValidationError] — per-field validation failures
package core
