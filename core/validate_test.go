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
		Name    string `validate:"required"`
		Address Address
	}
	err := validate(dto{Name: "Alice"})
	// City is required but empty → should fail with qualified path
	assertValidationField(t, err, "address.city", "address.city is required")
}

func TestValidateNestedStructWithTag(t *testing.T) {
	type Address struct {
		City string `validate:"required"`
	}
	type dto struct {
		Address Address `validate:"required"`
	}
	err := validate(dto{})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	// Should have both "address.city is required" (from recursion) and "address is required" (from tag)
	if len(ve.Errors) < 1 {
		t.Fatalf("expected at least 1 error, got %d", len(ve.Errors))
	}
}

// TestValidateNestedAmbiguousFields verifies that same-named fields in
// different nested structs are disambiguated by their path prefix (#86).
func TestValidateNestedAmbiguousFields(t *testing.T) {
	type ShippingAddress struct {
		City string `validate:"required"`
	}
	type BillingAddress struct {
		City string `validate:"required"`
	}
	type dto struct {
		Shipping ShippingAddress
		Billing  BillingAddress
	}
	err := validate(dto{})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	assertValidationField(t, err, "shipping.city", "shipping.city is required")
	assertValidationField(t, err, "billing.city", "billing.city is required")
}

// TestValidateDeepNested verifies 3-level deep nesting produces correct paths.
func TestValidateDeepNested(t *testing.T) {
	type Street struct {
		Name string `validate:"required"`
	}
	type Address struct {
		Street Street
	}
	type dto struct {
		Home Address
	}
	err := validate(dto{})
	assertValidationField(t, err, "home.address.street.name", "home.address.street.name is required")
}

// TestValidateCycleDetection verifies self-referential structs don't stack overflow (#89).
func TestValidateCycleDetection(t *testing.T) {
	type Node struct {
		Value  string `validate:"required"`
		Parent *Node
	}
	a := &Node{Value: "a"}
	b := &Node{Value: "b", Parent: a}
	a.Parent = b // cycle: a → b → a

	// Should not panic or infinite loop
	err := validate(a)
	if err != nil {
		t.Fatalf("valid cyclic struct should pass, got %v", err)
	}
}

// TestValidateCycleDetectionWithError verifies errors are still reported in cyclic structs.
func TestValidateCycleDetectionWithError(t *testing.T) {
	type Node struct {
		Value  string `validate:"required"`
		Parent *Node
	}
	a := &Node{Value: ""}
	b := &Node{Value: "ok", Parent: a}
	a.Parent = b // cycle: a → b → a

	err := validate(a)
	assertValidationField(t, err, "value", "value is required")
}

// TestValidateDive verifies slice element validation with dive rule (#87).
func TestValidateDive(t *testing.T) {
	type Address struct {
		City string `validate:"required"`
	}
	type dto struct {
		Addresses []Address `validate:"min=1,dive"`
	}
	err := validate(dto{Addresses: []Address{{City: "Seoul"}, {}}})
	assertValidationField(t, err, "addresses[1].city", "addresses[1].city is required")

	// All valid
	if err := validate(dto{Addresses: []Address{{City: "Seoul"}, {City: "Busan"}}}); err != nil {
		t.Fatalf("all valid elements should pass, got %v", err)
	}
}

// TestValidateDiveEmpty verifies dive on empty slice still respects min.
func TestValidateDiveEmpty(t *testing.T) {
	type Item struct {
		Name string `validate:"required"`
	}
	type dto struct {
		Items []Item `validate:"min=1,dive"`
	}
	err := validate(dto{Items: []Item{}})
	assertValidationField(t, err, "items", "items must have at least 1 items")
}

// TestValidateDivePointerElements verifies dive works with pointer slice elements.
func TestValidateDivePointerElements(t *testing.T) {
	type Item struct {
		Name string `validate:"required"`
	}
	type dto struct {
		Items []*Item `validate:"dive"`
	}
	err := validate(dto{Items: []*Item{{Name: "ok"}, {Name: ""}, nil}})
	assertValidationField(t, err, "items[1].name", "items[1].name is required")
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
