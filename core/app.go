package core

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	zlog "github.com/zenqos/zenqo/internal/log"
	"github.com/zenqos/zenqo/middleware"
)

// App is the central application instance.
type App struct {
	adapter         RouterAdapter
	modules         []Module
	controllers     []Controller
	globalGuards    []Guard
	prefix          string
	buildOnce       sync.Once
	root            BaseController
	errorHandler    ErrorHandlerFunc
	shutdownTimeout time.Duration
	rfc9457         bool
	docsPath        string // set by OpenAPI Mount to show docs URL at startup
}

// errorHandlerSetter is satisfied by any Controller that embeds BaseController.
// App.buildRoutes() uses it to propagate the configured error handler.
type errorHandlerSetter interface {
	setErrorHandler(ErrorHandlerFunc)
}

// urlParamContextKey is the context key used to store the URL parameter resolver
// for the active RouterAdapter. Using a per-request context value instead of a
// package-level variable eliminates data races when multiple App instances coexist
// (e.g. parallel tests each creating their own App).
type urlParamContextKey struct{}

// NewApp creates a new Zenqo application with sensible defaults:
// request ID injection, real-IP resolution, panic recovery, and JSON 404/405 responses.
func NewApp() *App {
	return NewAppWith(newChiRouterAdapter())
}

// NewAppWith creates a new Zenqo application using the provided RouterAdapter.
// Use this to bring your own router while keeping the rest of Zenqo's features.
func NewAppWith(adapter RouterAdapter) *App {
	adapter.Use(middleware.RequestID)
	adapter.Use(middleware.RealIPWithConfig(middleware.RealIPConfig{
		TrustedProxies: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8", "::1/128"},
	}))
	adapter.NotFound(func(w http.ResponseWriter, r *http.Request) {
		Error(w, 404, "not found")
	})
	adapter.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		Error(w, 405, "method not allowed")
	})

	// Capture the adapter's URLParam resolver and inject it into each request's
	// context. This avoids a package-level variable that would be overwritten on
	// every NewAppWith call, causing a data race when multiple App instances exist.
	fn := adapter.URLParam
	adapter.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), urlParamContextKey{}, fn)))
		})
	})

	a := &App{adapter: adapter}
	a.shutdownTimeout = 30 * time.Second
	a.root.basePath = "/"
	return a
}

// SetErrorHandler overrides Zenqo's default error handler for all routes.
// The handler receives (w, r, err) for every error returned by a HandlerFunc.
// Call DefaultErrorHandler(w, r, err) inside your implementation to fall back to built-in behavior.
func (a *App) SetErrorHandler(fn ErrorHandlerFunc) *App {
	a.errorHandler = fn
	return a
}

// UseRFC9457 enables RFC 9457 Problem Details for all error responses.
// When enabled, errors are returned as application/problem+json instead of
// the default ErrorResponse format.
// This also re-registers 404/405 handlers and the panic recoverer to use the RFC format.
func (a *App) UseRFC9457() *App {
	a.rfc9457 = true
	a.errorHandler = RFC9457ErrorHandler
	return a
}

func (a *App) SetGlobalPrefix(prefix string) *App {
	a.prefix = prefix
	return a
}

// Prefix returns the global URL prefix set via SetGlobalPrefix.
// Returns an empty string if no prefix has been configured.
func (a *App) Prefix() string { return a.prefix }

// SetDocsPath stores the OpenAPI docs path for display at startup.
// Called internally by openapi.Mount().
func (a *App) SetDocsPath(path string) { a.docsPath = path }

// SetShutdownTimeout sets the maximum duration to wait for in-flight requests
// to complete when the server receives SIGINT or SIGTERM.
// Default is 30 seconds.
func (a *App) SetShutdownTimeout(d time.Duration) *App {
	a.shutdownTimeout = d
	return a
}

func (a *App) UseGlobalGuard(guards ...Guard) *App {
	a.globalGuards = append(a.globalGuards, guards...)
	return a
}

func (a *App) Use(mws ...MiddlewareFunc) *App {
	a.adapter.Use(mws...)
	return a
}

func (a *App) UseModule(modules ...Module) *App {
	a.modules = append(a.modules, modules...)
	return a
}

