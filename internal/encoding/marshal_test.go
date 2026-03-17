package encoding_test

import (
	"encoding/json"
	"testing"

	enc "github.com/zenqos/zenqo/internal/encoding"
)

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"ID", "id"},
		{"Name", "name"},
		{"CreatedAt", "createdAt"},
		{"UserID", "userId"},
		{"HTMLContent", "htmlContent"},
		{"", ""},
		{"A", "a"},
		{"URL", "url"},
	}
	for _, tt := range tests {
		got := enc.ToCamelCase(tt.in)
		if got != tt.want {
			t.Errorf("ToCamelCase(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMarshalSimpleStruct(t *testing.T) {
	type User struct {
		ID    int
		Name  string
		Email string
	}
	b, err := enc.Marshal(User{ID: 1, Name: "Alice", Email: "a@b.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if m["id"] != float64(1) {
		t.Errorf("expected id=1, got %v", m["id"])
	}
	if m["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", m["name"])
	}
	if m["email"] != "a@b.com" {
		t.Errorf("expected email=a@b.com, got %v", m["email"])
	}
}

func TestMarshalOmitempty(t *testing.T) {
	type Data struct {
		Name  string `json:"name"`
		Value string `json:"value,omitempty"`
	}
	b, err := enc.Marshal(Data{Name: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if _, exists := m["value"]; exists {
		t.Error("omitempty field should not be present when zero")
	}
}

func TestMarshalExcludeTag(t *testing.T) {
	type Data struct {
		Public  string
		Private string `json:"-"`
	}
	b, err := enc.Marshal(Data{Public: "yes", Private: "no"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if _, exists := m["private"]; exists {
		t.Error("excluded field should not be present")
	}
}

func TestMarshalNilSlice(t *testing.T) {
	type Data struct {
		Items []string
	}
	b, err := enc.Marshal(Data{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// nil slice should encode as []
	if string(b) != `{"items":[]}` {
		t.Errorf("expected null slice to encode as [], got %s", string(b))
	}
}

func TestMarshalNil(t *testing.T) {
	b, err := enc.Marshal(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(b) != "null" {
		t.Errorf("expected null, got %s", string(b))
	}
}

func TestMarshalEmbeddedStruct(t *testing.T) {
	type Base struct {
		ID int
	}
	type User struct {
		Base
		Name string
	}
	b, err := enc.Marshal(User{Base: Base{ID: 1}, Name: "Alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// Embedded fields should be inlined
	if m["id"] != float64(1) {
		t.Errorf("expected inlined id=1, got %v", m["id"])
	}
	if m["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", m["name"])
	}
}

func TestMarshalZenqoTag(t *testing.T) {
	type Data struct {
		MyField string `zenqo:"custom_key"`
	}
	b, err := enc.Marshal(Data{MyField: "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if m["custom_key"] != "value" {
		t.Errorf("expected custom_key=value, got %v", m)
	}
}

// TestFieldCacheConsistency ensures repeated calls produce the same result.
func TestFieldCacheConsistency(t *testing.T) {
	type Data struct {
		Name  string
		Email string
	}
	b1, _ := enc.Marshal(Data{Name: "a", Email: "b"})
	b2, _ := enc.Marshal(Data{Name: "a", Email: "b"})
	if string(b1) != string(b2) {
		t.Errorf("cached marshal should produce identical output:\n  %s\n  %s", b1, b2)
	}
}

// TestMarshalMapWithStructValues verifies camelCase is applied to struct
// values inside maps (was previously falling through to json.Marshal).
func TestMarshalMapWithStructValues(t *testing.T) {
	type User struct {
		FirstName string
		LastName  string
	}
	data := map[string]User{
		"alice": {FirstName: "Alice", LastName: "Smith"},
	}
	b, err := enc.Marshal(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	alice := m["alice"]
	if alice["firstName"] != "Alice" {
		t.Errorf("expected firstName=Alice, got %v (keys: %v)", alice["firstName"], alice)
	}
	if alice["lastName"] != "Smith" {
		t.Errorf("expected lastName=Smith, got %v (keys: %v)", alice["lastName"], alice)
	}
	// Ensure PascalCase keys are NOT present
	if _, exists := alice["FirstName"]; exists {
		t.Error("PascalCase key FirstName should not exist in map struct values")
	}
}

// TestMarshalNilMap verifies nil map encodes as null.
func TestMarshalNilMap(t *testing.T) {
	type Data struct {
		Meta map[string]string
	}
	b, err := enc.Marshal(Data{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// nil map field should encode as null via json.Marshal fallback
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if m["meta"] != nil {
		t.Errorf("expected nil map to encode as null, got %v", m["meta"])
	}
}

// TestMarshalDeepEmbedded verifies multi-level embedded struct inlining.
// Previously, only one level of embedding was inlined.
func TestMarshalDeepEmbedded(t *testing.T) {
	type A struct {
		X int
	}
	type B struct {
		A
		Y int
	}
	type C struct {
		B
		Z int
	}
	b, err := enc.Marshal(C{B: B{A: A{X: 1}, Y: 2}, Z: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// All three levels should be inlined flat
	if m["x"] != float64(1) {
		t.Errorf("expected x=1 (inlined from A), got %v", m["x"])
	}
	if m["y"] != float64(2) {
		t.Errorf("expected y=2 (inlined from B), got %v", m["y"])
	}
	if m["z"] != float64(3) {
		t.Errorf("expected z=3, got %v", m["z"])
	}
	// Should NOT have nested objects
	if _, exists := m["a"]; exists {
		t.Error("A should be inlined, not a nested object")
	}
	if _, exists := m["b"]; exists {
		t.Error("B should be inlined, not a nested object")
	}
}

// TestMarshalTextMarshaler verifies encoding.TextMarshaler support.
type textIP struct {
	addr string
}

func (t textIP) MarshalText() ([]byte, error) {
	return []byte(t.addr), nil
}

func TestMarshalTextMarshaler(t *testing.T) {
	type Server struct {
		Host textIP
		Port int
	}
	b, err := enc.Marshal(Server{Host: textIP{addr: "192.168.1.1"}, Port: 8080})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if m["host"] != "192.168.1.1" {
		t.Errorf("expected host=192.168.1.1, got %v", m["host"])
	}
}

// TestMarshalMapSortedKeys verifies map keys are sorted for deterministic output.
func TestMarshalMapSortedKeys(t *testing.T) {
	data := map[string]int{"c": 3, "a": 1, "b": 2}
	b, err := enc.Marshal(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"a":1,"b":2,"c":3}`
	if string(b) != expected {
		t.Errorf("expected sorted keys %s, got %s", expected, string(b))
	}
}

// TestMarshalMapNested verifies nested maps with struct values.
func TestMarshalMapNested(t *testing.T) {
	type Config struct {
		MaxRetry int
	}
	data := map[string]map[string]Config{
		"prod": {"api": {MaxRetry: 3}},
	}
	b, err := enc.Marshal(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]map[string]map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if m["prod"]["api"]["maxRetry"] != float64(3) {
		t.Errorf("expected nested map struct to use camelCase, got %v", m["prod"]["api"])
	}
}
