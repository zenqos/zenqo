package core

import (
	"errors"
	"net/http"
)

// RouteDefinition holds the configuration for a single route,
// including its HTTP method, path, handler, and any Guards,
// Interceptors, or Middlewares scoped to that route.
// Use the builder methods (UseGuard, UseInterceptor, Use) to chain configuration.
type RouteDefinition struct {
	Method       string
	Path         string
	HandlerFunc  http.HandlerFunc
	Guards       []Guard
	Interceptors []Interceptor
	Middlewares  []MiddlewareFunc
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
	routes       []*RouteDefinition
	guards       []Guard
	interceptors []Interceptor
	middlewares  []MiddlewareFunc
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
	return bc.addRoute("GET", p, adapt("GET", h))
}

// POST registers a POST route and returns a RouteDefinition for further configuration.
// Automatically responds with 201 Created when data is returned.
func (bc *BaseController) POST(p string, h HandlerFunc) *RouteDefinition {
	return bc.addRoute("POST", p, adapt("POST", h))
}

// PUT registers a PUT route and returns a RouteDefinition for further configuration.
func (bc *BaseController) PUT(p string, h HandlerFunc) *RouteDefinition {
	return bc.addRoute("PUT", p, adapt("PUT", h))
}

// PATCH registers a PATCH route and returns a RouteDefinition for further configuration.
func (bc *BaseController) PATCH(p string, h HandlerFunc) *RouteDefinition {
	return bc.addRoute("PATCH", p, adapt("PATCH", h))
}

// DELETE registers a DELETE route and returns a RouteDefinition for further configuration.
// Automatically responds with 204 No Content when nil is returned.
func (bc *BaseController) DELETE(p string, h HandlerFunc) *RouteDefinition {
	return bc.addRoute("DELETE", p, adapt("DELETE", h))
}

// Handle registers a raw net/http handler for cases that need full ResponseWriter control.
func (bc *BaseController) Handle(method, path string, h http.HandlerFunc) *RouteDefinition {
	return bc.addRoute(method, path, h)
}

// adapt converts a Zenqo HandlerFunc into a standard http.HandlerFunc.
// It handles JSON serialization and HTTP error mapping automatically.
func adapt(method string, h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := h(r)
		if err != nil {
			var he *HTTPError
			if errors.As(err, &he) {
				Error(w, he.Status, he.Message)
			} else {
				InternalError(w, "internal server error")
			}
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
func (bc *BaseController) RegisterRoutes(r Router) {
	if bc.basePath == "" {
		panic("zenqo: SetBasePath must be called before the controller is registered")
	}
	r.Group(bc.basePath, func(r Router) {
		r.Use(bc.middlewares...)
		for _, g := range bc.guards {
			r.Use(GuardToMiddleware(g))
		}
		for _, i := range bc.interceptors {
			r.Use(InterceptorToMiddleware(i))
		}

		for _, route := range bc.routes {
			handler := route.HandlerFunc
			for i := len(route.Interceptors) - 1; i >= 0; i-- {
				handler = applyInterceptor(route.Interceptors[i], handler)
			}
			for i := len(route.Guards) - 1; i >= 0; i-- {
				handler = applyGuard(route.Guards[i], handler)
			}
			var h http.Handler = handler
			for i := len(route.Middlewares) - 1; i >= 0; i-- {
				h = route.Middlewares[i](h)
			}
			switch route.Method {
			case "GET":
				r.Get(route.Path, h.ServeHTTP)
			case "POST":
				r.Post(route.Path, h.ServeHTTP)
			case "PUT":
				r.Put(route.Path, h.ServeHTTP)
			case "PATCH":
				r.Patch(route.Path, h.ServeHTTP)
			case "DELETE":
				r.Delete(route.Path, h.ServeHTTP)
			}
		}
	})
}
