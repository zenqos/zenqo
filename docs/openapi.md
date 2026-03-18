# OpenAPI

> 한국어: [OpenAPI](./openapi.ko.md)

Zenqo generates an OpenAPI 3.1 spec and serves Swagger UI automatically — no manual YAML needed.

## Setup

Call `openapi.Mount` after all controllers are registered, before `app.Start`:

```go
import "github.com/zenqos/zenqo/openapi"

openapi.Mount(app, openapi.Config{
    Title:       "My API",
    Version:     "1.0.0",
    Description: "Optional description",
    SpecPath:    "/openapi.json", // default
    YAMLPath:    "/openapi.yaml", // default (set to "-" to disable)
    DocsPath:    "/docs",         // default
    Security:    []openapi.SecurityDef{openapi.BearerAuth()}, // optional
})

app.Start(":3000")
// Swagger UI: http://localhost:3000/docs
// JSON spec:  http://localhost:3000/openapi.json
// YAML spec:  http://localhost:3000/openapi.yaml
```

## Route Annotations

Chain builder methods on `*RouteDefinition` to enrich the spec:

```go
c.GET("/users", c.list).
    Summary("List all users").
    Description("Returns a paginated list of users.").
    Tags("users").
    Response(200, []UserDTO{})

c.POST("/users", c.create).
    Summary("Create a user").
    Tags("users").
    Body(CreateUserDTO{}).
    Response(201, UserDTO{}).
    Response(400, nil).
    Response(409, nil)

c.DELETE("/users/{id}", c.remove).
    Summary("Delete a user").
    Tags("users").
    Deprecated()
```

| Builder | Description |
|---------|-------------|
| `.Summary(s)` | One-line operation summary |
| `.Description(s)` | Longer operation description |
| `.Tags(t...)` | Group under one or more tags |
| `.Body(v)` | Pass a zero-value struct to infer the request body schema |
| `.Response(status, v)` | Pass a zero-value struct or `nil` for empty bodies |
| `.Deprecated()` | Mark as deprecated |
| `.NoSecurity()` | Override global security (mark as public) |

## Schema Inference

Schemas are inferred automatically from Go struct types:

| Go type | OpenAPI type |
|---------|--------------|
| `string` | `string` |
| `int`, `int64` | `integer` |
| `float32`, `float64` | `number` |
| `bool` | `boolean` |
| `[]T` | `array` |
| `time.Time` | `string / date-time` |
| `[]byte` | `string / byte` |
| `map[K]V` | `object` with `additionalProperties` |
| named struct | `$ref` → `#/components/schemas/Name` |

Constraints from `validate` struct tags are also reflected:

| Tag | Schema constraint |
|-----|-------------------|
| `required` | Added to parent `required` array |
| `min=N` | `minLength` (string) or `minimum` (number) |
| `max=N` | `maxLength` (string) or `maximum` (number) |
| `email` | `format: email` |
| `url` | `format: uri` |
| `uuid` | `format: uuid` |
| `oneof=a\|b` | `enum: [a, b]` |

## Path Parameters

Path parameters are extracted automatically from route paths:

```
/users/{id}  →  parameter: { name: "id", in: "path", required: true }
```

## Security Schemes

Add global security schemes via `Config.Security`:

```go
openapi.Mount(app, openapi.Config{
    Title:   "My API",
    Version: "1.0.0",
    Security: []openapi.SecurityDef{
        openapi.BearerAuth(),              // Authorization: Bearer <token>
        openapi.APIKeyHeader("X-API-Key"), // custom header
    },
})
```

| Helper | Description |
|--------|-------------|
| `openapi.BearerAuth()` | Bearer JWT authentication |
| `openapi.APIKeyHeader(name)` | API key via custom header |
| `openapi.APIKeyCookie(name)` | API key via cookie |
| `openapi.OAuth2AuthCode(cfg)` | OAuth2 authorization code flow |

Mark individual routes as public with `.NoSecurity()`:

```go
c.GET("/public", c.publicHandler).NoSecurity()
```

## Config Options

| Field | Default | Description |
|-------|---------|-------------|
| `Title` | (required) | API name shown in Swagger UI |
| `Version` | `"1.0.0"` | API version string |
| `Description` | `""` | Markdown description |
| `SpecPath` | `"/openapi.json"` | JSON spec endpoint |
| `YAMLPath` | `"/openapi.yaml"` | YAML spec endpoint (set `"-"` to disable) |
| `DocsPath` | `"/docs"` | Swagger UI endpoint |
| `Security` | `nil` | Global security schemes |
| `AutoErrorResponses` | `true` | Auto-inject 400/404/422/500 responses |
| `TryItOutEnabled` | `false` | Pre-activate "Try it out" in Swagger UI |
| `UseRFC9457` | `false` | Use ProblemDetail schema for error responses |

## Example Spec Output

```json
{
  "openapi": "3.1.0",
  "info": { "title": "My API", "version": "1.0.0" },
  "paths": {
    "/users": {
      "get": {
        "summary": "List all users",
        "tags": ["users"],
        "responses": {
          "200": {
            "content": {
              "application/json": {
                "schema": { "type": "array", "items": { "$ref": "#/components/schemas/UserDTO" } }
              }
            }
          }
        }
      }
    }
  }
}
```