// UseController registers one or more Controllers directly without a Module wrapper.
// This is the recommended approach for most projects — no module.go needed.
func (a *App) UseController(controllers ...Controller) *App {
	a.controllers = append(a.controllers, controllers...)
	return a
}

// GET registers a top-level GET route directly on the app.
func (a *App) GET(path string, h HandlerFunc) *RouteDefinition { return a.root.GET(path, h) }

// POST registers a top-level POST route directly on the app.
func (a *App) POST(path string, h HandlerFunc) *RouteDefinition { return a.root.POST(path, h) }

// PUT registers a top-level PUT route directly on the app.
func (a *App) PUT(path string, h HandlerFunc) *RouteDefinition { return a.root.PUT(path, h) }

// PATCH registers a top-level PATCH route directly on the app.
func (a *App) PATCH(path string, h HandlerFunc) *RouteDefinition { return a.root.PATCH(path, h) }

// DELETE registers a top-level DELETE route directly on the app.
func (a *App) DELETE(path string, h HandlerFunc) *RouteDefinition { return a.root.DELETE(path, h) }

// UseStatic serves files from dir under the given URL prefix.
// The global prefix set via SetGlobalPrefix is automatically prepended.
// Example: UseStatic("/", "./public") serves index.html, CSS, JS, etc.
func (a *App) UseStatic(prefix, dir string) *App {
	fs := http.FileServer(http.Dir(dir))
	fullPrefix := a.prefix + prefix
	a.adapter.Handle(fullPrefix, http.StripPrefix(fullPrefix, fs))
	a.adapter.Handle(fullPrefix+"/*", http.StripPrefix(fullPrefix, fs))
	return a
}

func (a *App) buildRoutes() {
	a.buildOnce.Do(func() {
		// Resolve the effective error handler and propagate to all controllers
		// before RegisterRoutes is called so every route gets the right handler.
		errHandler := a.errorHandler
		if errHandler == nil {
			errHandler = DefaultErrorHandler
		}
		a.root.errHandler = errHandler
		for _, c := range a.controllers {
			if s, ok := c.(errorHandlerSetter); ok {
				s.setErrorHandler(errHandler)
			}
		}
		for _, m := range a.modules {
			for _, c := range m.Controllers() {
				if s, ok := c.(errorHandlerSetter); ok {
					s.setErrorHandler(errHandler)
				}
			}
		}

		// RFC 9457: re-register 404/405 and recoverer in problem+json format
		if a.rfc9457 {
			a.adapter.NotFound(func(w http.ResponseWriter, r *http.Request) {
				ProblemJSON(w, ProblemDetail{Status: 404, Instance: r.URL.Path})
			})
			a.adapter.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
				ProblemJSON(w, ProblemDetail{Status: 405, Instance: r.URL.Path})
			})
			a.adapter.Use(zenqoRecovererWith(errHandler))
		}

		for _, g := range a.globalGuards {
			a.adapter.Use(guardToMiddleware(g, errHandler))
		}
		mount := func(r Router) {
			for _, m := range a.modules {
				for _, c := range m.Controllers() {
					c.RegisterRoutes(r)
				}
			}
			for _, c := range a.controllers {
				c.RegisterRoutes(r)
			}
			if len(a.root.routes) > 0 {
				a.root.RegisterRoutes(r)
			}
		}
		if a.prefix != "" {
			a.adapter.Route(a.prefix, func(r Router) {
				mount(r)
			})
		} else {
			a.adapter.Mount(mount)
		}
	})
}

