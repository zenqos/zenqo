package openapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zenqos/zenqo/core"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newSchemaBuilder returns a fresh schemaBuilder for test use.
func newSchemaBuilder() *schemaBuilder {
	return &schemaBuilder{
		schemas:  make(map[string]*Schema),
		building: make(map[string]bool),
	}
}

// specFromApp mounts openapi on app with the given config, fires a GET to
// SpecPath, and returns the decoded spec map.
func specFromApp(t *testing.T, app *core.App, cfg Config) map[string]any {
	t.Helper()
	Mount(app, cfg)
	w := httptest.NewRecorder()
	sp := cfg.SpecPath
	if sp == "" {
		sp = "/openapi.json"
	}
	r := httptest.NewRequest("GET", sp, nil)
	app.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("spec request returned %d: %s", w.Code, w.Body.String())
	}
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("json.Unmarshal spec: %v", err)
	}
	return m
}

// operationResponses extracts the "responses" map for the given path+method.
func operationResponses(t *testing.T, spec map[string]any, path, method string) map[string]any {
	t.Helper()
	paths, _ := spec["paths"].(map[string]any)
	item, _ := paths[path].(map[string]any)
	op, _ := item[strings.ToLower(method)].(map[string]any)
	responses, _ := op["responses"].(map[string]any)
	return responses
}

// ---------------------------------------------------------------------------
// autoErrorResponsesEnabled
// ---------------------------------------------------------------------------

func TestAutoErrorResponsesEnabled_DefaultNil(t *testing.T) {
	if !autoErrorResponsesEnabled(Config{}) {
		t.Error("expected auto error responses to be enabled by default (nil)")
	}
}

func TestAutoErrorResponsesEnabled_ExplicitTrue(t *testing.T) {
	b := true
	if !autoErrorResponsesEnabled(Config{AutoErrorResponses: &b}) {
		t.Error("expected auto error responses to be enabled when explicitly true")
	}
}

func TestAutoErrorResponsesEnabled_ExplicitFalse(t *testing.T) {
	b := false
	if autoErrorResponsesEnabled(Config{AutoErrorResponses: &b}) {
		t.Error("expected auto error responses to be disabled when explicitly false")
	}
}

// ---------------------------------------------------------------------------
// 500 always injected
// ---------------------------------------------------------------------------

func TestAutoErrorResponses_500AlwaysInjected(t *testing.T) {
	app := core.NewApp()
	c := &core.BaseController{}
	c.SetBasePath("/items")
	c.GET("/", func(r *http.Request) (any, error) { return nil, nil })
	app.UseController(c)

	spec := specFromApp(t, app, Config{Title: "T"})
	resp := operationResponses(t, spec, "/items", "GET")
	if _, ok := resp["500"]; !ok {
		t.Error("expected 500 to be auto-injected on GET /items")
	}
}

// ---------------------------------------------------------------------------
// 400 + 422 injected for routes with a request body
// ---------------------------------------------------------------------------

func TestAutoErrorResponses_400_422_WithBody(t *testing.T) {
	type CreateDTO struct{ Name string }

	app := core.NewApp()
	c := &core.BaseController{}
	c.SetBasePath("/items")
	c.POST("/", func(r *http.Request) (any, error) { return nil, nil }).
		Body(CreateDTO{})
	app.UseController(c)

	spec := specFromApp(t, app, Config{Title: "T"})
	resp := operationResponses(t, spec, "/items", "POST")

	if _, ok := resp["400"]; !ok {
		t.Error("expected 400 to be auto-injected on POST with body")
	}
	if _, ok := resp["422"]; !ok {
		t.Error("expected 422 to be auto-injected on POST with body")
	}
}

func TestAutoErrorResponses_No400_422_WithoutBody(t *testing.T) {
	app := core.NewApp()
	c := &core.BaseController{}
	c.SetBasePath("/items")
	c.GET("/", func(r *http.Request) (any, error) { return nil, nil })
	app.UseController(c)

	spec := specFromApp(t, app, Config{Title: "T"})
	resp := operationResponses(t, spec, "/items", "GET")

	if _, ok := resp["400"]; ok {
		t.Error("did not expect 400 on GET without body")
	}
	if _, ok := resp["422"]; ok {
		t.Error("did not expect 422 on GET without body")
	}
}

// ---------------------------------------------------------------------------
// 404 injected for routes with path parameters
// ---------------------------------------------------------------------------

