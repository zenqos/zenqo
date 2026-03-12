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
// Unit tests for the JSON → YAML converter
// ---------------------------------------------------------------------------

func TestJSONToYAML_SimpleObject(t *testing.T) {
  	input := `{"title":"My API","version":"1.0.0"}`
  	out, err := jsonToYAML([]byte(input))
  	if err != nil {
      		t.Fatalf("unexpected error: %v", err)
      	}
  	yaml := string(out)

  	assertContains(t, yaml, "title: My API")
  	assertContains(t, yaml, "version: 1.0.0")
  }

func TestJSONToYAML_NestedObject(t *testing.T) {
  	input := `{"info":{"title":"Test","version":"2.0"}}`
  	out, err := jsonToYAML([]byte(input))
  	if err != nil {
      		t.Fatalf("unexpected error: %v", err)
      	}
  	yaml := string(out)

  	assertContains(t, yaml, "info:")
  	assertContains(t, yaml, "  title: Test")
  	// "2.0" is a string in JSON; the converter must quote it so YAML parsers
  	// treat it as a string rather than a float scalar.
  	assertContains(t, yaml, `  version: "2.0"`)
  }

func TestJSONToYAML_Array(t *testing.T) {
  	input := `{"tags":["users","auth"]}`
  	out, err := jsonToYAML([]byte(input))
  	if err != nil {
      		t.Fatalf("unexpected error: %v", err)
      	}
  	yaml := string(out)

  	assertContains(t, yaml, "tags:")
  	assertContains(t, yaml, "- users")
  	assertContains(t, yaml, "- auth")
  }

func TestJSONToYAML_BooleanAndNull(t *testing.T) {
  	input := `{"deprecated":true,"nullable":false,"example":null}`
  	out, err := jsonToYAML([]byte(input))
  	if err != nil {
      		t.Fatalf("unexpected error: %v", err)
      	}
  	yaml := string(out)

  	assertContains(t, yaml, "deprecated: true")
  	assertContains(t, yaml, "nullable: false")
  	assertContains(t, yaml, "example: null")
  }

func TestJSONToYAML_SpecialCharsInValue(t *testing.T) {
  	input := `{"description":"Error: bad request","path":"/users/{id}"}`
  	out, err := jsonToYAML([]byte(input))
  	if err != nil {
      		t.Fatalf("unexpected error: %v", err)
      	}
  	yaml := string(out)
  	assertContains(t, yaml, `"Error: bad request"`)
  }

func TestJSONToYAML_RefKey(t *testing.T) {
  	input := `{"$ref":"#/components/schemas/User"}`
  	out, err := jsonToYAML([]byte(input))
  	if err != nil {
      		t.Fatalf("unexpected error: %v", err)
      	}
  	yaml := string(out)
  	assertContains(t, yaml, "$ref:")
  }

func TestJSONToYAML_StableKeyOrder(t *testing.T) {
  	input := `{"z":1,"a":2,"m":3}`
  	out, err := jsonToYAML([]byte(input))
  	if err != nil {
      		t.Fatalf("unexpected error: %v", err)
      	}
  	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
  	if len(lines) != 3 {
      		t.Fatalf("expected 3 lines, got %d: %s", len(lines), out)
      	}
  	if !strings.HasPrefix(lines[0], "a:") {
      		t.Errorf("first key should be 'a', got %q", lines[0])
      	}
  	if !strings.HasPrefix(lines[1], "m:") {
      		t.Errorf("second key should be 'm', got %q", lines[1])
      	}
  	if !strings.HasPrefix(lines[2], "z:") {
      		t.Errorf("third key should be 'z', got %q", lines[2])
      	}
  }

func TestJSONToYAML_InvalidInput(t *testing.T) {
  	_, err := jsonToYAML([]byte("not valid json"))
  	if err == nil {
      		t.Fatal("expected error for invalid JSON input")
      	}
  }

