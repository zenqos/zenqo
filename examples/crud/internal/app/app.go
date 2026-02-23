package app

import (
	"github.com/ftery0/zenqo/core"
	"my-app/internal/config"
	"my-app/internal/user"
)

// New wires all controllers together.
// cfg is used in main.go for Port; pass it to controllers that need DB connections, etc.
// Add UseController calls here when you need new features.
func New(cfg config.Config) *core.App {
	return core.NewApp().
		SetGlobalPrefix("/api/v1").
		UseController(user.NewController())
}