func TestAutoErrorResponses_404_WithPathParam(t *testing.T) {
	app := core.NewApp()
	c := &core.BaseController{}
	c.SetBasePath("/users")
	c.GET("/{id}", func(r *http.Request) (any, error) { return nil, nil })
	app.UseController(c)

	spec := specFromApp(t, app, Config{Title: "T"})
	resp := operationResponses(t, spec, "/users/{id}", "GET")

	if _, ok := resp["404"]; !ok {
		t.Error("expected 404 to be auto-injected on GET /users/{id}")
	}
}

func TestAutoErrorResponses_No404_WithoutPathParam(t *testing.T) {
	app := core.NewApp()
	c := &core.BaseController{}
	c.SetBasePath("/users")
	c.GET("/", func(r *http.Request) (any, error) { return nil, nil })
	app.UseController(c)

	spec := specFromApp(t, app, Config{Title: "T"})
	resp := operationResponses(t, spec, "/users", "GET")

	if _, ok := resp["404"]; ok {
		t.Error("did not expect 404 on GET without path param")
	}
}

// ---------------------------------------------------------------------------
// Explicit responses are not overwritten
// ---------------------------------------------------------------------------

func TestAutoErrorResponses_DoesNotOverwriteExplicit(t *testing.T) {
	type MyErr struct{ Msg string }

	app := core.NewApp()
	c := &core.BaseController{}
	c.SetBasePath("/items")
	c.GET("/{id}", func(r *http.Request) (any, error) { return nil, nil }).
		Response(200, struct{ ID string }{}).
		Response(404, MyErr{})
	app.UseController(c)

	spec := specFromApp(t, app, Config{Title: "T"})
	resp := operationResponses(t, spec, "/items/{id}", "GET")

	// The explicitly declared 404 must survive; the auto one must not replace it.
	r404, ok := resp["404"].(map[string]any)
	if !ok {
		t.Fatal("expected 404 in responses")
	}
	// The explicit 404 declares a body (MyErr), so it should have "content".
	if _, hasContent := r404["content"]; !hasContent {
		t.Error("expected explicit 404 response to preserve its content schema")
	}
}

// ---------------------------------------------------------------------------
// Opt-out via AutoErrorResponses = false
// ---------------------------------------------------------------------------

func TestAutoErrorResponses_OptOut(t *testing.T) {
	disabled := false

	app := core.NewApp()
	c := &core.BaseController{}
	c.SetBasePath("/items")
	c.GET("/{id}", func(r *http.Request) (any, error) { return nil, nil })
	app.UseController(c)

	spec := specFromApp(t, app, Config{
		Title:              "T",
		AutoErrorResponses: &disabled,
	})
	resp := operationResponses(t, spec, "/items/{id}", "GET")

	if _, ok := resp["500"]; ok {
		t.Error("expected 500 to NOT be injected when AutoErrorResponses is disabled")
	}
	if _, ok := resp["404"]; ok {
		t.Error("expected 404 to NOT be injected when AutoErrorResponses is disabled")
	}
}

// ---------------------------------------------------------------------------
// RFC 9457 error schema
// ---------------------------------------------------------------------------

func TestAutoErrorResponses_RFC9457Schema(t *testing.T) {
	app := core.NewApp()
	c := &core.BaseController{}
	c.SetBasePath("/items")
	c.GET("/", func(r *http.Request) (any, error) { return nil, nil })
	app.UseController(c)

	spec := specFromApp(t, app, Config{Title: "T", UseRFC9457: true})

	// ProblemDetail should appear in components/schemas.
	components, _ := spec["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	if _, ok := schemas["ProblemDetail"]; !ok {
		t.Error("expected ProblemDetail schema in components/schemas when UseRFC9457=true")
	}
}

func TestAutoErrorResponses_DefaultErrorResponseSchema(t *testing.T) {
	app := core.NewApp()
	c := &core.BaseController{}
	c.SetBasePath("/items")
	c.GET("/", func(r *http.Request) (any, error) { return nil, nil })
	app.UseController(c)

	spec := specFromApp(t, app, Config{Title: "T"})

	// ErrorResponse should appear in components/schemas.
	components, _ := spec["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	if _, ok := schemas["ErrorResponse"]; !ok {
		t.Error("expected ErrorResponse schema in components/schemas with default config")
	}
}
