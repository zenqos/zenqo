# Middleware

> 한국어: [미들웨어](./middleware.ko.md)

Zenqo uses standard Go middleware: `func(http.Handler) http.Handler`.

## Built-in Middleware

| Middleware | Description |
|------------|-------------|
| `middleware.Logger` | Structured request/response logging with status, latency, and IP |
| `middleware.RequestID` | Injects `X-Request-Id` (validates existing header or generates a new one) |
| `middleware.RealIPWithConfig()` | Sets `r.RemoteAddr` from trusted proxy headers |
| `middleware.CORS()` | Configurable CORS headers |
| `middleware.SecureHeaders()` | Security headers: CSP, HSTS, X-Frame-Options, etc. |
| `middleware.RateLimit()` | Sliding-window per-IP rate limiting |
| `middleware.CSRF()` | CSRF protection with double-submit cookie pattern |

`Logger`, `RequestID`, `RealIP` (private network CIDRs), and panic recovery are registered automatically when using `core.NewApp()`.

## Apply Middleware

```go
// Global
app.Use(middleware.CORS())
app.Use(middleware.SecureHeaders())

// Controller-level
c.UseControllerMiddleware(middleware.RateLimit())

// Route-level
c.GET("/upload", c.upload).Use(myMiddleware)
```

## CORS

```go
app.Use(middleware.CORS(middleware.CORSConfig{
    AllowOrigins: []string{"https://example.com"},
    AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders: []string{"Authorization", "Content-Type"},
    MaxAge:       86400,
}))
```

Call with no arguments for permissive defaults (all origins, common methods). Partial config merges with defaults — only override what you need.

## RealIP

By default, `NewApp()` trusts proxy headers from private network CIDRs only. To customize:

```go
app.Use(middleware.RealIPWithConfig(middleware.RealIPConfig{
    TrustedProxies: []string{"10.0.0.0/8", "172.16.0.0/12"},
}))
```

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
