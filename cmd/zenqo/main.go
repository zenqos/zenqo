// Zenqo CLI — scaffold a new Zenqo project or generate boilerplate in one command.
//
// Install:
//
//	go install github.com/zenqos/zenqo/cmd/zenqo@latest
//
// Usage:
//
//	zenqo new my-app
//	zenqo generate resource user
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const zenqoModule = "github.com/zenqos/zenqo"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "new", "n":
		err = runNew(os.Args[2:])
	case "generate", "g":
		err = runGenerate(os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌  %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`
  ____
 |__  | ___ _ __   __ _  ___
   / / / _ \ '_ \ / _` + "`" + ` |/ _ \
  / /_|  __/ | | | (_| | (_) |
 /____|\___| |_| |_|\__, |\___/
                    |___/

  Zenqo CLI — build Go APIs with less boilerplate

Commands:
  new        Scaffold a new Zenqo project
  generate   Generate boilerplate code (alias: g)

Run 'zenqo <command> --help' for command-specific usage.

Examples:
  zenqo new my-app
  zenqo generate resource user
  zenqo generate guard jwt
  zenqo generate interceptor logging

`)
}

// ─────────────────────────────────────────────────────────
// zenqo new
// ─────────────────────────────────────────────────────────

type projectData struct {
	ModuleName  string
	ProjectName string
	Port        string
}

func runNew(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("project name is required\n\n  Usage: zenqo new <project-name>")
	}

	projectDir := args[0]
	moduleName := projectDir
	port := "3000"

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--module":
			if i+1 < len(args) {
				moduleName = args[i+1]
				i++
			}
		case "--port":
			if i+1 < len(args) {
				port = args[i+1]
				i++
			}
		}
	}

	data := projectData{
		ModuleName:  moduleName,
		ProjectName: filepath.Base(projectDir),
		Port:        port,
	}

	if _, err := os.Stat(projectDir); err == nil {
		return fmt.Errorf("directory %q already exists", projectDir)
	}

	fmt.Printf("\n  ✨ Creating Zenqo project: %s\n", projectDir)
	fmt.Printf("     Module : %s\n", moduleName)
	fmt.Printf("     Port   : %s\n\n", port)

	if err := scaffold(projectDir, data); err != nil {
		os.RemoveAll(projectDir)
		return err
	}

	fmt.Printf("\n  📦 Running go get %s@latest...\n", zenqoModule)
	cmd := exec.Command("go", "get", zenqoModule+"@latest")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("\n  ⚠️  go get failed — run manually:\n     cd %s && go get %s@latest\n", projectDir, zenqoModule)
	}

	line := strings.Repeat("─", 46)
	fmt.Printf(`
  %s
  ✅  Project ready!
  %s

  Next steps:

    cd %s
    go run .

  Your API → http://localhost:%s/api/v1

  Try it:
    curl http://localhost:%s/api/v1/users

  %s

`, line, line, projectDir, port, port, line)

	return nil
}

func scaffold(dir string, data projectData) error {
	files := map[string]string{
		"go.mod":                    tmplGoMod,
		".gitignore":                tmplGitignore,
		"main.go":                   tmplMain,
		"internal/config/config.go": tmplConfig,
		"internal/app/app.go":       tmplApp,
	}

	for relPath, content := range files {
		fullPath := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", relPath, err)
		}
		tmpl, err := template.New(relPath).Parse(content)
		if err != nil {
			return fmt.Errorf("template parse %s: %w", relPath, err)
		}
		f, err := os.Create(fullPath)
		if err != nil {
			return fmt.Errorf("create %s: %w", relPath, err)
		}
		if err := tmpl.Execute(f, data); err != nil {
			f.Close()
			return fmt.Errorf("write %s: %w", relPath, err)
		}
		f.Close()
		fmt.Printf("  ✓  %s\n", relPath)
	}
	return nil
}

// ─────────────────────────────────────────────────────────
// zenqo generate
// ─────────────────────────────────────────────────────────

type generateData struct {
	Name       string // lowercase, e.g. "user"
	NameTitle  string // PascalCase, e.g. "User"
	NamePlural string // plural lowercase, e.g. "users"
	Module     string // go module name from go.mod
	Package    string // package name (same as Name)
}