func (a *App) Start(addr string) error {
	started := time.Now()
	a.adapter.Use(middleware.Logger)
	a.buildRoutes()

	zlog.Log("Boot", "Starting application...")

	if len(a.modules) == 0 && len(a.controllers) == 0 && len(a.root.routes) == 0 {
		zlog.Log("App", "Ready — add controllers in internal/app/app.go")
	}

	for _, m := range a.modules {
		zlog.Log("Module", m.Name()+" initialized")
	}
	for _, c := range a.controllers {
		zlog.Log("Controller", c.BasePath()+" registered")
	}

	if w, ok := a.adapter.(Walker); ok {
		_ = w.Walk(func(method, route string, _ http.Handler) error {
			zlog.Log("Router", fmt.Sprintf("%-6s %s", method, route))
			return nil
		})
	}

	host := addr
	if len(host) > 0 && host[0] == ':' {
		host = "http://localhost" + host
	}
	elapsed := time.Since(started).Milliseconds()
	zlog.Log("Server", fmt.Sprintf("Listening on %s  +%dms", host, elapsed))
	if a.docsPath != "" {
		zlog.Log("Server", fmt.Sprintf("API Docs   %s%s", host, a.docsPath))
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      a.adapter.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		zlog.Log("Server", "Shutting down gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			zlog.Err("Server", "Shutdown error: "+err.Error())
		}
	}()

	return srv.ListenAndServe()
}

// Handler builds all routes and returns the underlying http.Handler.
// Use this for testing with httptest or for mounting Zenqo inside another server.
func (a *App) Handler() http.Handler {
	a.buildRoutes()
	return a.adapter.Handler()
}

// URLParam extracts a named URL parameter set by the underlying router.
func URLParam(r *http.Request, key string) string {
	if fn, ok := r.Context().Value(urlParamContextKey{}).(func(*http.Request, string) string); ok {
		return fn(r, key)
	}
	return chi.URLParam(r, key)
}

// Zlog is the public logger for use inside modules and handlers.
func Zlog(label, msg string) { zlog.Log(label, msg) }

// Zerr is the public error logger for use inside modules and handlers.
func Zerr(label, msg string) { zlog.Err(label, msg) }

// --- chi RouterAdapter (default) ---

type chiRouterAdapter struct{ r chi.Router }

func newChiRouterAdapter() *chiRouterAdapter {
	return &chiRouterAdapter{r: chi.NewRouter()}
}

func (a *chiRouterAdapter) Handler() http.Handler { return a.r }

func (a *chiRouterAdapter) URLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

func (a *chiRouterAdapter) Use(mw ...MiddlewareFunc) {
	for _, m := range mw {
		a.r.Use(m)
	}
}

func (a *chiRouterAdapter) Route(pattern string, fn func(r Router)) {
	a.r.Route(pattern, func(cr chi.Router) {
		fn(newChiSubRouter(cr))
	})
}

func (a *chiRouterAdapter) Mount(fn func(r Router)) {
	fn(newChiSubRouter(a.r))
}

func (a *chiRouterAdapter) Handle(pattern string, h http.Handler) {
	a.r.Handle(pattern, h)
}

func (a *chiRouterAdapter) NotFound(h http.HandlerFunc) {
	a.r.NotFound(h)
}

func (a *chiRouterAdapter) MethodNotAllowed(h http.HandlerFunc) {
	a.r.MethodNotAllowed(h)
}

// Walk implements the Walker interface for route introspection.
func (a *chiRouterAdapter) Walk(fn func(method, route string, handler http.Handler) error) error {
	return chi.Walk(a.r, func(method, route string, handler http.Handler, _ ...func(http.Handler) http.Handler) error {
		return fn(method, route, handler)
	})
}

// --- chi sub-router (implements Router for controllers) ---

type chiSubRouter struct{ r chi.Router }

func newChiSubRouter(r chi.Router) Router { return &chiSubRouter{r: r} }

func (a *chiSubRouter) Get(path string, h http.HandlerFunc)    { a.r.Get(path, h) }
func (a *chiSubRouter) Post(path string, h http.HandlerFunc)   { a.r.Post(path, h) }
func (a *chiSubRouter) Put(path string, h http.HandlerFunc)    { a.r.Put(path, h) }
func (a *chiSubRouter) Patch(path string, h http.HandlerFunc)  { a.r.Patch(path, h) }
func (a *chiSubRouter) Delete(path string, h http.HandlerFunc) { a.r.Delete(path, h) }

func (a *chiSubRouter) Use(mw ...MiddlewareFunc) {
	for _, m := range mw {
		a.r.Use(m)
	}
}

func (a *chiSubRouter) Group(path string, fn func(r Router)) {
	a.r.Route(path, func(cr chi.Router) {
		fn(newChiSubRouter(cr))
	})
}
