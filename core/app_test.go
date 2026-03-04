package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- Integration tests ---

func TestAppGETRoute(t *testing.T) {
	app := NewApp()
	app.GET("/hello", func(r *http.Request) (any, error) {
		return map[string]string{"msg": "hello"}, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/hello", nil)
	app.Handler().ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if body["success"] != true {
		t.Fatalf("expected success=true, got %v", body["success"])
	}
}

func TestAppPOSTRoute(t *testing.T) {
	app := NewApp()
	app.POST("/items", func(r *http.Request) (any, error) {
		return map[string]string{"id": "1"}, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/items", nil)
	app.Handler().ServeHTTP(w, r)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestAppGlobalPrefix(t *testing.T) {
	app := NewApp()
	app.SetGlobalPrefix("/api/v1")

	c := &BaseController{}
	c.SetBasePath("/users")
	c.GET("/", func(r *http.Request) (any, error) {
		return []string{"alice"}, nil
	})
	app.UseController(c)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/users/", nil)
	app.Handler().ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAppModuleRegistration(t *testing.T) {
	c := &BaseController{}
	c.SetBasePath("/items")
	c.GET("/", func(r *http.Request) (any, error) {
		return "items", nil
	})

	m := &testModule{name: "items", controllers: []Controller{c}}
	app := NewApp()
	app.UseModule(m)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/items/", nil)
	app.Handler().ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

type testModule struct {
	name        string
	controllers []Controller
}

func (m *testModule) Name() string {
	return m.name
}

func (m *testModule) Controllers() []Controller {
	return m.controllers
}

func TestAppGlobalGuard(t *testing.T) {
	app := NewApp()
	app.UseGlobalGuard(&testGuard{allow: false})
	app.GET("/hello", func(r *http.Request) (any, error) {
		return "hello", nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/hello", nil)
	app.Handler().ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestApp404JSON(t *testing.T) {
	app := NewApp()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/nonexistent", nil)
	app.Handler().ServeHTTP(w, r)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if body["message"] != "not found" {
		t.Fatalf("expected 'not found', got %v", body["message"])
	}
}

func TestApp405JSON(t *testing.T) {
	app := NewApp()
	app.GET("/only-get", func(r *http.Request) (any, error) {
		return "ok", nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/only-get", nil)
	app.Handler().ServeHTTP(w, r)

	if w.Code != 405 {
		t.Fatalf("expected 405, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}
}

// --- Bind integration tests ---

func TestBindValidJSON(t *testing.T) {
	type dto struct {
		Name string `validate:"required"`
	}
	body := `{"name":"Alice"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	v, err := Bind[dto](r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Name != "Alice" {
		t.Fatalf("expected Alice, got %q", v.Name)
	}
}

func TestBindWrongContentType(t *testing.T) {
	type dto struct {
		Name string
	}
	r := httptest.NewRequest("POST", "/", strings.NewReader("name=Alice"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err := Bind[dto](r)
	if err == nil {
		t.Fatal("expected error for wrong Content-Type")
	}
	he, ok := err.(*HTTPError)
	if !ok || he.Status != 400 {
		t.Fatalf("expected 400 HTTPError, got %v", err)
	}
}

func TestBindBodyTooLarge(t *testing.T) {
	type dto struct {
		Data string
	}
	old := MaxBodySize
	MaxBodySize = 10
	defer func() { MaxBodySize = old }()

	bigBody := `{"data":"` + strings.Repeat("x", 100) + `"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(bigBody))
	r.Header.Set("Content-Type", "application/json")

	_, err := Bind[dto](r)
	if err == nil {
		t.Fatal("expected error for oversized body")
	}
}

func TestBindValidationFailure(t *testing.T) {
	type dto struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
	}
	body := `{"name":"","email":"bad"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	_, err := Bind[dto](r)
	if err == nil {
		t.Fatal("expected validation error")
	}
	var ve *ValidationError
	if ok := isValidationError(err, &ve); !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if len(ve.Errors) == 0 {
		t.Fatal("expected at least one field error")
	}
}

func TestBindInvalidJSON(t *testing.T) {
	type dto struct {
		Name string
	}
	r := httptest.NewRequest("POST", "/", strings.NewReader("not json"))
	r.Header.Set("Content-Type", "application/json")

	_, err := Bind[dto](r)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	he, ok := err.(*HTTPError)
	if !ok || he.Status != 400 {
		t.Fatalf("expected 400 HTTPError, got %v", err)
	}
}

func isValidationError(err error, target **ValidationError) bool {
	ve, ok := err.(*ValidationError)
	if ok {
		*target = ve
	}
	return ok
}

// --- Custom error handler test ---

func TestAppCustomErrorHandler(t *testing.T) {
	app := NewApp()
	app.SetErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
		Error(w, 418, "i'm a teapot")
	})
	app.GET("/err", func(r *http.Request) (any, error) {
		return nil, ErrInternal("fail")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/err", nil)
	app.Handler().ServeHTTP(w, r)

	if w.Code != 418 {
		t.Fatalf("expected 418, got %d", w.Code)
	}
}

// --- DELETE returns 204 ---

func TestAppDELETEReturnsNoContent(t *testing.T) {
	app := NewApp()
	app.DELETE("/items/{id}", func(r *http.Request) (any, error) {
		return nil, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/items/1", nil)
	app.Handler().ServeHTTP(w, r)

	if w.Code != 204 {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

// --- PUT/PATCH routes ---

func TestAppPUTRoute(t *testing.T) {
	app := NewApp()
	app.PUT("/items/{id}", func(r *http.Request) (any, error) {
		return map[string]string{"updated": "true"}, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/items/1", nil)
	app.Handler().ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAppPATCHRoute(t *testing.T) {
	app := NewApp()
	app.PATCH("/items/{id}", func(r *http.Request) (any, error) {
		return map[string]string{"patched": "true"}, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/items/1", nil)
	app.Handler().ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ---- SetShutdownTimeout tests ----

func TestSetShutdownTimeoutDefault(t *testing.T) {
	app := NewApp()
	if app.shutdownTimeout != 30*time.Second {
		t.Fatalf("expected default shutdownTimeout 30s, got %v", app.shutdownTimeout)
	}
}

func TestSetShutdownTimeoutUpdates(t *testing.T) {
	app := NewApp()
	app.SetShutdownTimeout(60 * time.Second)
	if app.shutdownTimeout != 60*time.Second {
		t.Fatalf("expected shutdownTimeout 60s after SetShutdownTimeout, got %v", app.shutdownTimeout)
	}
}
