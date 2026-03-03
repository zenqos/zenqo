# Changelog

## Unreleased

### 10. Guard Interface Improvement

Guards can now return custom HTTP status codes instead of only 403.

**Updated `core/middleware.go`:**
- New `guardReject()` helper centralizes Guard denial logic
- `GuardToMiddleware()` and `applyGuard()` now check `*HTTPError` type:
  - `(false, *HTTPError)` → responds with the HTTPError's status and message (e.g. 401, 429)
  - `(false, nil)` → 403 Forbidden (unchanged)
  - `(false, other error)` → 500 Internal Server Error (unchanged)
- **Backward compatible** — no interface changes, existing Guards work as before

```go
// Before: Guards could only return 403
func (g *AuthGuard) CanActivate(r *http.Request) (bool, error) {
    return false, nil // always 403
}

// After: Guards can return any HTTP status
func (g *AuthGuard) CanActivate(r *http.Request) (bool, error) {
    return false, core.ErrUnauthorized("invalid token") // 401
}
```

---

### 11. Extended Validation Rules

Added 12 new validation rules to `core/rules.go`, bringing the total to 17.

| Rule | Example | Behavior |
|------|---------|----------|
| `url` | `validate:"url"` | Must be a valid URL (http/https/ftp) |
| `uuid` | `validate:"uuid"` | Must be a valid UUID format |
| `alpha` | `validate:"alpha"` | Letters only |
| `alphanum` | `validate:"alphanum"` | Letters and numbers only |
| `numeric` | `validate:"numeric"` | Digit characters only |
| `len=N` | `validate:"len=10"` | Exact length (strings, slices, maps) |
| `regex=PATTERN` | `validate:"regex=^[a-z]+$"` | Must match regex pattern |
| `contains=STR` | `validate:"contains=@"` | Must contain substring |
| `startswith=STR` | `validate:"startswith=http"` | Must start with prefix |
| `endswith=STR` | `validate:"endswith=.com"` | Must end with suffix |
| `lowercase` | `validate:"lowercase"` | Must be all lowercase |
| `uppercase` | `validate:"uppercase"` | Must be all uppercase |

New test file: `core/rules_test.go` — success/failure tests for all 12 rules.

---

### 12. Comprehensive Core Tests

New test files covering all core logic that previously had no tests:

| File | Tests |
|------|-------|
| `core/controller_test.go` | `adapt()` (GET/POST/nil/error), `RegisterRoutes` (controller guard, route-level guard), `SetBasePath` panics |
| `core/middleware_test.go` | `GuardToMiddleware` (allow, deny nil, deny HTTPError 401/429), `InterceptorToMiddleware` (Before/After, context propagation), `statusWriter` (WriteHeader, Flush, Hijack) |
| `core/errors_test.go` | `DefaultErrorHandler` (ValidationError, HTTPError, generic error), error helper constructors |
| `core/app_test.go` | Full integration: GET/POST/PUT/PATCH/DELETE routes, GlobalPrefix, Module registration, Global Guard, 404/405 JSON, `Bind` integration (valid JSON, wrong Content-Type, body too large, validation failure, invalid JSON), custom error handler |
| `core/recover_test.go` | Panic recovery → 500 JSON, `http.ErrAbortHandler` re-panic, no-panic passthrough |

---

### 13. Benchmarks

New file: `core/benchmark_test.go`

Run with `go test -bench=. -benchmem ./core/`

| Benchmark | What it measures |
|-----------|-----------------|
| `BenchmarkChiRawHandler` vs `BenchmarkZenqoAdapt` | Framework overhead per request |
| `BenchmarkStdJSONMarshal` vs `BenchmarkZenqoMarshal` | Custom encoder vs encoding/json |
| `BenchmarkValidatePass` / `BenchmarkValidateFail` | Struct validation speed |
| `BenchmarkGuardAllow` / `BenchmarkGuardDeny` / `BenchmarkGuardChain3` | Guard middleware overhead |

---

### 14. JWT Authentication Example

New example: `examples/auth/` — demonstrates real-world JWT authentication.

**Features demonstrated:**
- `JWTGuard` — Guard returning 401 Unauthorized (improved Guard interface)
- Controller-level Guard — all user routes protected
- `LogInterceptor` — request/response logging via Interceptor
- `Bind[T]` + validation — DTO decoding with struct tag validation
- Full auth flow — register, login, protected CRUD

**Structure:**
```
examples/auth/
├── main.go
├── go.mod
├── .gitignore
└── internal/
    ├── app/app.go          # Wiring with adapter pattern
    ├── config/config.go
    ├── auth/
    │   ├── guard.go        # JWTGuard
    │   ├── handler.go      # Login/Register
    │   ├── jwt.go          # Token generation/parsing
    │   └── dto.go          # LoginDTO, RegisterDTO
    └── user/
        ├── handler.go      # Protected user CRUD
        └── service.go      # In-memory store
```

---

### 1. ValidationError Type & Field-Level Error Response

Added structured validation error support so that API consumers receive per-field error details instead of a generic 400 message.

