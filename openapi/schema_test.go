package openapi

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func freshSB() *schemaBuilder {
	return &schemaBuilder{
		schemas:  make(map[string]*Schema),
		building: make(map[string]bool),
	}
}

// propNames returns the set of property names on a Schema, resolving $ref
// schemas from the builder's registry when necessary.
func propNames(t *testing.T, sb *schemaBuilder, s *Schema) map[string]bool {
	t.Helper()
	if s.Ref != "" {
		// Strip "#/components/schemas/"
		name := s.Ref[len("#/components/schemas/"):]
		resolved, ok := sb.schemas[name]
		if !ok {
			t.Fatalf("schema %q not found in registry", name)
		}
		s = resolved
	}
	out := make(map[string]bool)
	for k := range s.Properties {
		out[k] = true
	}
	return out
}

// ---------------------------------------------------------------------------
// Basic embedding — single level
// ---------------------------------------------------------------------------

func TestFromStruct_EmbeddedFlattened(t *testing.T) {
	type Timestamps struct {
		CreatedAt time.Time
		UpdatedAt time.Time
	}
	type User struct {
		ID         int64
		Name       string
		Timestamps // embedded ― should be inlined
	}

	sb := freshSB()
	schema := sb.fromValue(User{})
	props := propNames(t, sb, schema)

	for _, want := range []string{"id", "name", "createdAt", "updatedAt"} {
		if !props[want] {
			t.Errorf("expected property %q in User schema; got %v", want, props)
		}
	}
}

// ---------------------------------------------------------------------------
// Pointer embedding
// ---------------------------------------------------------------------------

func TestFromStruct_PointerEmbeddedFlattened(t *testing.T) {
	type Meta struct {
		Version int
		Author  string
	}
	type Document struct {
		Title string
		*Meta // pointer embedding — should also be inlined
	}

	sb := freshSB()
	schema := sb.fromValue(Document{})
	props := propNames(t, sb, schema)

	for _, want := range []string{"title", "version", "author"} {
		if !props[want] {
			t.Errorf("expected property %q in Document schema; got %v", want, props)
		}
	}
}

// ---------------------------------------------------------------------------
// Multi-level (nested) embedding
// ---------------------------------------------------------------------------

func TestFromStruct_MultiLevelEmbeddingFlattened(t *testing.T) {
	type Base struct {
		ID int64
	}
	type Audit struct {
		Base
		CreatedBy string
	}
	type Record struct {
		Audit
		Data string
	}

	sb := freshSB()
	schema := sb.fromValue(Record{})
	props := propNames(t, sb, schema)

	for _, want := range []string{"id", "createdBy", "data"} {
		if !props[want] {
			t.Errorf("expected property %q in Record schema; got %v", want, props)
		}
	}
}

// ---------------------------------------------------------------------------
// Multiple sibling embeddings
// ---------------------------------------------------------------------------

func TestFromStruct_MultipleEmbeddingsFlattened(t *testing.T) {
	type Timestamps struct {
		CreatedAt time.Time
		UpdatedAt time.Time
	}
	type SoftDelete struct {
		DeletedAt *time.Time
	}
	type Item struct {
		Name string
		Timestamps
		SoftDelete
	}

	sb := freshSB()
	schema := sb.fromValue(Item{})
	props := propNames(t, sb, schema)

	for _, want := range []string{"name", "createdAt", "updatedAt", "deletedAt"} {
		if !props[want] {
			t.Errorf("expected property %q in Item schema; got %v", want, props)
		}
	}
}

// ---------------------------------------------------------------------------
// Embedded interface — kmust be skipped gracefully
// ---------------------------------------------------------------------------

func TestFromStruct_EmbeddedInterfaceSkipped(t *testing.T) {
	type WithError struct {
		Name  string
		error // embedded interface — kmust not panic or crash
	}

	sb := freshSB()
	// Should not panic.
	schema := sb.fromValue(WithError{})
	props := propNames(t, sb, schema)

	if !props["name"] {
		t.Errorf("expected 'name' property, got %v", props)
	}

	// The embedded interface must NOT appear as a property.
	for k := range props {
		if k == "error" {
			t.Errorf("embedded interface 'error' must not appear as a property")
		}
	}
}

// ---------------------------------------------------------------------------
// Non-embedded named field (must NOT be flattened)
// ---------------------------------------------------------------------------

func TestFromStruct_NamedNestedStructNotFlattened(t *testing.T) {
	type Address struct {
		City    string
		Country string
	}
	type Person struct {
		Name    string
		Address Address // named field, NOT embedded — must remain nested
	}

	sb := freshSB()
	schema := sb.fromValue(Person{})
	props := propNames(t, sb, schema)

	// "address" should be a property on Person, not city/country.
	if !props["address"] {
		t.Errorf("expected 'address' property on Person, got %v", props)
	}
	if props["city"] || props["country"] {
		t.Errorf("city/country must not be inlined for a named (non-anonymous) Address field")
	}
}

// ---------------------------------------------------------------------------
// Unexported regular fields — must be skipped
// ---------------------------------------------------------------------------

func TestFromStruct_UnexportedFieldsSkipped(t *testing.T) {
	type User struct {
		Name   string // exported — should appear
		secret string // unexported — must not appear
	}

	sb := freshSB()
	schema := sb.fromValue(User{})
	props := propNames(t, sb, schema)

	if !props["name"] {
		t.Errorf("expected 'name' property, got %v", props)
	}
	if props["secret"] {
		t.Errorf("unexported field 'secret' must not appear in schema")
	}
}

// ---------------------------------------------------------------------------
// validate tags on embedded fields must still be applied
// ---------------------------------------------------------------------------

func TestFromStruct_EmbeddedValidateTagsApplied(t *testing.T) {
	type Required struct {
		Email string `validate:"required,email"`
	}
	type Form struct {
		Name string
		Required
	}

	sb := freshSB()
	schema := sb.fromValue(Form{})
	resolved := schema
	if schema.Ref != "" {
		name := schema.Ref[len("#/components/schemas/"):]
		resolved = sb.schemas[name]
	}

	// "email" must be in the required list from the embedded struct's validate tag.
	found := false
	for _, r := range resolved.Required {
		if r == "email" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'email' in required list from embedded struct validate tag; required=%v", resolved.Required)
	}

	// The email format must also be applied.
	emailProp, ok := resolved.Properties["email"]
	if !ok {
		t.Fatal("expected 'email' property")
	}
	if emailProp.Format != "email" {
		t.Errorf("expected format=email on 'email' property, got %q", emailProp.Format)
	}
}
