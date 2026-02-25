<p align="center">
  <img src="https://img.shields.io/badge/Zenqo-Framework-blue?style=for-the-badge&labelColor=000000" alt="Zenqo" />
</p>

<p align="center">A return-value-based <a href="https://go.dev" target="_blank">Go</a> web framework for building clean, scalable server-side applications.</p>

<p align="center">
  <a href="https://github.com/zenqos/zenqo/actions/workflows/ci.yml"><img src="https://github.com/zenqos/zenqo/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <a href="https://pkg.go.dev/github.com/zenqos/zenqo"><img src="https://pkg.go.dev/badge/github.com/zenqos/zenqo.svg" alt="Go Reference" /></a>
  <a href="https://goreportcard.com/report/github.com/zenqos/zenqo"><img src="https://goreportcard.com/badge/github.com/zenqos/zenqo" alt="Go Report Card" /></a>
  <a href="https://github.com/zenqos/zenqo/blob/main/LICENSE"><img src="https://img.shields.io/github/license/zenqos/zenqo" alt="License" /></a>
</p>

## Description

Zenqo is a framework for building efficient, maintainable Go web applications. Handlers return `(data, error)` instead of manually writing to `http.ResponseWriter` — the framework handles JSON serialization, status codes, and error responses automatically.

Under the hood, Zenqo uses <a href="https://github.com/go-chi/chi" target="_blank">chi</a> as its router, keeping Go's performance while removing the boilerplate that slows you down.

## Philosophy

Go's standard library gives you full control over HTTP, but that control comes with repetition. Every handler manually sets headers, encodes JSON, picks status codes, and writes error responses. The logic that matters — your business code — gets buried.

Zenqo takes a different approach. Handlers are pure functions that return values. The framework decides how to respond. This means less boilerplate, consistent API responses, and a clear separation between what your code does and how it's delivered.

```go
// Standard Go — you manage everything
func getUser(w http.ResponseWriter, r *http.Request) {
    user, err := svc.FindByID(id)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(404)
        json.NewEncoder(w).Encode(map[string]any{"code": 404, "message": "not found"})
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(200)
    json.NewEncoder(w).Encode(map[string]any{"success": true, "data": user})
}

// Zenqo — you focus on logic
func getUser(r *http.Request) (any, error) {
    user, err := svc.FindByID(id)
    if err != nil {
        return nil, core.ErrNotFound("user not found")
    }
    return user, nil
}
```

## Getting Started

### Install the CLI

```bash
# with Go
go install github.com/zenqos/zenqo/cmd/zenqo@latest

# or Homebrew
brew install zenqos/tap/zenqo
```

### Scaffold a New Project

```bash
zenqo new my-app
cd my-app && go run .
```

### Hello World

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

    log.Fatal(app.Start(":3000"))
}
```

```bash
curl http://localhost:3000/
# { "success": true, "data": { "message": "Hello, Zenqo!" } }
```

## Features

- **Return-value handlers** — `(any, error)` signature, automatic JSON & status codes
- **Controllers** — group routes under a base path with `BaseController`
- **Bind & Validate** — `Bind[T]` decodes JSON and runs struct validation in one call
- **Guards & Interceptors** — access control and lifecycle hooks at route, controller, or global level
- **Error handling** — panic recovery, typed errors, customizable global error handler
- **Auto camelCase** — struct fields serialize as camelCase without `json` tags
- **Built-in middleware** — CORS, secure headers, request ID, panic recovery

## Examples

- [`examples/basic`](./examples/basic) — Direct routing without controllers
- [`examples/crud`](./examples/crud) — Full CRUD API with Controller + Service pattern
- [`examples/auth`](./examples/auth) — JWT authentication with Guards, Interceptors, and Bind+Validation

## Documentation

- [CHANGELOG](./CHANGELOG.md) — detailed feature breakdown and version history
- [pkg.go.dev](https://pkg.go.dev/github.com/zenqos/zenqo) — API reference (GoDoc)

## Contributing

We welcome contributions! Please read the [Contributing Guide](CONTRIBUTING.md) before submitting a Pull Request.

## Security

If you discover a security vulnerability, **do not open a public issue**. Please follow the instructions in our [Security Policy](SECURITY.md).

## Stay in Touch

- Author — [@ftery0](https://github.com/ftery0)

## License

Zenqo is [MIT licensed](LICENSE).
