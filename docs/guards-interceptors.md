# Guards & Interceptors

> 한국어: [가드 & 인터셉터](./guards-interceptors.ko.md)

## Guards

Guards control whether a request is allowed to proceed. Implement `core.Guard`:

```go
type Guard interface {
    CanActivate(r *http.Request) (bool, error)
}
```

Return `(true, nil)` to allow, or `(false, err)` to reject.

```go
type AuthGuard struct{}

func (g *AuthGuard) CanActivate(r *http.Request) (bool, error) {
    token := r.Header.Get("Authorization")
    if token == "" {
        return false, core.ErrUnauthorized("missing token")
    }
    // validate token...
    return true, nil
}
```

### Apply at Any Level

```go
app.UseGlobalGuard(&AuthGuard{})           // every route in the app
c.UseControllerGuard(&AuthGuard{})         // every route in the controller
c.GET("/admin", h).UseGuard(&AdminGuard{}) // single route
```

### Guard Rejection Behavior

| `CanActivate` return | Response |
|----------------------|----------|
| `(false, *HTTPError)` | HTTPError's status + message |
| `(false, nil)` | 403 Forbidden |
| `(false, other error)` | 500 Internal Server Error |

### Use as Middleware

```go
// Default error format
app.Use(core.GuardToMiddleware(&AuthGuard{}))

// With custom error handler (e.g. RFC 9457)
app.Use(core.GuardToMiddleware(&AuthGuard{}, myErrorHandler))
```

---

## Interceptors

Interceptors run code **before** and **after** the handler. Use them for logging, timing, caching, or injecting context values.

Implement `core.Interceptor`:

```go
type Interceptor interface {
    Before(ctx context.Context, r *http.Request) context.Context
    After(ctx context.Context, w http.ResponseWriter, statusCode int)
}
```

`Before` runs before the handler and can attach values to the context.  
`After` runs after the handler with access to the response status code.

### Example: Request Timing

```go
type TimingInterceptor struct{}

func (i *TimingInterceptor) Before(ctx context.Context, r *http.Request) context.Context {
    return context.WithValue(ctx, "start", time.Now())
}

func (i *TimingInterceptor) After(ctx context.Context, w http.ResponseWriter, status int) {
    start := ctx.Value("start").(time.Time)
    log.Printf("%-6s %s %d %s", r.Method, r.URL.Path, status, time.Since(start))
}
```

### Apply at Any Level

```go
c.UseControllerInterceptor(&TimingInterceptor{}) // every route in controller
c.GET("/", h).UseInterceptor(&CacheInterceptor{}) // single route
```

### Use as Middleware

```go
app.Use(core.InterceptorToMiddleware(&TimingInterceptor{}))
```
