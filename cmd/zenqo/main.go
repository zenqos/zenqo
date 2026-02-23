// Zenqo CLI — scaffold a new Zenqo project in one command.
//
// Install:
//
//	go install github.com/ftery0/zenqo/cmd/zenqo@latest
//
// Usage:
//
//	zenqo new my-app
//	zenqo new my-app --module github.com/myorg/my-app
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

type projectData struct {
	ModuleName  string // go module name (e.g. "my-app" or "github.com/user/my-app")
	ProjectName string // directory name only
	Port        string // default server port
}

const zenqoModule = "github.com/ftery0/zenqo"

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "new", "n":
		if err := runNew(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "\n❌  %v\n", err)
			os.Exit(1)
		}
default:
		printUsage()
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

  Zenqo CLI — NestJS-inspired Go framework

Usage:
  zenqo new <project-name> [flags]

Flags:
  --module   Go module name (default: project-name)
  --port     Default server port (default: 3000)

Examples:
  zenqo new my-app
  zenqo new my-app --module github.com/myorg/my-app

`)
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

// scaffold creates all project directories and files.
// Core files are not copied — they are downloaded via go get.
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
// Templates
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
	"github.com/ftery0/zenqo/core"
	"{{.ModuleName}}/internal/config"
)

// New wires all controllers together.
// cfg is accepted for future use — pass it to controllers that need DB connections, ports, etc.
// Add UseController calls here when you need new features.
func New(cfg config.Config) *core.App {
	return core.NewApp()
	// Example:
	// return core.NewApp().
	// 	UseController(user.NewController()).
	// 	UseController(product.NewController())
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

