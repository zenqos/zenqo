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

// ────────────────────────────────────────────────────────────────────────────
// BindQuery edge cases
// ────────────────────────────────────────────────────────────────────────────

func TestBindQueryPresent(t *testing.T) {
		r, _ := http.NewRequest("GET", "/?page=3", nil)
		v := BindQuery(r, "page")
		if v != "3" {
					t.Fatalf("expected \"3\", got %q", v)
				}
}

func TestBindQueryMissing(t *testing.T) {
		r, _ := http.NewRequest("GET", "/", nil)
		v := BindQuery(r, "page")
		if v != "" {
					t.Fatalf("expected empty string for missing key, got %q", v)
				}
}

func TestBindQueryEmpty(t *testing.T) {
		r, _ := http.NewRequest("GET", "/?page=", nil)
		v := BindQuery(r, "page")
		if v != "" {
					t.Fatalf("expected empty string for empty value, got %q", v)
				}
}

func TestBindQueryMultipleValues(t *testing.T) {
		// net/url.Values.Get returns the first value when a key appears multiple times.
		r, _ := http.NewRequest("GET", "/?tag=go&tag=http", nil)
		v := BindQuery(r, "tag")
		if v != "go" {
					t.Fatalf("expected first value \"go\", got %q", v)
				}
}

func TestBindQuerySpecialChars(t *testing.T) {
		r, _ := http.NewRequest("GET", "/?q=hello+world&filter=a%26b", nil)
		q := BindQuery(r, "q")
		if q != "hello world" {
					t.Fatalf("expected \"hello world\", got %q", q)
				}
		f := BindQuery(r, "filter")
		if f != "a&b" {
					t.Fatalf("expected \"a&b\", got %q", f)
				}
}

// ────────────────────────────────────────────────────────────────────────────
// BindHeader edge cases
// ────────────────────────────────────────────────────────────────────────────

func TestBindHeaderPresent(t *testing.T) {
		r, _ := http.NewRequest("GET", "/", nil)
		r.Header.Set("Authorization", "Bearer token123")
		v := BindHeader(r, "Authorization")
		if v != "Bearer token123" {
					t.Fatalf("expected \"Bearer token123\", got %q", v)
				}
}

func TestBindHeaderMissing(t *testing.T) {
		r, _ := http.NewRequest("GET", "/", nil)
		v := BindHeader(r, "Authorization")
		if v != "" {
					t.Fatalf("expected empty string for missing header, got %q", v)
				}
}

func TestBindHeaderEmpty(t *testing.T) {
		r, _ := http.NewRequest("GET", "/", nil)
		r.Header.Set("X-Custom", "")
		v := BindHeader(r, "X-Custom")
		if v != "" {
					t.Fatalf("expected empty string for empty header value, got %q", v)
				}
}

func TestBindHeaderCaseInsensitive(t *testing.T) {
		// Go's http.Header.Get canonicalises the key, making lookup case-insensitive.
		r, _ := http.NewRequest("GET", "/", nil)
		r.Header.Set("content-type", "application/json")
		v := BindHeader(r, "Content-Type")
		if v != "application/json" {
					t.Fatalf("expected \"application/json\", got %q", v)
				}
}
