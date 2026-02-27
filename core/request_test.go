package core

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

// newRequestWithParam creates a request with a chi URL param set.
func newRequestWithParam(key, value string) *http.Request {
	r, _ := http.NewRequest("GET", "/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestParamString(t *testing.T) {
	r := newRequestWithParam("name", "alice")
	v, err := Param[string](r, "name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "alice" {
		t.Fatalf("expected alice, got %q", v)
	}
}

func TestParamInt64(t *testing.T) {
	r := newRequestWithParam("id", "42")
	v, err := Param[int64](r, "id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

func TestParamInt(t *testing.T) {
	r := newRequestWithParam("id", "7")
	v, err := Param[int](r, "id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 7 {
		t.Fatalf("expected 7, got %d", v)
	}
}

func TestParamUint64(t *testing.T) {
	r := newRequestWithParam("id", "99")
	v, err := Param[uint64](r, "id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 99 {
		t.Fatalf("expected 99, got %d", v)
	}
}

func TestParamMissing(t *testing.T) {
	r := newRequestWithParam("other", "val")
	_, err := Param[int64](r, "id")
	if err == nil {
		t.Fatal("expected error for missing param")
	}
	he, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected *HTTPError, got %T", err)
	}
	if he.Status != 400 {
		t.Fatalf("expected 400, got %d", he.Status)
	}
}

func TestParamInvalidInt(t *testing.T) {
	r := newRequestWithParam("id", "abc")
	_, err := Param[int64](r, "id")
	if err == nil {
		t.Fatal("expected error for non-integer param")
	}
	he, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected *HTTPError, got %T", err)
	}
	if he.Status != 400 {
		t.Fatalf("expected 400, got %d", he.Status)
	}
}

func TestParamNegativeUint(t *testing.T) {
	r := newRequestWithParam("id", "-1")
	_, err := Param[uint64](r, "id")
	if err == nil {
		t.Fatal("expected error for negative uint param")
	}
}

// --- BindQuery edge-case tests ---

func TestBindQueryPresent(t *testing.T) {
	r, _ := http.NewRequest("GET", "/search?q=hello", nil)
	got := BindQuery(r, "q")
	if got != "hello" {
		t.Fatalf("expected %q, got %q", "hello", got)
	}
}

func TestBindQueryMissing(t *testing.T) {
	r, _ := http.NewRequest("GET", "/search", nil)
	got := BindQuery(r, "q")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestBindQueryEmptyValue(t *testing.T) {
	r, _ := http.NewRequest("GET", "/search?key=", nil)
	got := BindQuery(r, "key")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestBindQueryURLEncoded(t *testing.T) {
	r, _ := http.NewRequest("GET", "/search?q=hello+world", nil)
	got := BindQuery(r, "q")
	if got != "hello world" {
		t.Fatalf("expected %q, got %q", "hello world", got)
	}
}

func TestBindQueryMultipleValues(t *testing.T) {
	r, _ := http.NewRequest("GET", "/search?q=first&q=second", nil)
	got := BindQuery(r, "q")
	if got != "first" {
		t.Fatalf("expected %q, got %q", "first", got)
	}
}

func TestBindQuerySpecialChars(t *testing.T) {
	r, _ := http.NewRequest("GET", "/search?q=%ED%95%9C%EA%B8%80", nil)
	got := BindQuery(r, "q")
	if got != "한글" {
		t.Fatalf("expected %q, got %q", "한글", got)
	}
}

// --- BindHeader edge-case tests ---

func TestBindHeaderPresent(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer token123")
	got := BindHeader(r, "Authorization")
	if got != "Bearer token123" {
		t.Fatalf("expected %q, got %q", "Bearer token123", got)
	}
}

func TestBindHeaderMissing(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	got := BindHeader(r, "X-Missing")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestBindHeaderCaseInsensitive(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Content-Type", "application/json")
	got := BindHeader(r, "content-type")
	if got != "application/json" {
		t.Fatalf("expected %q, got %q", "application/json", got)
	}
}

func TestBindHeaderEmptyValue(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("X-Empty", "")
	got := BindHeader(r, "X-Empty")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}
