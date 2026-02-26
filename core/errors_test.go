package core

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
)

func TestDefaultErrorHandler_ValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	err := &ValidationError{Errors: []FieldError{
		{Field: "name", Message: "name is required"},
	}}
	DefaultErrorHandler(w, r, err)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["message"] != "validation failed" {
		t.Fatalf("expected 'validation failed', got %v", body["message"])
	}
	errs, ok := body["errors"].([]interface{})
	if !ok || len(errs) != 1 {
		t.Fatalf("expected 1 field error, got %v", body["errors"])
	}
}

func TestDefaultErrorHandler_HTTPError(t *testing.T) {
	tests := []struct {
		status int
		msg    string
	}{
		{400, "bad request"},
		{401, "unauthorized"},
		{403, "forbidden"},
		{404, "not found"},
		{500, "internal"},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		DefaultErrorHandler(w, r, &HTTPError{Status: tt.status, Message: tt.msg})

		if w.Code != tt.status {
			t.Fatalf("expected %d, got %d", tt.status, w.Code)
		}
		var body map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if body["message"] != tt.msg {
			t.Fatalf("expected %q, got %v", tt.msg, body["message"])
		}
	}
}

func TestDefaultErrorHandler_GenericError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	DefaultErrorHandler(w, r, errors.New("unexpected"))

	if w.Code != 500 {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["message"] != "internal server error" {
		t.Fatalf("expected 'internal server error', got %v", body["message"])
	}
}

func TestHTTPErrorHelpers(t *testing.T) {
	tests := []struct {
		fn     func(string) error
		status int
	}{
		{ErrBadRequest, 400},
		{ErrUnauthorized, 401},
		{ErrForbidden, 403},
		{ErrNotFound, 404},
		{ErrInternal, 500},
	}
	for _, tt := range tests {
		err := tt.fn("test")
		he, ok := err.(*HTTPError)
		if !ok {
			t.Fatalf("expected *HTTPError, got %T", err)
		}
		if he.Status != tt.status {
			t.Fatalf("expected %d, got %d", tt.status, he.Status)
		}
		if he.Error() != "test" {
			t.Fatalf("expected 'test', got %q", he.Error())
		}
	}
}

func TestValidationErrorMessage(t *testing.T) {
	err := &ValidationError{Errors: []FieldError{{Field: "x", Message: "y"}}}
	if err.Error() != "validation failed" {
		t.Fatalf("expected 'validation failed', got %q", err.Error())
	}
}

func TestErrValidation(t *testing.T) {
	err := ErrValidation(FieldError{Field: "a", Message: "b"}, FieldError{Field: "c", Message: "d"})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if len(ve.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(ve.Errors))
	}
}
