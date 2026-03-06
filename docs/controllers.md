# Controllers

> 한국어: [컨트롤러](./controllers.ko.md)

Controllers group related routes under a shared base path. Embed `core.BaseController` and call `SetBasePath` in the constructor.

## Basic Setup

```go
type UserController struct {
    core.BaseController
    svc *UserService
}

func NewUserController(svc *UserService) *UserController {
    c := &UserController{svc: svc}
    c.SetBasePath("/users")

    c.GET("/", c.list)
    c.GET("/{id}", c.getOne)
    c.POST("/", c.create)
    c.PUT("/{id}", c.update)
    c.DELETE("/{id}", c.remove)

    return c
}
```

Register with the app:

```go
app.UseController(NewUserController(svc))
```

## Route Methods

| Method | Status on success |
|--------|-------------------|
| `GET` | 200 OK |
| `POST` | 201 Created |
| `PUT` | 200 OK |
| `PATCH` | 200 OK |
| `DELETE` | 204 No Content |

Returning `nil` data from any method also yields 204.

## Route-Level Configuration

Chain options on the `*RouteDefinition` returned by each method:

```go
c.GET("/{id}", c.getOne).
    UseGuard(&OwnerGuard{}).
    UseInterceptor(&CacheInterceptor{}).
    Use(myMiddleware)
```

Available chain methods: `UseGuard`, `UseInterceptor`, `Use`

## Controller-Level Guards / Interceptors / Middleware

Apply to every route in the controller:

```go
c.UseControllerGuard(&AuthGuard{})
c.UseControllerInterceptor(&LogInterceptor{})
c.UseControllerMiddleware(middleware.RateLimit())
```

## Raw net/http Handler

When you need full `ResponseWriter` control:

```go
c.Handle("GET", "/stream", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    fmt.Fprintf(w, "data: hello\n\n")
})
```

## Path Parameters

Use `{name}` syntax (chi-style). Extract with `core.Param`:

```go
// Route: c.GET("/{id}", c.getOne)

func (c *UserController) getOne(r *http.Request) (any, error) {
    id, err := core.Param[int64](r, "id")
    if err != nil {
        return nil, err
    }
    return c.svc.FindByID(id)
}
```

Regex constraints are supported: `/{id:[0-9]+}`

## Modules

For larger apps, group controllers into a Module:

```go
type UserModule struct{}

func (m *UserModule) Name() string { return "UserModule" }
func (m *UserModule) Controllers() []core.Controller {
    svc := NewUserService()
    return []core.Controller{NewUserController(svc)}
}

// In app setup:
app.UseModule(&UserModule{})
```
