package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─────────────────────────────────────────────────────────
// pluralize
// ─────────────────────────────────────────────────────────

func TestPluralize(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"user", "users"},
		{"product", "products"},
		{"category", "categories"},
		{"status", "statuses"},
		{"box", "boxes"},
		{"buzz", "buzzes"},
		{"branch", "branches"},
		{"watch", "watches"},
		{"day", "days"},   // vowel before y → just add s
		{"key", "keys"},   // vowel before y → just add s
		{"city", "cities"}, // consonant before y → ies
	}
	for _, c := range cases {
		got := pluralize(c.in)
		if got != c.want {
			t.Errorf("pluralize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ─────────────────────────────────────────────────────────
// detectModule
// ─────────────────────────────────────────────────────────

func TestDetectModule(t *testing.T) {
	dir := t.TempDir()
	gomod := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(gomod, []byte("module github.com/example/myapp\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig) //nolint:errcheck

	mod, err := detectModule()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mod != "github.com/example/myapp" {
		t.Errorf("got %q, want %q", mod, "github.com/example/myapp")
	}
}

func TestDetectModule_Missing(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig) //nolint:errcheck

	if _, err := detectModule(); err == nil {
		t.Fatal("expected error when go.mod is missing")
	}
}

// ─────────────────────────────────────────────────────────
// scaffold (zenqo new)
// ─────────────────────────────────────────────────────────

func TestScaffold(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "myapp")

	data := projectData{
		ModuleName:  "github.com/example/myapp",
		ProjectName: "myapp",
		Port:        "3000",
	}

	if err := scaffold(target, data); err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	wantFiles := []string{
		"go.mod",
		".gitignore",
		"main.go",
		"internal/config/config.go",
		"internal/app/app.go",
	}
	for _, f := range wantFiles {
		path := filepath.Join(target, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q to exist", f)
		}
	}

	// go.mod must contain the module name.
	b, _ := os.ReadFile(filepath.Join(target, "go.mod"))
	if !strings.Contains(string(b), "github.com/example/myapp") {
		t.Errorf("go.mod missing module name")
	}
}

func TestScaffold_ExistingDir(t *testing.T) {
	dir := t.TempDir()
	// dir already exists — runNew should return an error
	data := projectData{ModuleName: "m", ProjectName: "m", Port: "3000"}
	// scaffold itself doesn't check existence; runNew does — just confirm scaffold doesn't panic
	_ = scaffold(dir, data)
	_ = data
}

// ─────────────────────────────────────────────────────────
// generateFiles (zenqo generate)
// ─────────────────────────────────────────────────────────

func TestGenerateResource(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig) //nolint:errcheck

	data := generateData{
		Name:       "order",
		NameTitle:  "Order",
		NamePlural: "orders",
		Module:     "github.com/example/myapp",
		Package:    "order",
	}

	if err := generateResource(data); err != nil {
		t.Fatalf("generateResource failed: %v", err)
	}

	for _, f := range []string{
		"internal/order/handler.go",
		"internal/order/service.go",
		"internal/order/dto.go",
		"internal/order/handler_test.go",
	} {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("expected %q to exist", f)
		}
	}

	// handler.go must reference the Order type.
	b, _ := os.ReadFile("internal/order/handler.go")
	if !strings.Contains(string(b), "Order") {
		t.Error("handler.go does not mention Order type")
	}
}

func TestGenerateFiles_RefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig) //nolint:errcheck

	data := generateData{
		Name: "user", NameTitle: "User", NamePlural: "users",
		Module: "m", Package: "user",
	}

	// First generation should succeed.
	if err := generateResource(data); err != nil {
		t.Fatalf("first generate failed: %v", err)
	}
	// Second generation must fail with "already exists".
	err := generateResource(data)
	if err == nil {
		t.Fatal("expected error on second generate, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGenerateGuard(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig) //nolint:errcheck

	data := generateData{
		Name: "jwt", NameTitle: "Jwt", NamePlural: "jwts",
		Module: "m", Package: "jwt",
	}

	if err := generateFiles(data, map[string]string{
		filepath.Join("internal", "jwt", "jwt_guard.go"): tmplGenGuard,
	}); err != nil {
		t.Fatalf("generateFiles failed: %v", err)
	}

	b, _ := os.ReadFile("internal/jwt/jwt_guard.go")
	if !strings.Contains(string(b), "JwtGuard") {
		t.Error("guard file does not contain JwtGuard")
	}
}

func TestGenerateInterceptor(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig) //nolint:errcheck

	data := generateData{
		Name: "logging", NameTitle: "Logging", NamePlural: "loggings",
		Module: "m", Package: "logging",
	}

	if err := generateFiles(data, map[string]string{
		filepath.Join("internal", "logging", "logging_interceptor.go"): tmplGenInterceptor,
	}); err != nil {
		t.Fatalf("generateFiles failed: %v", err)
	}

	b, _ := os.ReadFile("internal/logging/logging_interceptor.go")
	if !strings.Contains(string(b), "LoggingInterceptor") {
		t.Error("interceptor file does not contain LoggingInterceptor")
	}
}
