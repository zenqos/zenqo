<p align="right">
  <b>English</b> | <a href="README.ko.md">한국어</a>
</p>

<p align="center">
  <img src="./assets/logo-wordmark.svg" alt="Zenqo" width="260" />
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

Handlers are pure functions that return `(data, error)`. The framework handles JSON serialization, status codes, and error responses — so your code stays focused on business logic.

```go
// Zenqo
func getUser(r *http.Request) (any, error) {
    user, err := svc.FindByID(id)
    if err != nil {
        return nil, core.ErrNotFound("user not found")
    }
    return user, nil
}
```

## Getting Started

```bash
go install github.com/zenqos/zenqo/cmd/zenqo@latest
zenqo new my-app
cd my-app && go run .
```

→ Full guide: [Getting Started](./docs/getting-started.md)

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

| Topic | EN | KO |
|-------|----|----|
| Getting Started | [→](./docs/getting-started.md) | [→](./docs/getting-started.ko.md) |
| Controllers | [→](./docs/controllers.md) | [→](./docs/controllers.ko.md) |
| Request Binding | [→](./docs/request-binding.md) | [→](./docs/request-binding.ko.md) |
| Error Handling | [→](./docs/error-handling.md) | [→](./docs/error-handling.ko.md) |
| Guards & Interceptors | [→](./docs/guards-interceptors.md) | [→](./docs/guards-interceptors.ko.md) |
| Middleware | [→](./docs/middleware.md) | [→](./docs/middleware.ko.md) |
| OpenAPI | [→](./docs/openapi.md) | [→](./docs/openapi.ko.md) |

- [CHANGELOG](./CHANGELOG.md) — version history
- [pkg.go.dev](https://pkg.go.dev/github.com/zenqos/zenqo) — API reference (GoDoc)

## Contributors

<table>
  <tbody>
    <tr>
      <td align="center" valign="top" width="14.28%">
        <a href="https://github.com/ftery0">
          <img src="https://avatars.githubusercontent.com/u/127281057?v=4" width="80px;" alt="ftery0"/><br />
          <sub><b>ftery0</b></sub>
        </a><br />
        <a title="Code">💻</a>
      </td>
      <td align="center" valign="top" width="14.28%">
        <a href="https://github.com/RogueTex">
          <img src="https://avatars.githubusercontent.com/u/218665445?v=4" width="80px;" alt="Raghu :)"/><br />
          <sub><b>Raghu :)</b></sub>
        </a><br />
        <a title="Code">💻</a>
      </td>
      <td align="center" valign="top" width="14.28%">
        <a href="https://github.com/aicontentcreate2023-star">
          <img src="https://avatars.githubusercontent.com/u/259026231?v=4" width="80px;" alt="aicontentcreate2023-star"/><br />
          <sub><b>aicontentcreate2023-star</b></sub>
        </a><br />
        <a title="Code">💻</a>
      </td>
    </tr>
  </tbody>
</table>

## Contributing

We welcome contributions! Please read the [Contributing Guide](CONTRIBUTING.md) before submitting a Pull Request.

## Security

If you discover a security vulnerability, **do not open a public issue**. Please follow the instructions in our [Security Policy](SECURITY.md).

## Stay in Touch

- Author — [@ftery0](https://github.com/ftery0)

## License

Zenqo is [MIT licensed](LICENSE).
