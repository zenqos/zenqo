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
    DocsPath:    "/docs",         // default
})

app.Start(":3000")
// Swagger UI: http://localhost:3000/docs
// Raw spec:   http://localhost:3000/openapi.json
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
