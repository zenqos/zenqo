// Package main demonstrates the minimal setup for a Zenqo application.
// Run: go run examples/basic/main.go
// Test: curl http://localhost:3000/api/v1/hello
package main

import (
	"log"
	"net/http"

	"zenqo/core"
)

// ── Controller ───────────────────────────────────────────────────────────────

type HelloController struct {
	core.BaseController
}

func NewHelloController() *HelloController {
	h := &HelloController{}
	h.SetBasePath("/hello")
	h.GET("/", h.hello)
	h.GET("/{name}", h.helloName)
	return h
}

func (h *HelloController) hello(w http.ResponseWriter, r *http.Request) {
	core.OK(w, map[string]string{"message": "Hello, World!"})
}

func (h *HelloController) helloName(w http.ResponseWriter, r *http.Request) {
	// Path parameters (e.g. {name}) are resolved by the router internally.
	// Use a helper or middleware to extract them from the request context.
	core.OK(w, map[string]string{"message": "Hello!"})
}

// ── Module ────────────────────────────────────────────────────────────────────

type HelloModule struct{}

func (m *HelloModule) Name() string                   { return "HelloModule" }
func (m *HelloModule) Controllers() []core.Controller { return []core.Controller{NewHelloController()} }

// ── Bootstrap ─────────────────────────────────────────────────────────────────

func main() {
	app := core.NewApp().
		SetGlobalPrefix("/api/v1").
		UseModule(&HelloModule{})

	log.Fatal(app.Start(":3000"))
}
