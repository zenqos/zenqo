# Getting Started

> 한국어: [시작하기](./getting-started.ko.md)

## Install

```bash
# Go
go install github.com/zenqos/zenqo/cmd/zenqo@latest

# Homebrew
brew install zenqos/tap/zenqo
```

## Scaffold a New Project

```bash
zenqo new my-app
cd my-app && go run .
```

This generates a ready-to-run project with the recommended directory layout:

```
my-app/
├── cmd/my-app/main.go         # entry point
├── internal/
│   ├── app/app.go             # register controllers
│   └── example/               # generated feature module
│       ├── controller.go
│       ├── service.go
│       └── dto.go
└── go.mod
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

    log.Fatal(app.Start(":3000"))
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

## Next Steps

- [Controllers](./controllers.md) — group routes under a base path
- [Request Binding](./request-binding.md) — decode body, params, headers, files
- [Error Handling](./error-handling.md) — typed errors and custom handlers
- [Guards & Interceptors](./guards-interceptors.md) — access control and lifecycle hooks
- [Middleware](./middleware.md) — built-in and custom middleware
- [OpenAPI](./openapi.md) — auto-generated spec and Swagger UI