func runGenerate(args []string) error {
	if len(args) < 2 {
		printGenerateUsage()
		return nil
	}

	kind := args[0]
	name := strings.ToLower(args[1])

	module, err := detectModule()
	if err != nil {
		module = "my-app" // fallback — user can fix imports manually
	}

	data := generateData{
		Name:       name,
		NameTitle:  strings.ToUpper(name[:1]) + name[1:],
		NamePlural: pluralize(name),
		Module:     module,
		Package:    name,
	}

	switch kind {
	case "resource", "r":
		return generateResource(data)
	case "controller", "c":
		return generateFiles(data, map[string]string{
			filepath.Join("internal", data.Package, "handler.go"): tmplGenHandler,
		})
	case "guard", "g":
		return generateFiles(data, map[string]string{
			filepath.Join("internal", data.Package, data.Name+"_guard.go"): tmplGenGuard,
		})
	case "interceptor", "i":
		return generateFiles(data, map[string]string{
			filepath.Join("internal", data.Package, data.Name+"_interceptor.go"): tmplGenInterceptor,
		})
	default:
		return fmt.Errorf("unknown generator %q\n\nAvailable: resource, controller, guard, interceptor", kind)
	}
}

func printGenerateUsage() {
	fmt.Print(`
  Usage: zenqo generate <generator> <name>

  Generators:
    resource     controller + service + dto + test  (alias: r)
    controller   controller only                    (alias: c)
    guard        guard boilerplate                  (alias: g)
    interceptor  interceptor boilerplate            (alias: i)

  Examples:
    zenqo generate resource user
    zenqo generate controller product
    zenqo generate guard jwt
    zenqo generate interceptor logging

`)
}

func generateResource(data generateData) error {
	fmt.Printf("\n  ✨ Generating resource: %s\n\n", data.NameTitle)
	dir := filepath.Join("internal", data.Package)
	return generateFiles(data, map[string]string{
		filepath.Join(dir, "handler.go"):      tmplGenHandler,
		filepath.Join(dir, "service.go"):      tmplGenService,
		filepath.Join(dir, "dto.go"):          tmplGenDTO,
		filepath.Join(dir, "handler_test.go"): tmplGenTest,
	})
}

// generateFiles writes a map of relPath → templateStr, applying data to each template.
// Aborts if any target file already exists.
func generateFiles(data generateData, files map[string]string) error {
	// Check for existing files first to avoid partial writes.
	for relPath := range files {
		if _, err := os.Stat(relPath); err == nil {
			return fmt.Errorf("file %q already exists — aborting to prevent overwrite", relPath)
		}
	}

	for relPath, tmplStr := range files {
		if err := os.MkdirAll(filepath.Dir(relPath), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", relPath, err)
		}
		tmpl, err := template.New(relPath).Parse(tmplStr)
		if err != nil {
			return fmt.Errorf("template parse %s: %w", relPath, err)
		}
		f, err := os.Create(relPath)
		if err != nil {
			return fmt.Errorf("create %s: %w", relPath, err)
		}
		if execErr := tmpl.Execute(f, data); execErr != nil {
			f.Close()
			return fmt.Errorf("write %s: %w", relPath, execErr)
		}
		f.Close()
		fmt.Printf("  ✓  %s\n", relPath)
	}

	fmt.Printf("\n  ✅  Done! Wire it up in internal/app/app.go:\n")
	fmt.Printf("       app.UseController(%s.NewController())\n\n", data.Package)
	return nil
}

// detectModule reads the module name from go.mod in the current directory.
func detectModule() (string, error) {
	b, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(line[7:]), nil
		}
	}
	return "", fmt.Errorf("module declaration not found in go.mod")
}

// pluralize returns a naive English plural of s.
func pluralize(s string) string {
	switch {
	case strings.HasSuffix(s, "s"), strings.HasSuffix(s, "x"), strings.HasSuffix(s, "z"),
		strings.HasSuffix(s, "sh"), strings.HasSuffix(s, "ch"):
		return s + "es"
	case strings.HasSuffix(s, "y") && len(s) > 1:
		vowels := "aeiou"
		prev := string(s[len(s)-2])
		if !strings.Contains(vowels, prev) {
			return s[:len(s)-1] + "ies"
		}
	}
	return s + "s"
}

// ─────────────────────────────────────────────────────────
// Templates — zenqo new
// ─────────────────────────────────────────────────────────

const tmplGoMod = `module {{.ModuleName}}

go 1.23
`

const tmplGitignore = `# Binaries
*.exe
*.out
/bin/

# Go
vendor/

# IDE
.idea/
.vscode/
*.swp
`

