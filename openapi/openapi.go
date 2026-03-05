// Package openapi generates an OpenAPI 3.1 spec from Zenqo route definitions
// and serves it alongside a Swagger UI explorer.
//
// Usage:
//
//	app := core.NewApp()
//	app.UseController(userController)
//
//	openapi.Mount(app, openapi.Config{
//	    Title:   "My API",
//	    Version: "1.0.0",
//	})
//
//	app.Start(":3000")
//	// → GET /openapi.json  — machine-readable spec
//	// → GET /docs          — Swagger UI
package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zenqos/zenqo/core"
	zlog "github.com/zenqos/zenqo/internal/log"
)

// Config holds metadata for the generated OpenAPI spec.
type Config struct {
	// Title is the API name shown in Swagger UI (required).
	Title string
	// Version is the API version string (default: "1.0.0").
	Version string
	// Description is an optional Markdown description shown in Swagger UI.
	Description string
	// SpecPath is the URL path that serves the JSON spec (default: "/openapi.json").
	SpecPath string
	// DocsPath is the URL path that serves the Swagger UI (default: "/docs").
	DocsPath string
}

// Mount registers the OpenAPI JSON spec and Swagger UI endpoints on the app.
// Call this after registering all controllers and modules, but before app.Start().
//
// Endpoints registered:
//   - GET {SpecPath}  → application/json OpenAPI 3.1 spec
//   - GET {DocsPath}  → Swagger UI HTML
func Mount(app *core.App, cfg Config) {
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}
	if cfg.SpecPath == "" {
		cfg.SpecPath = "/openapi.json"
	}
	if cfg.DocsPath == "" {
		cfg.DocsPath = "/docs"
	}

	// Collect routes NOW — before adding the openAPI controller itself.
	routes := app.CollectRoutes()
	sb := &schemaBuilder{
		schemas:  make(map[string]*Schema),
		building: make(map[string]bool),
	}
	spec := buildSpec(sb, cfg, routes)
	if len(sb.schemas) > 0 {
		spec.Components = &Components{Schemas: sb.schemas}
	}
	specJSON, marshalErr := json.MarshalIndent(spec, "", "  ")
	if marshalErr != nil {
		zlog.Err("OpenAPI", fmt.Sprintf("failed to marshal spec: %v", marshalErr))
	}

	specPath := cfg.SpecPath
	docsPath := cfg.DocsPath
	title := cfg.Title

	// The Swagger UI must fetch the spec using the full URL path (including any
	// global prefix set via app.SetGlobalPrefix), so the browser request resolves
	// correctly regardless of which path the docs page is served from.
	fullSpecURL := app.Prefix() + specPath

	ctrl := &openAPICtrl{}
	ctrl.SetBasePath("/")
	ctrl.Handle("GET", specPath, func(w http.ResponseWriter, r *http.Request) {
		if marshalErr != nil {
			http.Error(w, `{"code":500,"message":"failed to generate OpenAPI spec"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write(specJSON) //nolint:errcheck
	})
	ctrl.Handle("GET", docsPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, swaggerUIHTML, title, fullSpecURL) //nolint:errcheck
	})
	app.UseController(ctrl)
}