**New types in `core/errors.go`:**
- `FieldError` — carries a `field` name and `message`
- `ValidationError` — wraps a slice of `FieldError`
- `ErrValidation()` — convenience constructor

**New response helpers in `core/response.go`:**
- `ValidationErrorResponse` — JSON envelope: `{ "code": 400, "message": "validation failed", "errors": [...] }`
- `ValidationFailed()` — writes a 400 response with field-level errors

**Updated `core/controller.go`:**
- `adapt()` now checks for `*ValidationError` before `*HTTPError`, routing validation failures to `ValidationFailed()` automatically.

---

### 2. `core.Param[T]` — Type-Safe URL Parameters

New file: **`core/params.go`**

A generic function that extracts and converts URL path parameters in a single call.

```go
// Before (3 lines)
id, err := strconv.ParseInt(core.URLParam(r, "id"), 10, 64)
if err != nil { return nil, core.ErrBadRequest("invalid user id") }

// After (1 line)
id, err := core.Param[int64](r, "id")
```

Supported types: `string`, `int`, `int64`, `uint`, `uint64`.
Returns `ErrBadRequest` automatically for missing or unparseable values.

---

### 3. Validation on Bind

New file: **`core/validate.go`**

Struct fields annotated with `validate:"..."` tags are automatically checked when `Bind[T]()` decodes the request body.

| Rule | Example | Behavior |
|------|---------|----------|
| `required` | `validate:"required"` | Fails on zero value |
| `min=N` | `validate:"min=2"` | String: len < N fails. Int: value < N fails |
| `max=N` | `validate:"max=50"` | String: len > N fails. Int: value > N fails |
| `email` | `validate:"email"` | Must match email regex (empty string passes — use `required` separately) |
| `oneof=a\|b\|c` | `validate:"oneof=admin\|user"` | Value must be in the allowed set |

