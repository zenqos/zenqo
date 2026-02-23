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
)

// App is the central application instance.
type App struct {
	router       chi.Router
	modules      []Module
	controllers  []Controller
	globalGuards []Guard
	prefix       string
	buildOnce    sync.Once
}

func NewApp() *App {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	return &App{router: r}
}

func (a *App) SetGlobalPrefix(prefix string) *App {
	a.prefix = prefix
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

	zlog("Boot", "Starting application...")

	if len(a.modules) == 0 && len(a.controllers) == 0 {
		zlog("App", "Ready — add controllers in internal/app/app.go")
	}

	for _, m := range a.modules {
		zlog("Module", m.Name()+" initialized")
	}
	for _, c := range a.controllers {
		zlog("Controller", c.BasePath()+" registered")
	}

	chi.Walk(a.router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		zlog("Router", fmt.Sprintf("%-6s %s", method, route))
		return nil
	})

	host := addr
	if host[0] == ':' {
		host = "http://localhost" + host
	}
	elapsed := time.Since(started).Milliseconds()
	zlog("Server", fmt.Sprintf("Listening on %s  +%dms", host, elapsed))

	a.router.Use(chimw.Logger)

	srv := &http.Server{Addr: addr, Handler: a.router}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		zlog("Server", "Shutting down gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	return srv.ListenAndServe()
}

func (a *App) Handler() http.Handler {
	a.buildRoutes()
	return a.router
}

// URLParam extracts a named URL parameter set by the chi router.
func URLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

// zlog prints a structured INFO log in Zenqo format.
func zlog(label, msg string) {
	ts := time.Now().Format("2006/01/02 15:04:05")
	fmt.Fprintf(os.Stdout, "[Zenqo] %s  \033[32mLOG\033[0m  [%s]  %s\n", ts, label, msg)
}

// zerr prints a structured ERROR log in Zenqo format.
func zerr(label, msg string) {
	ts := time.Now().Format("2006/01/02 15:04:05")
	fmt.Fprintf(os.Stderr, "[Zenqo] %s  \033[31mERR\033[0m  [%s]  %s\n", ts, label, msg)
}

// Zlog is the public logger for use inside modules and handlers.
func Zlog(label, msg string) { zlog(label, msg) }

// Zerr is the public error logger for use inside modules and handlers.
func Zerr(label, msg string) { zerr(label, msg) }

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