func TestJSONToYAML_RoundTrip(t *testing.T) {
  	original := map[string]any{
      		"openapi": "3.1.0",
      		"info": map[string]any{
            			"title":   "Round-trip API",
            			"version": "0.1.0",
            		},
      		"paths": map[string]any{},
      	}
  	jsonBytes, _ := json.Marshal(original)
  	yamlBytes, err := jsonToYAML(jsonBytes)
  	if err != nil {
      		t.Fatalf("unexpected error: %v", err)
      	}
  	if len(yamlBytes) == 0 {
      		t.Fatal("expected non-empty YAML output")
      	}
  	yaml := string(yamlBytes)
  	assertContains(t, yaml, "openapi: 3.1.0")
  	assertContains(t, yaml, "Round-trip API")
  }

// ---------------------------------------------------------------------------
// Integration tests for the /openapi.yaml HTTP endpoint
// ---------------------------------------------------------------------------

func TestMount_YAMLEndpoint(t *testing.T) {
  	app := core.NewApp()

  	c := &core.BaseController{}
  	c.SetBasePath("/users")
  	c.GET("/{id}", func(r *http.Request) (any, error) {
      		return map[string]string{"id": "1"}, nil
      	}).Response(200, struct{ ID string }{})
  	app.UseController(c)

  	Mount(app, Config{Title: "Test API", Version: "1.0.0"})

  	w := httptest.NewRecorder()
  	r := httptest.NewRequest("GET", "/openapi.yaml", nil)
  	app.Handler().ServeHTTP(w, r)

  	if w.Code != http.StatusOK {
      		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
      	}
  	ct := w.Header().Get("Content-Type")
  	if !strings.HasPrefix(ct, "application/yaml") {
      		t.Errorf("expected application/yaml Content-Type, got %q", ct)
      	}
  	body := w.Body.String()
  	assertContains(t, body, "openapi:")
  	assertContains(t, body, "3.1.0")
  	assertContains(t, body, "Test API")
  }

func TestMount_YAMLEndpoint_CustomPath(t *testing.T) {
  	app := core.NewApp()

  	Mount(app, Config{
      		Title:    "Test",
      		YAMLPath: "/spec.yaml",
      	})

  	w := httptest.NewRecorder()
  	r := httptest.NewRequest("GET", "/spec.yaml", nil)
  	app.Handler().ServeHTTP(w, r)
  	if w.Code != http.StatusOK {
      		t.Fatalf("expected 200 at /spec.yaml, got %d", w.Code)
      	}

  	w2 := httptest.NewRecorder()
  	r2 := httptest.NewRequest("GET", "/openapi.yaml", nil)
  	app.Handler().ServeHTTP(w2, r2)
  	if w2.Code == http.StatusOK {
      		t.Error("expected non-200 at /openapi.yaml when custom YAMLPath is set")
      	}
  }

func TestMount_YAMLEndpoint_DisabledWithDash(t *testing.T) {
  	app := core.NewApp()
  	Mount(app, Config{Title: "Test", YAMLPath: "-"})

  	w := httptest.NewRecorder()
  	r := httptest.NewRequest("GET", "/openapi.yaml", nil)
  	app.Handler().ServeHTTP(w, r)
  	if w.Code == http.StatusOK {
      		t.Error("expected non-200 when YAML endpoint is disabled with '-'")
      	}
  }

func TestMount_JSONEndpointUnaffected(t *testing.T) {
  	app := core.NewApp()
  	Mount(app, Config{Title: "Test API", Version: "1.0.0"})

  	w := httptest.NewRecorder()
  	r := httptest.NewRequest("GET", "/openapi.json", nil)
  	app.Handler().ServeHTTP(w, r)

  	if w.Code != http.StatusOK {
      		t.Fatalf("expected 200 at /openapi.json, got %d", w.Code)
      	}
  	ct := w.Header().Get("Content-Type")
  	if !strings.HasPrefix(ct, "application/json") {
      		t.Errorf("expected application/json, got %q", ct)
      	}
  }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func assertContains(t *testing.T, haystack, needle string) {
  	t.Helper()
  	if !strings.Contains(haystack, needle) {
      		t.Errorf("expected output to contain %q\ngot:\n%s", needle, haystack)
      	}
  }