const tmplMain = `package main

import (
	"log"

	"{{.ModuleName}}/internal/app"
	"{{.ModuleName}}/internal/config"
)

func main() {
	cfg := config.Load()
	log.Fatal(app.New(cfg).Start(":" + cfg.Port))
}
`

const tmplApp = `package app

import (
	"github.com/zenqos/zenqo/core"
	"github.com/zenqos/zenqo/middleware"
	"{{.ModuleName}}/internal/config"
)

// New wires all controllers together.
// cfg is accepted for future use — pass it to controllers that need DB connections, ports, etc.
// Add UseController calls here when you need new features.
func New(cfg config.Config) *core.App {
	app := core.NewApp()
	app.Use(middleware.SecureHeaders())
	return app
	// Example:
	// app.UseController(user.NewController())
	// app.UseController(product.NewController())
}
`

const tmplConfig = `package config

import (
	"log"
	"os"
)

type Config struct {
	Port string
	Env  string
}

func Load() Config {
	return Config{
		Port: getEnv("PORT", "3000"),
		Env:  getEnv("APP_ENV", "development"),
	}
}

// getEnv returns the env value or a fallback default.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// mustGetEnv returns the env value or exits if not set.
// Use this for required secrets like DB credentials, API keys, etc.
func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("[config] required environment variable %q is not set", key)
	}
	return v
}
`

// ─────────────────────────────────────────────────────────
// Templates — zenqo generate
// ─────────────────────────────────────────────────────────

const tmplGenHandler = `package {{.Package}}

import (
	"net/http"

	"github.com/zenqos/zenqo/core"
)

// Controller handles HTTP requests for the /{{.NamePlural}} resource.
type Controller struct {
	core.BaseController
	svc *Service
}

// NewController creates the {{.Name}} controller and registers all routes.
func NewController() *Controller {
	c := &Controller{svc: NewService()}
	c.SetBasePath("/{{.NamePlural}}")

	c.GET("/", c.findAll).
		Summary("List all {{.NamePlural}}").
		Tags("{{.NamePlural}}").
		Response(200, []{{.NameTitle}}{})

	c.GET("/{id}", c.findOne).
		Summary("Get {{.Name}} by ID").
		Tags("{{.NamePlural}}").
		Response(200, {{.NameTitle}}{})

	c.POST("/", c.create).
		Summary("Create a new {{.Name}}").
		Tags("{{.NamePlural}}").
		Body(Create{{.NameTitle}}DTO{}).
		Response(201, {{.NameTitle}}{})

	c.PUT("/{id}", c.update).
		Summary("Update a {{.Name}}").
		Tags("{{.NamePlural}}").
		Body(Update{{.NameTitle}}DTO{}).
		Response(200, {{.NameTitle}}{})

	c.DELETE("/{id}", c.remove).
		Summary("Delete a {{.Name}}").
		Tags("{{.NamePlural}}").
		Response(204, nil)

	return c
}

func (c *Controller) findAll(r *http.Request) (any, error) {
	return c.svc.FindAll(), nil
}

func (c *Controller) findOne(r *http.Request) (any, error) {
	id, err := core.Param[int64](r, "id")
	if err != nil {
		return nil, err
	}
	item, err := c.svc.FindOne(id)
	if err != nil {
		return nil, core.ErrNotFound("{{.Name}} not found")
	}
	return item, nil
}

func (c *Controller) create(r *http.Request) (any, error) {
	dto, err := core.Bind[Create{{.NameTitle}}DTO](r)
	if err != nil {
		return nil, err
	}
	return c.svc.Create(dto), nil
}

func (c *Controller) update(r *http.Request) (any, error) {
	id, err := core.Param[int64](r, "id")
	if err != nil {
		return nil, err
	}
	dto, err := core.Bind[Update{{.NameTitle}}DTO](r)
	if err != nil {
		return nil, err
	}
	item, err := c.svc.Update(id, dto)
	if err != nil {
		return nil, core.ErrNotFound("{{.Name}} not found")
	}
	return item, nil
}

func (c *Controller) remove(r *http.Request) (any, error) {
	id, err := core.Param[int64](r, "id")
	if err != nil {
		return nil, err
	}
	if err := c.svc.Delete(id); err != nil {
		return nil, core.ErrNotFound("{{.Name}} not found")
	}
	return nil, nil
}
`

