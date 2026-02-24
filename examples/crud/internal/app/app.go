package app

import (
	"github.com/ftery0/zenqo/core"
	"github.com/ftery0/zenqo/middleware"
	"my-app/internal/config"
	"my-app/internal/user"
)

// New creates the application and wires all controllers.
// Add more UseController calls as you build new features.
func New(_ config.Config) *core.App {
	app := core.NewApp()
	app.Use(middleware.SecureHeaders())
	return app.
		SetGlobalPrefix("/api/v1").
		UseController(user.NewController())
}
