package core

import "testing"

func TestCheckURL(t *testing.T) {
	type dto struct {
		URL string `validate:"url"`
	}
	// empty passes
	if err := validate(dto{}); err != nil {
		t.Fatalf("empty should pass, got %v", err)
	}
	// valid URLs
	for _, u := range []string{"http://example.com", "https://example.com/path", "https://example.com:8080/api"} {
		if err := validate(dto{URL: u}); err != nil {
			t.Fatalf("valid URL %q should pass, got %v", u, err)
		}
	}
	// invalid URLs
	for _, u := range []string{
		"example.com",       // no scheme
		"not-a-url",         // no scheme
		"://missing-scheme", // no scheme
		"http://",           // no host
		"https://",          // no host
		"ftp://example.com", // unsupported scheme
	} {
		assertValidationField(t, validate(dto{URL: u}), "url", "url must be a valid URL")
	}
}

func TestCheckUUID(t *testing.T) {
	type dto struct {
		ID string `validate:"uuid"`
	}
	if err := validate(dto{}); err != nil {
		t.Fatalf("empty should pass, got %v", err)
	}
	if err := validate(dto{ID: "550e8400-e29b-41d4-a716-446655440000"}); err != nil {
		t.Fatalf("valid UUID should pass, got %v", err)
	}
	assertValidationField(t, validate(dto{ID: "not-a-uuid"}), "id", "id must be a valid UUID")
	assertValidationField(t, validate(dto{ID: "550e8400e29b41d4a716446655440000"}), "id", "id must be a valid UUID")
}

func TestCheckAlpha(t *testing.T) {
	type dto struct {
		Name string `validate:"alpha"`
	}
	if err := validate(dto{}); err != nil {
		t.Fatalf("empty should pass, got %v", err)
	}
	if err := validate(dto{Name: "Hello"}); err != nil {
		t.Fatalf("alpha string should pass, got %v", err)
	}
	assertValidationField(t, validate(dto{Name: "Hello123"}), "name", "name must contain only letters")
	assertValidationField(t, validate(dto{Name: "Hello World"}), "name", "name must contain only letters")
}

func TestCheckAlphaNum(t *testing.T) {
	type dto struct {
		Code string `validate:"alphanum"`
	}
	if err := validate(dto{}); err != nil {
		t.Fatalf("empty should pass, got %v", err)
	}
	if err := validate(dto{Code: "abc123"}); err != nil {
		t.Fatalf("alphanum string should pass, got %v", err)
	}
	assertValidationField(t, validate(dto{Code: "abc-123"}), "code", "code must contain only letters and numbers")
}

func TestCheckNumeric(t *testing.T) {
	type dto struct {
		Code string `validate:"numeric"`
	}
	if err := validate(dto{}); err != nil {
		t.Fatalf("empty should pass, got %v", err)
	}
	if err := validate(dto{Code: "12345"}); err != nil {
		t.Fatalf("numeric string should pass, got %v", err)
	}
	assertValidationField(t, validate(dto{Code: "123abc"}), "code", "code must contain only numbers")
}

func TestCheckLen(t *testing.T) {
	type dto struct {
		Code string `validate:"len=5"`
	}
	if err := validate(dto{Code: "ABCDE"}); err != nil {
		t.Fatalf("exact length should pass, got %v", err)
	}
	assertValidationField(t, validate(dto{Code: "ABC"}), "code", "code must be exactly 5 characters")
	assertValidationField(t, validate(dto{Code: "ABCDEFG"}), "code", "code must be exactly 5 characters")
}

func TestCheckRegex(t *testing.T) {
	type dto struct {
		Code string `validate:"regex=^[a-z]+$"`
	}
	if err := validate(dto{}); err != nil {
		t.Fatalf("empty should pass, got %v", err)
	}
	if err := validate(dto{Code: "abc"}); err != nil {
		t.Fatalf("matching regex should pass, got %v", err)
	}
	assertValidationField(t, validate(dto{Code: "ABC"}), "code", "code must match pattern ^[a-z]+$")
}

func TestCheckContains(t *testing.T) {
	type dto struct {
		Email string `validate:"contains=@"`
	}
	if err := validate(dto{}); err != nil {
		t.Fatalf("empty should pass, got %v", err)
	}
	if err := validate(dto{Email: "user@example.com"}); err != nil {
		t.Fatalf("containing @ should pass, got %v", err)
	}
	assertValidationField(t, validate(dto{Email: "nope"}), "email", `email must contain "@"`)
}

func TestCheckStartsWith(t *testing.T) {
	type dto struct {
		URL string `validate:"startswith=http"`
	}
	if err := validate(dto{}); err != nil {
		t.Fatalf("empty should pass, got %v", err)
	}
	if err := validate(dto{URL: "https://example.com"}); err != nil {
		t.Fatalf("startswith http should pass, got %v", err)
	}
	assertValidationField(t, validate(dto{URL: "ftp://example.com"}), "url", `url must start with "http"`)
}

func TestCheckEndsWith(t *testing.T) {
	type dto struct {
		Domain string `validate:"endswith=.com"`
	}
	if err := validate(dto{}); err != nil {
		t.Fatalf("empty should pass, got %v", err)
	}
	if err := validate(dto{Domain: "example.com"}); err != nil {
		t.Fatalf("endswith .com should pass, got %v", err)
	}
	assertValidationField(t, validate(dto{Domain: "example.org"}), "domain", `domain must end with ".com"`)
}

func TestCheckLowercase(t *testing.T) {
	type dto struct {
		Name string `validate:"lowercase"`
	}
	if err := validate(dto{}); err != nil {
		t.Fatalf("empty should pass, got %v", err)
	}
	if err := validate(dto{Name: "hello"}); err != nil {
		t.Fatalf("lowercase should pass, got %v", err)
	}
	assertValidationField(t, validate(dto{Name: "Hello"}), "name", "name must be lowercase")
}

func TestCheckUppercase(t *testing.T) {
	type dto struct {
		Name string `validate:"uppercase"`
	}
	if err := validate(dto{}); err != nil {
		t.Fatalf("empty should pass, got %v", err)
	}
	if err := validate(dto{Name: "HELLO"}); err != nil {
		t.Fatalf("uppercase should pass, got %v", err)
	}
	assertValidationField(t, validate(dto{Name: "Hello"}), "name", "name must be uppercase")
}
