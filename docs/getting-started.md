# Getting Started

> 한국어: [시작하기](./getting-started.ko.md)

## Install

### macOS

```bash
# Homebrew (recommended)
brew install zenqos/tap/zenqo

# Verify
zenqo --help
```

### Linux

```bash
# One-liner install script
curl -fsSL https://raw.githubusercontent.com/zenqos/zenqo/main/install.sh | sh

# Verify
zenqo --help
```

### Windows

Open **PowerShell as Administrator** and run:

```powershell
irm https://raw.githubusercontent.com/zenqos/zenqo/main/install.ps1 | iex
```

This downloads the binary to `%LOCALAPPDATA%\zenqo` and adds it to your PATH automatically.

```powershell
# Verify (open a new terminal)
zenqo --help
```

### via go install (all platforms)

Requires Go 1.21+. The binary is placed in `$GOPATH/bin`.

```bash
go install github.com/zenqos/zenqo/cmd/zenqo@latest

# Add to PATH if not already (add this line to ~/.zshrc or ~/.bashrc)
export PATH="$PATH:$(go env GOPATH)/bin"

# Verify
zenqo --help
```

---

## Scaffold a New Project

```bash
zenqo new my-app
cd my-app && zenqo dev
```

This generates a ready-to-run project with the recommended directory layout:

```
my-app/
├── main.go                    # entry point
├── internal/
│   ├── app/app.go             # register controllers
│   └── config/config.go       # environment config
├── go.mod
└── .gitignore
```

Add features with the generator:

```bash
zenqo generate resource user   # creates internal/user/ with handler, service, dto, test
```

## Hello World (no CLI)

```go
package main

import (
    "log"
    "net/http"
    "github.com/zenqos/zenqo/core"
)

func main() {
    app := core.NewApp()

    app.GET("/", func(r *http.Request) (any, error) {
        return map[string]string{"message": "Hello, Zenqo!"}, nil
    })

    if err := app.Start(":3000"); err != nil {
        log.Fatal(err)
    }
}
```

```bash
curl http://localhost:3000/
# { "success": true, "data": { "message": "Hello, Zenqo!" } }
```

## Global Prefix

```go
app.SetGlobalPrefix("/api")
// All routes are now served under /api
```

## Graceful Shutdown

Zenqo handles `SIGINT` / `SIGTERM` automatically. To adjust the wait time:

```go
app.SetShutdownTimeout(10 * time.Second) // default: 30s
```

## Generate Boilerplate

```bash
# Full resource: controller + service + dto + test
zenqo generate resource user

# Individual pieces
zenqo generate controller order
zenqo generate guard auth
zenqo generate interceptor logging
```

## Use as HTTP Handler (testing / embedding)

```go
handler := app.Handler() // builds routes, returns http.Handler
ts := httptest.NewServer(handler)
defer ts.Close()
```

## CLI Commands

```bash
zenqo new <name>                 # scaffold new project
zenqo dev                        # run development server
zenqo dev --watch                # run with hot reload (restart on .go file changes)
zenqo generate resource <name>   # controller + service + dto + test
zenqo generate controller <name> # controller only
zenqo generate guard <name>      # guard boilerplate
zenqo generate interceptor <name># interceptor boilerplate
```

## Next Steps

- [Controllers](./controllers.md) — group routes under a base path
- [Request Binding](./request-binding.md) — decode body, params, headers, files
- [Error Handling](./error-handling.md) — typed errors and custom handlers
- [Guards & Interceptors](./guards-interceptors.md) — access control and lifecycle hooks
- [Middleware](./middleware.md) — built-in and custom middleware
- [OpenAPI](./openapi.md) — auto-generated spec and Swagger UI
