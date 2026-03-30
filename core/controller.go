package core

import (
	"fmt"
	"net/http"

	zlog "github.com/zenqos/zenqo/internal/log"
)

// RouteDocMeta holds optional OpenAPI documentation metadata for a route.
// Populate it using the builder methods (Summary, Tags, Body, Response, etc.)
// on the RouteDefinition returned by GET/POST/PUT/PATCH/DELETE.
type RouteDocMeta struct {
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
	RequestBody any // Go type used to infer the request body JSON schema
	Responses   []ResponseDoc
	NoSecurity  bool // true = override global security with empty (public route)
}

// ResponseDoc describes a single HTTP response for OpenAPI schema generation.
type ResponseDoc struct {
	Status int
	Body   any // Go type used to infer the response body JSON schema; nil for empty bodies
}

// RouteDefinition holds the configuration for a single route,
// including its HTTP method, path, handler, and any Guards,
// Interceptors, or Middlewares scoped to that route.
// Use the builder methods (UseGuard, UseInterceptor, Use) to chain configuration.
type RouteDefinition struct {
	Method       string
	Path         string
	HandlerFunc  http.HandlerFunc // set by Handle() for raw net/http handlers
	zenqoHandler HandlerFunc      // set by GET/POST/etc; adapted lazily in RegisterRoutes
	Guards       []Guard
	Interceptors []Interceptor
	Middlewares  []MiddlewareFunc
	Meta         RouteDocMeta // OpenAPI documentation metadata
}

// Summary sets the one-line OpenAPI summary for this route.
func (rd *RouteDefinition) Summary(s string) *RouteDefinition {
	rd.Meta.Summary = s
	return rd
}

// Description sets the longer OpenAPI description for this route.
func (rd *RouteDefinition) Description(s string) *RouteDefinition {
	rd.Meta.Description = s
	return rd
}

// Tags groups this route under one or more OpenAPI tags (e.g. "users", "auth").
func (rd *RouteDefinition) Tags(tags ...string) *RouteDefinition {
	rd.Meta.Tags = append(rd.Meta.Tags, tags...)
	return rd
}

// Deprecated marks this route as deprecated in the OpenAPI spec.
func (rd *RouteDefinition) Deprecated() *RouteDefinition {
	rd.Meta.Deprecated = true
	return rd
}

// Body sets the Go type used to infer the request body JSON schema.
// Pass a zero-value struct: .Body(CreateUserDTO{})
func (rd *RouteDefinition) Body(body any) *RouteDefinition {
	rd.Meta.RequestBody = body
	return rd
}

// Response adds an HTTP response to the OpenAPI spec for this route.
// Pass a zero-value struct as body, or nil for empty bodies (e.g. 204).
// Example: .Response(200, UserDTO{}) or .Response(204, nil)
func (rd *RouteDefinition) Response(status int, body any) *RouteDefinition {
	rd.Meta.Responses = append(rd.Meta.Responses, ResponseDoc{Status: status, Body: body})
	return rd
}

// NoSecurity marks this route as public, overriding any global security schemes.
// The route will have an empty security requirement in the OpenAPI spec,
// and Swagger UI will not show a lock icon for it.
//
// Example:
//
//	c.GET("/public", c.publicHandler).NoSecurity()
func (rd *RouteDefinition) NoSecurity() *RouteDefinition {
	rd.Meta.NoSecurity = true
	return rd
}

// UseGuard attaches one or more Guards to this route.
func (rd *RouteDefinition) UseGuard(guards ...Guard) *RouteDefinition {
	rd.Guards = append(rd.Guards, guards...)
	return rd
}

// UseInterceptor attaches one or more Interceptors to this route.
func (rd *RouteDefinition) UseInterceptor(i ...Interceptor) *RouteDefinition {
	rd.Interceptors = append(rd.Interceptors, i...)
	return rd
}

// Use attaches one or more raw middleware functions to this route.
func (rd *RouteDefinition) Use(mw ...MiddlewareFunc) *RouteDefinition {
	rd.Middlewares = append(rd.Middlewares, mw...)
	return rd
}

