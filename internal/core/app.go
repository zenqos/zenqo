package core

import (
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// App is the central application instance.
// It wires modules together, applies global middleware and guards,
// and starts the HTTP server.
type App struct {
	router       chi.Router
	modules      []Module
	globalGuards []Guard
	prefix       string
	buildOnce    sync.Once // ensures routes are registered exactly once, even under concurrent access
}

// NewApp creates a new App with sensible defaults:
// RequestID, RealIP, Logger, and Recoverer middleware are pre-registered.
func NewApp() *App {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	return &App{router: r}
}

// SetGlobalPrefix sets a URL prefix applied to all registered routes (e.g. "/api/v1").
func (a *App) SetGlobalPrefix(prefix string) *App {
	a.prefix = prefix
	return a
}

// UseGlobalGuard registers Guards that run on every request, before any route-level guards.
func (a *App) UseGlobalGuard(guards ...Guard) *App {
	a.globalGuards = append(a.globalGuards, guards...)
	return a
}

// Use registers raw middleware on the root router.
func (a *App) Use(mws ...MiddlewareFunc) *App {
	for _, mw := range mws {
		a.router.Use(mw)
	}
	return a
}

// UseModule registers one or more Modules and their Controllers with the application.
func (a *App) UseModule(modules ...Module) *App {
	for _, m := range modules {
		a.modules = append(a.modules, m)
		log.Printf("[Zenqo] Module registered: %s", m.Name())
	}
	return a
}

// buildRoutes registers all routes exactly once.
// sync.Once guarantees this is safe under concurrent access (e.g. parallel tests).
func (a *App) buildRoutes() {
	a.buildOnce.Do(func() {
		for _, g := range a.globalGuards {
			a.router.Use(GuardToMiddleware(g))
		}

		routeHandler := func(r chi.Router) {
			for _, m := range a.modules {
				for _, c := range m.Controllers() {
					c.RegisterRoutes(r)
					log.Printf("[Zenqo]   → Controller: %s%s", a.prefix, c.BasePath())
				}
			}
		}
		if a.prefix != "" {
			a.router.Route(a.prefix, routeHandler)
		} else {
			routeHandler(a.router)
		}
	})
}

// Start builds all routes and starts the HTTP server on the given address (e.g. ":3000").
func (a *App) Start(addr string) error {
	a.buildRoutes()

	totalRoutes := 0
	chi.Walk(a.router, func(_, _ string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		totalRoutes++
		return nil
	})
	if totalRoutes == 0 {
		log.Printf("[Zenqo] WARNING: no routes registered — did you forget to call UseModule?")
	}

	log.Printf("[Zenqo] Modules  : %d", len(a.modules))
	for _, m := range a.modules {
		log.Printf("[Zenqo]   📦 %s (%d controllers)", m.Name(), len(m.Controllers()))
	}
	chi.Walk(a.router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		log.Printf("[Zenqo] Route    : %-6s %s", method, route)
		return nil
	})
	log.Printf("[Zenqo] Listening on %s", addr)

	return http.ListenAndServe(addr, a.router)
}

// Handler builds all routes and returns the underlying http.Handler.
// Intended for use with httptest in unit and integration tests.
// Safe to call multiple times — routes are only registered once.
func (a *App) Handler() http.Handler {
	a.buildRoutes()
	return a.router
}
