# 컨트롤러

> English: [Controllers](./controllers.md)

컨트롤러는 관련 라우트를 공통 기본 경로 아래에 묶습니다. `core.BaseController`를 임베드하고 생성자에서 `SetBasePath`를 호출합니다.

## 기본 설정

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

앱에 등록:

```go
app.UseController(NewUserController(svc))
```

## 라우트 메서드

| 메서드 | 성공 시 상태 코드 |
|--------|-------------------|
| `GET` | 200 OK |
| `POST` | 201 Created |
| `PUT` | 200 OK |
| `PATCH` | 200 OK |
| `DELETE` | 204 No Content |

어떤 메서드든 `nil` 데이터를 반환하면 204로 응답합니다.

## 라우트 수준 설정

각 메서드가 반환하는 `*RouteDefinition`에 체이닝:

```go
c.GET("/{id}", c.getOne).
    UseGuard(&OwnerGuard{}).
    UseInterceptor(&CacheInterceptor{}).
    Use(myMiddleware)
```

사용 가능한 체인 메서드: `UseGuard`, `UseInterceptor`, `Use`

## 컨트롤러 수준 가드 / 인터셉터 / 미들웨어

컨트롤러의 모든 라우트에 적용:

```go
c.UseControllerGuard(&AuthGuard{})
c.UseControllerInterceptor(&LogInterceptor{})
c.UseControllerMiddleware(middleware.RateLimit())
```

## 로우 net/http 핸들러

`ResponseWriter`를 직접 제어해야 할 때:

```go
c.Handle("GET", "/stream", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    fmt.Fprintf(w, "data: hello\n\n")
})
```

## 경로 파라미터

`{name}` 문법(chi 스타일)으로 선언하고 `core.Param`으로 추출:

```go
// 라우트: c.GET("/{id}", c.getOne)

func (c *UserController) getOne(r *http.Request) (any, error) {
    id, err := core.Param[int64](r, "id")
    if err != nil {
        return nil, err
    }
    return c.svc.FindByID(id)
}
```

정규식 제약도 지원합니다: `/{id:[0-9]+}`

## 모듈

대규모 앱에서는 컨트롤러를 모듈로 묶을 수 있습니다:

```go
type UserModule struct{}

func (m *UserModule) Name() string { return "UserModule" }
func (m *UserModule) Controllers() []core.Controller {
    svc := NewUserService()
    return []core.Controller{NewUserController(svc)}
}

// 앱 설정 시:
app.UseModule(&UserModule{})
```
