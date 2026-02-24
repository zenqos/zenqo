package app

import (
	"net/http"

	"github.com/zenqos/zenqo/core"
	"github.com/zenqos/zenqo/middleware"
	"my-app/internal/config"
)

// New creates the application with direct routing — no controller boilerplate needed.
func New(_ config.Config) *core.App {
	app := core.NewApp()
	app.Use(middleware.SecureHeaders())

	app.GET("/", func(r *http.Request) (any, error) {
		return map[string]string{"message": "Welcome to Zenqo!"}, nil
	})

	app.GET("/health", func(r *http.Request) (any, error) {
		return map[string]string{"status": "ok"}, nil
	})

	return app
}