**Pointer field support:** Validation fully supports `*string`, `*int`, etc. for partial-update DTOs.
- `nil` + `required` → validation fails
- `nil` + no `required` → field is skipped entirely (not sent = don't touch)
- Non-nil pointer → dereferenced and validated with normal rules

**Updated `core/bind.go`:**
- `Bind[T]()` now calls `validate()` after JSON decoding. Validation failures are returned as `*ValidationError`, which `adapt()` converts to a field-level 400 response.

---

### 4. `app.GET/POST/etc` — Direct Routing on App

**Updated `core/app.go`:**

Added `GET`, `POST`, `PUT`, `PATCH`, `DELETE` methods directly on the `App` struct, eliminating the need for a controller when defining simple routes.

```go
// Before — requires a controller struct
type helloController struct { core.BaseController }
func newHelloController() *helloController {
    c := &helloController{}
    c.SetBasePath("/")
    c.GET("/", c.index)
    return c
}

// After — direct routing, no boilerplate
app := core.NewApp()
app.GET("/", func(r *http.Request) (any, error) {
    return map[string]string{"message": "Hello!"}, nil
})
```

Implementation details:
- New `root` field (`BaseController`) on the `App` struct, initialized with `basePath = "/"` in `NewApp()`
- `buildRoutes()` registers root routes after controllers and modules
- Empty-server detection updated to also check `len(a.root.routes)`

---

### 5. Example Updates

**`examples/basic/internal/app/app.go`:**
- Removed `helloController` struct entirely
- Uses `app.GET()` direct routing — simpler than the equivalent Gin/Echo code

**`examples/crud/internal/user/dto.go`:**
- Added `validate` tags to `CreateUserDTO` (`required`, `min`, `max`, `email`) and `UpdateUserDTO` (`max`, `email`)
- `UpdateUserDTO` uses `*string` pointer fields to distinguish "not sent" (`nil`) from "set to empty" (`*""`)

**`examples/crud/internal/user/handler.go`:**
- Replaced all `strconv.ParseInt(core.URLParam(...))` calls with `core.Param[int64](r, "id")`
- Removed `strconv` import

**`examples/crud/internal/user/service.go`:**
- `Update()` now checks `dto.Name != nil` / `dto.Email != nil` instead of empty string comparison

---

### Files Changed

| File | Action | Description |
|------|--------|-------------|
| `core/errors.go` | Modified | Added `FieldError`, `ValidationError`, `ErrValidation` |
| `core/response.go` | Modified | Added `ValidationErrorResponse`, `ValidationFailed` |
| `core/controller.go` | Modified | `adapt()` handles `ValidationError` before `HTTPError` |
| `core/params.go` | **New** | `Param[T]` generic URL parameter extraction |
| `core/validate.go` | **New** | Struct tag validation engine with pointer field support |
| `core/bind.go` | Modified | `Bind[T]` calls `validate()` after decode |
| `core/app.go` | Modified | `root` field, `GET/POST/PUT/PATCH/DELETE`, `buildRoutes` update |
| `examples/basic/.../app.go` | Modified | Simplified with direct routing |
| `examples/crud/.../dto.go` | Modified | Added `validate` tags, `*string` for `UpdateUserDTO` |
| `examples/crud/.../handler.go` | Modified | Uses `Param[T]` instead of `strconv` |
| `examples/crud/.../service.go` | Modified | Pointer-based nil check for partial updates |

---

### 6. Production Hardening

**`core/controller.go` — Error logging in `adapt()`:**
- Internal errors (500) are now logged via `zerr()` before sending the response. Previously, the actual error was silently discarded, making debugging impossible.

**`core/bind.go` — Request body size limit:**
- `Bind[T]()` now wraps `r.Body` with `io.LimitReader` to prevent memory exhaustion from oversized payloads. Default limit: 1 MB. Configurable via `core.MaxBodySize`.

**`core/app.go` — Graceful shutdown error handling:**
- `srv.Shutdown()` error is now logged instead of silently discarded.

**`core/marshal.go` — Struct field caching:**
- Resolved struct field metadata (key names, omitempty flags) is now cached per type in a `sync.Map`. Eliminates repeated reflection overhead on every JSON response.

**`core/validate.go` — Nested struct support:**
- Validation now recurses into nested struct fields. If a struct field contains fields with `validate` tags, they are checked automatically.
- Added `IsExported()` guard to prevent reflection panics on unexported fields.

---

### 7. Built-in CORS Support

New file: **`core/cors.go`**

```go
// Development — allow everything
app.UseCORS()

// Production — restrict origins
app.UseCORS(core.CORSConfig{
    AllowOrigins: []string{"https://myapp.com"},
})
```

- `CORSConfig` struct with `AllowOrigins`, `AllowMethods`, `AllowHeaders`, `AllowCredentials`, `MaxAge`
- `DefaultCORSConfig()` returns a permissive default (`*`)
- Handles OPTIONS preflight requests automatically
- `AllowCredentials: true` enables cookie/auth header support

---

### 8. Core Tests

New test files:

| File | Coverage |
|------|----------|
| `core/validate_test.go` | required, min/max, email, oneof, pointer nil/skip/value, nested struct, multiple errors, non-struct |
| `core/params_test.go` | string/int/int64/uint64 extraction, missing param, invalid int, negative uint |
| `core/marshal_test.go` | camelCase conversion, simple struct, omitempty, exclude tag, nil slice, nil value, embedded struct, zenqo tag, cache consistency |

---

### Updated Files Summary

| File | Action | Description |
|------|--------|-------------|
| `core/controller.go` | Modified | Error logging in `adapt()` for 500 responses |
| `core/bind.go` | Modified | Body size limit via `io.LimitReader`, `MaxBodySize` var |
| `core/app.go` | Modified | Shutdown error handling |
| `core/marshal.go` | Modified | `sync.Map` field cache, `getStructFields()` |
| `core/validate.go` | Modified | Nested struct recursion, `IsExported()` guard |
| `core/cors.go` | **New** | CORS middleware with `UseCORS()` |
| `core/validate_test.go` | **New** | Validation test suite |
| `core/params_test.go` | **New** | URL parameter test suite |
| `core/marshal_test.go` | **New** | JSON marshaling test suite |

---

### 9. Security & Logging Hardening

**Server timeouts (Slowloris prevention)** — `core/app.go`:
- `ReadTimeout: 15s`, `WriteTimeout: 15s`, `IdleTimeout: 60s` added to `http.Server`
- Prevents slow clients from holding connections indefinitely

**Security headers** — `core/app.go`:
- `X-Content-Type-Options: nosniff` and `X-Frame-Options: DENY` set on every response via `secureHeaders` middleware registered in `NewApp()`

**Content-Type validation** — `core/bind.go`:
- `Bind[T]()` now rejects non-JSON content types with a 400 error
- Empty Content-Type is allowed (common in curl/testing)

**Unified logging** — `core/response.go`, `core/middleware.go`:
- All `log.Printf("[Zenqo] ...")` calls replaced with `zerr()` for consistent log format across the entire framework

**TTY-aware colors** — `core/app.go`:
- `zlog()` and `zerr()` now detect whether stdout/stderr is a terminal
- ANSI escape codes are automatically disabled when piping to files, Docker, or cloud log aggregators

**Request context in error logs** — `core/controller.go`:
- 500 errors now log request ID, HTTP method, and path: `[req-id] POST /api/v1/users — <error>`
- Enables correlation of errors with specific requests in production

**Empty addr guard** — `core/app.go`:
- `Start("")` no longer panics with index out of range

---

### Updated Files Summary (Security & Logging)

| File | Action | Description |
|------|--------|-------------|
| `core/app.go` | Modified | Server timeouts, `secureHeaders` middleware, TTY-aware colors, empty addr guard |
| `core/bind.go` | Modified | Content-Type validation |
| `core/controller.go` | Modified | Request context (method, path, reqID) in 500 error logs |
| `core/response.go` | Modified | `log.Printf` → `zerr()` |
| `core/middleware.go` | Modified | `log.Printf` → `zerr()` |
| `core/cors.go` | Modified | `AllowCredentials` support |
