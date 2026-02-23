package app

import (
	"github.com/ftery0/zenqo/core"
	"my-app/internal/config"
)

// New wires all controllers together.
// cfg is accepted for future use — pass it to controllers that need DB connections, ports, etc.
// Add UseController calls here when you need new features.
func New(cfg config.Config) *core.App {
	return core.NewApp()
	// Example:
	// return core.NewApp().
	// 	UseController(user.NewController()).
	// 	UseController(product.NewController())
}
