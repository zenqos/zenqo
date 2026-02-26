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
	chimw "github.com/go-chi/chi/v5/middleware"
	zlog "github.com/zenqos/zenqo/internal/log"
)

// App is the central application instance.
type App struct {
	router       chi.Router
	modules      []Module
	controllers  []Controller
	globalGuards []Guard
	prefix       string
	buildOnce    sync.Once
	root         BaseController
	errorHandler ErrorHandlerFunc
		shutdownTimeout time.Duration
}

// errorHandlerSetter is satisfied by any Controller that embeds BaseController.
// App.buildRoutes() uses it to propagate the configured error handler.
type errorHandlerSetter interface {
	setErrorHandler(ErrorHandlerFunc)
}

// NewApp creates a new Zenqo application with sensible defaults:
// request ID injection, real-IP resolution, panic recovery, and JSON 404/405 responses.
func NewApp() *App {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(zenqoRecoverer)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		Error(w, 404, "not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		Error(w, 405, "method not allowed")
	})
	a := &App{router: r}
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

func (a *App) SetGlobalPrefix(prefix string) *App {
	a.prefix = prefix
	return a
}

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
	for _, mw := range mws {
		a.router.Use(mw)
	}
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
// Example: UseStatic("/", "./public") serves index.html, CSS, JS, etc.
func (a *App) UseStatic(prefix, dir string) *App {
	fs := http.FileServer(http.Dir(dir))
	a.router.Handle(prefix, http.StripPrefix(prefix, fs))
	a.router.Handle(prefix+"/*", http.StripPrefix(prefix, fs))
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

		for _, g := range a.globalGuards {
			a.router.Use(GuardToMiddleware(g))
		}
		mount := func(cr chi.Router) {
			r := newChiAdapter(cr)
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
			a.router.Route(a.prefix, mount)
		} else {
			mount(a.router)
		}
	})
}

func (a *App) Start(addr string) error {
	started := time.Now()
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

	chi.Walk(a.router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		zlog.Log("Router", fmt.Sprintf("%-6s %s", method, route))
		return nil
	})

	host := addr
	if len(host) > 0 && host[0] == ':' {
		host = "http://localhost" + host
	}
	elapsed := time.Since(started).Milliseconds()
	zlog.Log("Server", fmt.Sprintf("Listening on %s  +%dms", host, elapsed))

	a.router.Use(chimw.Logger)

	srv := &http.Server{
		Addr:         addr,
		Handler:      a.router,
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
	return a.router
}

// URLParam extracts a named URL parameter set by the chi router.
func URLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

// Zlog is the public logger for use inside modules and handlers.
func Zlog(label, msg string) { zlog.Log(label, msg) }

// Zerr is the public error logger for use inside modules and handlers.
func Zerr(label, msg string) { zlog.Err(label, msg) }

type chiAdapter struct{ r chi.Router }

func newChiAdapter(r chi.Router) Router { return &chiAdapter{r: r} }

func (a *chiAdapter) Get(path string, h http.HandlerFunc)    { a.r.Get(path, h) }
func (a *chiAdapter) Post(path string, h http.HandlerFunc)   { a.r.Post(path, h) }
func (a *chiAdapter) Put(path string, h http.HandlerFunc)    { a.r.Put(path, h) }
func (a *chiAdapter) Patch(path string, h http.HandlerFunc)  { a.r.Patch(path, h) }
func (a *chiAdapter) Delete(path string, h http.HandlerFunc) { a.r.Delete(path, h) }

func (a *chiAdapter) Use(mw ...MiddlewareFunc) {
	for _, m := range mw {
		a.r.Use(m)
	}
}

func (a *chiAdapter) Group(path string, fn func(r Router)) {
	a.r.Route(path, func(cr chi.Router) {
		fn(newChiAdapter(cr))
	})
}

