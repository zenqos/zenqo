package core

import (
	"errors"
	"testing"
)

func TestValidateRequired(t *testing.T) {
	type dto struct {
		Name string `validate:"required"`
	}
	err := validate(dto{})
	assertValidationField(t, err, "name", "name is required")

	err = validate(dto{Name: "Alice"})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateMinMax(t *testing.T) {
	type dto struct {
		Name string `validate:"min=2,max=5"`
	}
	assertValidationField(t, validate(dto{Name: "A"}), "name", "name must be at least 2 characters")
	assertValidationField(t, validate(dto{Name: "TooLong"}), "name", "name must be at most 5 characters")

	if err := validate(dto{Name: "OK"}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateEmail(t *testing.T) {
	type dto struct {
		Email string `validate:"email"`
	}
	// Empty string passes (use required separately)
	if err := validate(dto{}); err != nil {
		t.Fatalf("empty email should pass, got %v", err)
	}
	assertValidationField(t, validate(dto{Email: "bad"}), "email", "email must be a valid email address")

	if err := validate(dto{Email: "a@b.com"}); err != nil {
		t.Fatalf("valid email should pass, got %v", err)
	}
}

func TestValidateOneOf(t *testing.T) {
	type dto struct {
		Role string `validate:"oneof=admin|user"`
	}
	assertValidationField(t, validate(dto{Role: "guest"}), "role", "role must be one of: admin, user")

	if err := validate(dto{Role: "admin"}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidatePointerNil(t *testing.T) {
	type dto struct {
		Name *string `validate:"required"`
	}
	// nil + required = fail
	assertValidationField(t, validate(dto{}), "name", "name is required")
}

func TestValidatePointerNilSkip(t *testing.T) {
	type dto struct {
		Name *string `validate:"max=50"`
	}
	// nil + no required = skip
	if err := validate(dto{}); err != nil {
		t.Fatalf("nil pointer without required should skip, got %v", err)
	}
}

func TestValidatePointerValue(t *testing.T) {
	type dto struct {
		Name *string `validate:"min=2"`
	}
	s := "A"
	assertValidationField(t, validate(dto{Name: &s}), "name", "name must be at least 2 characters")

	s2 := "OK"
	if err := validate(dto{Name: &s2}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateNestedStruct(t *testing.T) {
	type Address struct {
		City string `validate:"required"`
	}
	type dto struct {
		Name    string  `validate:"required"`
		Address Address
	}
	err := validate(dto{Name: "Alice"})
	// City is required but empty → should fail
	assertValidationField(t, err, "city", "city is required")
}

func TestValidateNestedStructWithTag(t *testing.T) {
	type Address struct {
		City string `validate:"required"`
	}
	type dto struct {
		Address Address `validate:"required"`
	}
	err := validate(dto{})
	// Both "city is required" (from recursion) and "address is required" (from tag)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if len(ve.Errors) < 1 {
		t.Fatalf("expected at least 1 error, got %d", len(ve.Errors))
	}
}

func TestValidateMultipleErrors(t *testing.T) {
	type dto struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
	}
	var ve *ValidationError
	err := validate(dto{})
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if len(ve.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d: %v", len(ve.Errors), ve.Errors)
	}
}

func TestValidateNonStruct(t *testing.T) {
	if err := validate("hello"); err != nil {
		t.Fatalf("non-struct should return nil, got %v", err)
	}
	if err := validate(42); err != nil {
		t.Fatalf("non-struct should return nil, got %v", err)
	}
}

func TestValidateIntMin(t *testing.T) {
	type dto struct {
		Age int `validate:"min=18"`
	}
	assertValidationField(t, validate(dto{Age: 10}), "age", "age must be at least 18")

	if err := validate(dto{Age: 20}); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// assertValidationField checks that err is a *ValidationError containing
// a FieldError with the given field and message.
func assertValidationField(t *testing.T, err error, field, msg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error for field %q, got nil", field)
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	for _, fe := range ve.Errors {
		if fe.Field == field && fe.Message == msg {
			return
		}
	}
	t.Fatalf("expected field error {%q: %q}, got %v", field, msg, ve.Errors)
}
