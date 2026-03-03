package app

import (
	"github.com/zenqos/zenqo/core"
	"github.com/zenqos/zenqo/middleware"
	"github.com/zenqos/zenqo/openapi"
	"my-app/internal/config"
	"my-app/internal/user"
)

// New creates the application and wires all controllers.
// Add more UseController calls as you build new features.
func New(_ config.Config) *core.App {
	app := core.NewApp()
	app.Use(middleware.SecureHeaders())
	app.SetGlobalPrefix("/api/v1")
	app.UseController(user.NewController())

	// OpenAPI spec: GET /api/v1/openapi.json
	// Swagger UI:   GET /api/v1/docs
	openapi.Mount(app, openapi.Config{
		Title:       "Zenqo CRUD Example",
		Version:     "1.0.0",
		Description: "A simple CRUD API built with Zenqo.",
	})

	return app
}