// BaseController is the default implementation of the Controller interface.
// Embed it in your handler struct and declare routes with GET/POST/PUT/PATCH/DELETE.
// Guards, Interceptors, and Middlewares can be applied at the controller level
// via UseControllerGuard, UseControllerInterceptor, and UseControllerMiddleware.
type BaseController struct {
	basePath     string
	skipPrefix   bool
	routes       []*RouteDefinition
	guards       []Guard
	interceptors []Interceptor
	middlewares  []MiddlewareFunc
	errHandler   ErrorHandlerFunc
}

// setErrorHandler satisfies the unexported errorHandlerSetter interface used by App.buildRoutes().
func (bc *BaseController) setErrorHandler(fn ErrorHandlerFunc) {
	bc.errHandler = fn
}

// SetBasePath sets the URL prefix for all routes in this controller.
// The path must begin with a '/'. Panics immediately on invalid input
// so misconfiguration is caught at startup, not at request time.
func (bc *BaseController) SetBasePath(p string) {
	if len(p) == 0 {
		panic("zenqo: controller basePath must not be empty")
	}
	if p[0] != '/' {
		panic("zenqo: controller basePath must begin with '/' — got: " + p)
	}
	bc.basePath = p
}

// BasePath returns the URL prefix for this controller.
func (bc *BaseController) BasePath() string { return bc.basePath }

// SkipGlobalPrefix marks this controller to be mounted at the root path,
// bypassing any global prefix set via SetGlobalPrefix.
//
// Example:
//
//	func NewController() *HealthController {
//	    c := &HealthController{}
//	    c.SetBasePath("/health")
//	    c.SkipGlobalPrefix()
//	    return c
//	}
func (bc *BaseController) SkipGlobalPrefix() {
	bc.skipPrefix = true
}

func (bc *BaseController) isSkippingPrefix() bool { return bc.skipPrefix }

// Routes returns all route definitions registered on this controller.
// Implements the RouteProvider interface used by the OpenAPI spec generator.
func (bc *BaseController) Routes() []*RouteDefinition { return bc.routes }

// UseControllerGuard applies Guards to every route in this controller.
func (bc *BaseController) UseControllerGuard(g ...Guard) {
	bc.guards = append(bc.guards, g...)
}

// UseControllerInterceptor applies Interceptors to every route in this controller.
func (bc *BaseController) UseControllerInterceptor(i ...Interceptor) {
	bc.interceptors = append(bc.interceptors, i...)
}

// UseControllerMiddleware applies raw middleware to every route in this controller.
func (bc *BaseController) UseControllerMiddleware(m ...MiddlewareFunc) {
	bc.middlewares = append(bc.middlewares, m...)
}

// GET registers a GET route and returns a RouteDefinition for further configuration.
func (bc *BaseController) GET(p string, h HandlerFunc) *RouteDefinition {
	rd := &RouteDefinition{Method: "GET", Path: p, zenqoHandler: h}
	bc.routes = append(bc.routes, rd)
	return rd
}

// POST registers a POST route and returns a RouteDefinition for further configuration.
// Automatically responds with 201 Created when data is returned.
func (bc *BaseController) POST(p string, h HandlerFunc) *RouteDefinition {
	rd := &RouteDefinition{Method: "POST", Path: p, zenqoHandler: h}
	bc.routes = append(bc.routes, rd)
	return rd
}

// PUT registers a PUT route and returns a RouteDefinition for further configuration.
func (bc *BaseController) PUT(p string, h HandlerFunc) *RouteDefinition {
	rd := &RouteDefinition{Method: "PUT", Path: p, zenqoHandler: h}
	bc.routes = append(bc.routes, rd)
	return rd
}

// PATCH registers a PATCH route and returns a RouteDefinition for further configuration.
func (bc *BaseController) PATCH(p string, h HandlerFunc) *RouteDefinition {
	rd := &RouteDefinition{Method: "PATCH", Path: p, zenqoHandler: h}
	bc.routes = append(bc.routes, rd)
	return rd
}

// DELETE registers a DELETE route and returns a RouteDefinition for further configuration.
// Automatically responds with 204 No Content when nil is returned.
func (bc *BaseController) DELETE(p string, h HandlerFunc) *RouteDefinition {
	rd := &RouteDefinition{Method: "DELETE", Path: p, zenqoHandler: h}
	bc.routes = append(bc.routes, rd)
	return rd
}