const tmplGenService = `package {{.Package}}

import (
	"fmt"
	"sync"
)

// Service manages {{.Name}} data.
// Replace the in-memory store with a real database repository in production.
type Service struct {
	mu      sync.RWMutex
	items   map[int64]*{{.NameTitle}}
	counter int64
}

func NewService() *Service {
	return &Service{items: make(map[int64]*{{.NameTitle}})}
}

func (s *Service) FindAll() []*{{.NameTitle}} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*{{.NameTitle}}, 0, len(s.items))
	for _, v := range s.items {
		list = append(list, v)
	}
	return list
}

func (s *Service) FindOne(id int64) (*{{.NameTitle}}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[id]
	if !ok {
		return nil, fmt.Errorf("{{.Name}} %d not found", id)
	}
	return v, nil
}

func (s *Service) Create(dto Create{{.NameTitle}}DTO) *{{.NameTitle}} {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	item := &{{.NameTitle}}{ID: s.counter}
	// TODO: populate item fields from dto
	s.items[item.ID] = item
	return item
}

func (s *Service) Update(id int64, dto Update{{.NameTitle}}DTO) (*{{.NameTitle}}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return nil, fmt.Errorf("{{.Name}} %d not found", id)
	}
	// TODO: apply dto fields to item
	_ = dto
	return item, nil
}

func (s *Service) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return fmt.Errorf("{{.Name}} %d not found", id)
	}
	delete(s.items, id)
	return nil
}
`

const tmplGenDTO = `package {{.Package}}

// {{.NameTitle}} is the API response entity.
type {{.NameTitle}} struct {
	ID int64
	// TODO: add your fields here
	// Name string
}

// Create{{.NameTitle}}DTO is the request body for POST /{{.NamePlural}}.
type Create{{.NameTitle}}DTO struct {
	// TODO: add your fields here with validate tags
	// Name string ` + "`" + `validate:"required,min=2,max=100"` + "`" + `
}

// Update{{.NameTitle}}DTO is the request body for PUT /{{.NamePlural}}/{id}.
// Use pointer fields to distinguish "not sent" from "set to empty".
type Update{{.NameTitle}}DTO struct {
	// TODO: add your fields here
	// Name *string ` + "`" + `validate:"max=100"` + "`" + `
}
`

const tmplGenTest = `package {{.Package}}_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zenqos/zenqo/core"
	"{{.Module}}/internal/{{.Package}}"
)

func setup() http.Handler {
	app := core.NewApp()
	app.SetGlobalPrefix("/api/v1")
	app.UseController({{.Package}}.NewController())
	return app.Handler()
}

func TestFindAll(t *testing.T) {
	h := setup()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/{{.NamePlural}}/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate(t *testing.T) {
	h := setup()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/{{.NamePlural}}/", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}
`

const tmplGenGuard = `package {{.Package}}

import (
	"net/http"

	"github.com/zenqos/zenqo/core"
)

// {{.NameTitle}}Guard controls access to protected routes.
// Register it with .UseGuard() on a route or .UseControllerGuard() on a controller.
//
// Example:
//
//	c.GET("/protected", c.handler).UseGuard(&{{.NameTitle}}Guard{})
type {{.NameTitle}}Guard struct{}

// CanActivate checks whether the incoming request is allowed to proceed.
// Return (true, nil) to allow, or (false, core.ErrUnauthorized("...")) to reject with 401.
func (g *{{.NameTitle}}Guard) CanActivate(r *http.Request) (bool, error) {
	// TODO: implement your authorization logic here.
	// Example: validate a Bearer token
	token := r.Header.Get("Authorization")
	if token == "" {
		return false, core.ErrUnauthorized("missing authorization token")
	}
	return true, nil
}
`

const tmplGenInterceptor = `package {{.Package}}

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/zenqos/zenqo/core"
)

type contextKey string

const startTimeKey contextKey = "startTime"

// {{.NameTitle}}Interceptor records timing around each request.
// Register it with .UseInterceptor() on a route or .UseControllerInterceptor() on a controller.
//
// Example:
//
//	c.GET("/users", c.findAll).UseInterceptor(&{{.NameTitle}}Interceptor{})
type {{.NameTitle}}Interceptor struct{}

// Before runs before the handler. Injects the start time into the context.
func (i *{{.NameTitle}}Interceptor) Before(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, startTimeKey, time.Now())
}

// After runs after the handler with the final HTTP status code.
func (i *{{.NameTitle}}Interceptor) After(ctx context.Context, w http.ResponseWriter, statusCode int) {
	if start, ok := ctx.Value(startTimeKey).(time.Time); ok {
		core.Zlog("{{.NameTitle}}", fmt.Sprintf("status=%d elapsed=%s", statusCode, time.Since(start)))
	}
}
`
