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
//	// → GET /openapi.yaml  — YAML spec
//	// → GET /docs          — Swagger UI
package openapi

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"sync"

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
	// YAMLPath is the URL path that serves the YAML spec (default: "/openapi.yaml").
	// Set to "-" to disable the YAML endpoint.
	YAMLPath string
	// DocsPath is the URL path that serves the Swagger UI (default: "/docs").
	DocsPath string
	// AutoErrorResponses controls whether standard error responses are
	// automatically added to every route in the generated spec (default: true).
	//
	// When enabled the following responses are injected per route:
	//   - 500 Internal Server Error — always added
	//   - 400 Bad Request          — routes with a request body (Body())
	//   - 404 Not Found            — routes with path parameters
	//   - 422 Unprocessable Entity — routes with a request body (validation errors)
	//
	// The response schema matches the error format active for the app:
	//   - ProblemDetail when UseRFC9457 middleware is in use
	//   - ErrorResponse otherwise
	//
	// Set to false to opt out and document error responses manually.
	AutoErrorResponses *bool
	// TryItOutEnabled controls whether the "Try it out" button is pre-activated
	// in Swagger UI for all endpoints (default: false — standard Swagger UI behavior).
	TryItOutEnabled *bool
	// UseRFC9457 controls the error schema used for auto-injected error
	// responses. When true, ProblemDetail is used instead of ErrorResponse.
	UseRFC9457 bool
}

// Mount registers the OpenAPI JSON spec and Swagger UI endpoints on the app.
// Call this after registering all controllers and modules, but before app.Start().
//
// The spec is generated lazily on the first request to SpecPath. This means
// routes registered after Mount() but before the first spec request are included.
//
// Endpoints registered:
//   - GET {SpecPath}  → application/json OpenAPI 3.1 spec
//   - GET {YAMLPath}  → application/yaml OpenAPI 3.1 spec (set YAMLPath to "-" to disable)
//   - GET {DocsPath}  → Swagger UI HTML
func Mount(app *core.App, cfg Config) {
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}
	if cfg.SpecPath == "" {
		cfg.SpecPath = "/openapi.json"
	}
	if cfg.YAMLPath == "" {
		cfg.YAMLPath = "/openapi.yaml"
	}
	if cfg.DocsPath == "" {
		cfg.DocsPath = "/docs"
	}

	specPath := cfg.SpecPath
	yamlPath := cfg.YAMLPath
	docsPath := cfg.DocsPath

	// The Swagger UI must fetch the spec using the full URL path (including any
	// global prefix set via app.SetGlobalPrefix), so the browser request resolves
	// correctly regardless of which path the docs page is served from.
	fullSpecURL := app.Prefix() + specPath

	// Pre-escape template values to prevent XSS via crafted Config.Title or SpecPath.
	// - Title is placed inside an HTML <title> tag → HTML-escape.
	// - fullSpecURL is placed inside a JS string literal → JSON-encode (includes quotes).
	escapedTitle := html.EscapeString(cfg.Title)
	urlJSON, _ := json.Marshal(fullSpecURL) // e.g. `"/api/openapi.json"`

	tryItOut := "false"
	if cfg.TryItOutEnabled != nil && *cfg.TryItOutEnabled {
		tryItOut = "true"
	}

	// Spec generation is deferred to the first request so that all controllers
	// registered after Mount() are included in the spec.
	var (
		once     sync.Once
		specJSON []byte
		specYAML []byte
		buildErr error
	)

	generateSpec := func() {
		routes := app.CollectRoutes()
		sb := &schemaBuilder{
			schemas:  make(map[string]*Schema),
			building: make(map[string]bool),
		}
		spec := buildSpec(sb, cfg, routes)
		if len(sb.schemas) > 0 {
			spec.Components = &Components{Schemas: sb.schemas}
		}
		specJSON, buildErr = json.MarshalIndent(spec, "", "  ")
		if buildErr != nil {
			zlog.Err("OpenAPI", fmt.Sprintf("failed to marshal spec: %v", buildErr))
			return
		}
		// Convert the JSON spec to YAML so both formats share one generation pass.
		var yamlErr error
		specYAML, yamlErr = jsonToYAML(specJSON)
		if yamlErr != nil {
			// YAML conversion failure is non-fatal; log and leave specYAML nil.
			zlog.Err("OpenAPI", fmt.Sprintf("failed to convert spec to YAML: %v", yamlErr))
		}
	}

	ctrl := &openAPICtrl{}
	ctrl.SetBasePath("/")
	ctrl.Handle("GET", specPath, func(w http.ResponseWriter, r *http.Request) {
		once.Do(generateSpec)
		if buildErr != nil {
			http.Error(w, `{"code":500,"message":"failed to generate OpenAPI spec"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write(specJSON) //nolint:errcheck
	})
	if yamlPath != "-" {
		ctrl.Handle("GET", yamlPath, func(w http.ResponseWriter, r *http.Request) {
			once.Do(generateSpec)
			if buildErr != nil {
				http.Error(w, `{"code":500,"message":"failed to generate OpenAPI spec"}`, http.StatusInternalServerError)
				return
			}
			if specYAML == nil {
				http.Error(w, `{"code":500,"message":"failed to convert spec to YAML"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/yaml")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusOK)
			w.Write(specYAML) //nolint:errcheck
		})
	}
	ctrl.Handle("GET", docsPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, swaggerUIHTML, escapedTitle, string(urlJSON), tryItOut) //nolint:errcheck
	})
	app.UseController(ctrl)
	app.SetDocsPath(app.Prefix() + docsPath)
}