// HEAD registers a HEAD route and returns a RouteDefinition for further configuration.
// Useful for checking resource existence and reading response headers without a body.
func (bc *BaseController) HEAD(p string, h HandlerFunc) *RouteDefinition {
	rd := &RouteDefinition{Method: "HEAD", Path: p, zenqoHandler: h}
	bc.routes = append(bc.routes, rd)
	return rd
}

// Handle registers a raw net/http handler for cases that need full ResponseWriter control.
func (bc *BaseController) Handle(method, path string, h http.HandlerFunc) *RouteDefinition {
	return bc.addRoute(method, path, h)
}

// adapt wraps a Zenqo HandlerFunc into a standard http.HandlerFunc.
// JSON serialization, HTTP status mapping, and error routing are handled automatically.
// Errors are dispatched to errHandler — DefaultErrorHandler if none is set.
func adapt(method string, h HandlerFunc, errHandler ErrorHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := h(r)
		if err != nil {
			errHandler(w, r, err)
			return
		}
		if data == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		status := http.StatusOK
		if method == "POST" {
			status = http.StatusCreated
		}
		JSON(w, status, SuccessResponse{true, data})
	}
}

func (bc *BaseController) addRoute(method, path string, handler http.HandlerFunc) *RouteDefinition {
	rd := &RouteDefinition{Method: method, Path: path, HandlerFunc: handler}
	bc.routes = append(bc.routes, rd)
	return rd
}

// RegisterRoutes mounts all declared routes onto the given Router,
// applying controller-level and route-level Guards, Interceptors, and Middlewares
// in the correct order.
//
// Routes are registered at their fully resolved paths (basePath + route.Path)
// rather than using a sub-router group. This allows multiple controllers to share
// the same base path without causing a chi panic, and allows "/" as a valid base path.
func (bc *BaseController) RegisterRoutes(r Router) {
	if bc.basePath == "" {
		panic("zenqo: SetBasePath must be called before the controller is registered")
	}

	errHandler := bc.errHandler
	if errHandler == nil {
		errHandler = DefaultErrorHandler
	}

	for _, route := range bc.routes {
		fullPath := joinPath(bc.basePath, route.Path)

		var handler http.HandlerFunc
		if route.zenqoHandler != nil {
			handler = adapt(route.Method, route.zenqoHandler, errHandler)
		} else {
			handler = route.HandlerFunc
		}

		// Apply route-level middleware from inside out:
		// interceptors → guards → middlewares
		for i := len(route.Interceptors) - 1; i >= 0; i-- {
			handler = applyInterceptor(route.Interceptors[i], handler)
		}
		for i := len(route.Guards) - 1; i >= 0; i-- {
			handler = applyGuard(route.Guards[i], handler, errHandler)
		}
		var h http.Handler = handler
		for i := len(route.Middlewares) - 1; i >= 0; i-- {
			h = route.Middlewares[i](h)
		}

		// Apply controller-level middleware from inside out:
		// interceptors → guards → middlewares
		for i := len(bc.interceptors) - 1; i >= 0; i-- {
			h = InterceptorToMiddleware(bc.interceptors[i])(h)
		}
		for i := len(bc.guards) - 1; i >= 0; i-- {
			h = guardToMiddleware(bc.guards[i], errHandler)(h)
		}
		for i := len(bc.middlewares) - 1; i >= 0; i-- {
			h = bc.middlewares[i](h)
		}

		switch route.Method {
		case "GET":
			r.Get(fullPath, h.ServeHTTP)
		case "POST":
			r.Post(fullPath, h.ServeHTTP)
		case "PUT":
			r.Put(fullPath, h.ServeHTTP)
		case "PATCH":
			r.Patch(fullPath, h.ServeHTTP)
		case "DELETE":
			r.Delete(fullPath, h.ServeHTTP)
		case "HEAD":
			r.Head(fullPath, h.ServeHTTP)
		default:
			zlog.Warn("RegisterRoutes", fmt.Sprintf("unsupported HTTP method %q for path %q — route not registered", route.Method, fullPath))
		}
	}
}
