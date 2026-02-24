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
