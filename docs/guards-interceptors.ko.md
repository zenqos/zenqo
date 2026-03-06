# 가드 & 인터셉터

> English: [Guards & Interceptors](./guards-interceptors.md)

## 가드

가드는 요청의 진행 여부를 제어합니다. `core.Guard`를 구현합니다:

```go
type Guard interface {
    CanActivate(r *http.Request) (bool, error)
}
```

`(true, nil)`을 반환하면 허용, `(false, err)`을 반환하면 거부합니다.

```go
type AuthGuard struct{}

func (g *AuthGuard) CanActivate(r *http.Request) (bool, error) {
    token := r.Header.Get("Authorization")
    if token == "" {
        return false, core.ErrUnauthorized("토큰이 없습니다")
    }
    // 토큰 검증...
    return true, nil
}
```

### 적용 레벨

```go
app.UseGlobalGuard(&AuthGuard{})           // 앱의 모든 라우트
c.UseControllerGuard(&AuthGuard{})         // 컨트롤러의 모든 라우트
c.GET("/admin", h).UseGuard(&AdminGuard{}) // 단일 라우트
```

### 가드 거부 동작

| `CanActivate` 반환값 | 응답 |
|----------------------|------|
| `(false, *HTTPError)` | HTTPError의 상태 코드 + 메시지 |
| `(false, nil)` | 403 Forbidden |
| `(false, 기타 에러)` | 500 Internal Server Error |

### 미들웨어로 사용

```go
// 기본 에러 형식
app.Use(core.GuardToMiddleware(&AuthGuard{}))

// 커스텀 에러 핸들러 포함 (예: RFC 9457)
app.Use(core.GuardToMiddleware(&AuthGuard{}, myErrorHandler))
```

---

## 인터셉터

인터셉터는 핸들러 **전후**에 코드를 실행합니다. 로깅, 타이밍, 캐싱, 컨텍스트 값 주입에 활용합니다.

`core.Interceptor`를 구현합니다:

```go
type Interceptor interface {
    Before(ctx context.Context, r *http.Request) context.Context
    After(ctx context.Context, w http.ResponseWriter, statusCode int)
}
```

`Before`는 핸들러 전에 실행되며 컨텍스트에 값을 첨부할 수 있습니다.  
`After`는 핸들러 후에 실행되며 응답 상태 코드에 접근할 수 있습니다.

### 예시: 요청 처리 시간 측정

```go
type TimingInterceptor struct{}

func (i *TimingInterceptor) Before(ctx context.Context, r *http.Request) context.Context {
    return context.WithValue(ctx, "start", time.Now())
}

func (i *TimingInterceptor) After(ctx context.Context, w http.ResponseWriter, status int) {
    start := ctx.Value("start").(time.Time)
    log.Printf("상태: %d, 처리 시간: %s", status, time.Since(start))
}
```

### 적용 레벨

```go
c.UseControllerInterceptor(&TimingInterceptor{}) // 컨트롤러의 모든 라우트
c.GET("/", h).UseInterceptor(&CacheInterceptor{}) // 단일 라우트
```

### 미들웨어로 사용

```go
app.Use(core.InterceptorToMiddleware(&TimingInterceptor{}))
```
