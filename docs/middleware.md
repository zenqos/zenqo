# Middleware

> 한국어: [미들웨어](./middleware.ko.md)

Zenqo uses standard Go middleware: `func(http.Handler) http.Handler`.

## Built-in Middleware

| Middleware | Description |
|------------|-------------|
| `middleware.Logger` | Structured request/response logging with status, latency, and IP |
| `middleware.RequestID` | Injects `X-Request-Id` (validates existing header or generates a new one) |
| `middleware.RealIP` | Sets `r.RemoteAddr` to the real client IP from `X-Forwarded-For` or `X-Real-IP` |
| `middleware.CORS()` | Configurable CORS headers |
| `middleware.SecureHeaders` | Security headers: CSP, HSTS, X-Frame-Options, etc. |
| `middleware.RateLimit()` | Fixed-window per-IP rate limiting |

`Logger`, `RequestID`, `RealIP`, and panic recovery are registered automatically when using `core.NewApp()`.

## Apply Middleware

```go
// Global
app.Use(middleware.CORS())
app.Use(middleware.SecureHeaders)

// Controller-level
c.UseControllerMiddleware(middleware.RateLimit())

// Route-level
c.GET("/upload", c.upload).Use(myMiddleware)
```

## CORS

```go
app.Use(middleware.CORS(middleware.CORSConfig{
    AllowedOrigins: []string{"https://example.com"},
    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders: []string{"Authorization", "Content-Type"},
    MaxAge:         86400,
}))
```

Call with no arguments for permissive defaults (all origins, common methods).

## Rate Limiting

### Simple (fire and forget)

```go
app.Use(middleware.RateLimit())  // 100 req/min per IP

app.Use(middleware.RateLimit(middleware.RateLimitConfig{
    Max:    50,
    Window: 30 * time.Second,
    KeyFunc: func(r *http.Request) string {
        return r.Header.Get("X-API-Key") // rate limit by API key
    },
    OnLimit: func(w http.ResponseWriter, r *http.Request) {
        core.Error(w, 429, "slow down")
    },
}))
```

### With Lifecycle Management (recommended for tests)

```go
rl := middleware.NewRateLimiter(middleware.RateLimitConfig{
    Max:    100,
    Window: time.Minute,
})
app.Use(rl.Middleware())
defer rl.Stop() // terminates the background cleanup goroutine
```

Response headers set on every request:
- `X-RateLimit-Limit` — max requests per window
- `X-RateLimit-Remaining` — remaining requests
- `X-RateLimit-Reset` — Unix timestamp when the window resets
- `Retry-After` — seconds until reset (only when limit exceeded)

## Static Files

```go
app.UseStatic("/", "./public")
// Serves ./public/index.html at /
// Respects global prefix automatically
```

## Request ID

`X-Request-Id` is injected into every request. Retrieve it in handlers:

```go
id := middleware.GetRequestID(r.Context())
```

If the client sends an `X-Request-Id` header, it is used as-is (validated against a safe character allowlist). Otherwise a new ID is generated.
