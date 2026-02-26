package main

import (
  	"os"
  	"path/filepath"
  	"strings"
  	"testing"
  )

func TestRunNewMissingProjectName(t *testing.T) {
  	err := runNew([]string{})
  	if err == nil {
      		t.Fatal("expected error when project name is missing")
      	}
  	if !strings.Contains(err.Error(), "project name is required") {
      		t.Fatalf("unexpected error message: %v", err)
      	}
  }

func TestRunNewDirectoryAlreadyExists(t *testing.T) {
  	dir := t.TempDir()
  	existing := filepath.Join(dir, "myapp")
  	if err := os.Mkdir(existing, 0o755); err != nil {
      		t.Fatalf("failed to create test directory: %v", err)
      	}
  	err := runNew([]string{existing})
  	if err == nil {
      		t.Fatal("expected error when target directory already exists")
      	}
  	if !strings.Contains(err.Error(), "already exists") {
      		t.Fatalf("unexpected error message: %v", err)
      	}
  }

func TestScaffoldCreatesExpectedFiles(t *testing.T) {
  	dir := t.TempDir()
  	target := filepath.Join(dir, "testproject")
  	data := projectData{
      		ModuleName:  "testproject",
      		ProjectName: "testproject",
      		Port:        "3000",
      	}

  	if err := scaffold(target, data); err != nil {
      		t.Fatalf("scaffold returned unexpected error: %v", err)
      	}

  	want := []string{
      		"go.mod",
      		".gitignore",
      		"main.go",
      		"internal/config/config.go",
      		"internal/app/app.go",
      	}
  	for _, rel := range want {
      		full := filepath.Join(target, rel)
      		if _, err := os.Stat(full); os.IsNotExist(err) {
            			t.Errorf("scaffold did not create expected file: %s", rel)
            		}
      	}
  }

func TestScaffoldGoModContainsModuleName(t *testing.T) {
  	dir := t.TempDir()
  	target := filepath.Join(dir, "mypkg")
  	data := projectData{
      		ModuleName:  "github.com/user/mypkg",
      		ProjectName: "mypkg",
      		Port:        "3000",
      	}

  	if err := scaffold(target, data); err != nil {
      		t.Fatalf("scaffold failed: %v", err)
      	}

  	content, err := os.ReadFile(filepath.Join(target, "go.mod"))
  	if err != nil {
      		t.Fatalf("failed to read go.mod: %v", err)
      	}
  	if !strings.Contains(string(content), "github.com/user/mypkg") {
      		t.Errorf("go.mod does not contain module name; got: %s", content)
      	}
  }

func TestScaffoldMainGoContainsModuleImports(t *testing.T) {
  	dir := t.TempDir()
  	target := filepath.Join(dir, "myapi")
  	data := projectData{
      		ModuleName:  "myapi",
      		ProjectName: "myapi",
      		Port:        "8080",
      	}

  	if err := scaffold(target, data); err != nil {
      		t.Fatalf("scaffold failed: %v", err)
      	}

  	content, err := os.ReadFile(filepath.Join(target, "main.go"))
  	if err != nil {
      		t.Fatalf("failed to read main.go: %v", err)
      	}
  	for _, imp := range []string{"myapi/internal/app", "myapi/internal/config"} {
      		if !strings.Contains(string(content), imp) {
            			t.Errorf("main.go missing import %q; content: %s", imp, content)
            		}
      	}
  }

func TestScaffoldAppGoContainsZenqoImport(t *testing.T) {
  	dir := t.TempDir()
  	target := filepath.Join(dir, "svc")
  	data := projectData{
      		ModuleName:  "github.com/org/svc",
      		ProjectName: "svc",
      		Port:        "4000",
      	}

  	if err := scaffold(target, data); err != nil {
      		t.Fatalf("scaffold failed: %v", err)
      	}

  	content, err := os.ReadFile(filepath.Join(target, "internal/app/app.go"))
  	if err != nil {
      		t.Fatalf("failed to read app.go: %v", err)
      	}
  	if !strings.Contains(string(content), "github.com/zenqos/zenqo/core") {
      		t.Errorf("app.go missing zenqo core import; content: %s", content)
      	}
  }

func TestScaffoldTemplateRendersPort(t *testing.T) {
  	dir := t.TempDir()
  	target := filepath.Join(dir, "porttest")
  	data := projectData{
      		ModuleName:  "porttest",
      		ProjectName: "porttest",
      		Port:        "9090",
      	}

  	if err := scaffold(target, data); err != nil {
      		t.Fatalf("scaffold failed: %v", err)
      	}

  	content, err := os.ReadFile(filepath.Join(target, "internal/config/config.go"))
  	if err != nil {
      		t.Fatalf("failed to read config.go: %v", err)
      	}
  	if !strings.Contains(string(content), "PORT") {
      		t.Errorf("config.go missing PORT env var reference; content: %s", content)
      	}
  }
