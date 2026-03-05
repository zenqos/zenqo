# 미들웨어

> English: [Middleware](./middleware.md)

Zenqo는 표준 Go 미들웨어를 사용합니다: `func(http.Handler) http.Handler`

## 내장 미들웨어

| 미들웨어 | 설명 |
|----------|------|
| `middleware.Logger` | 상태 코드, 지연 시간, IP를 포함한 구조화된 요청/응답 로깅 |
| `middleware.RequestID` | `X-Request-Id` 주입 (기존 헤더 검증 또는 새로 생성) |
| `middleware.RealIP` | `X-Forwarded-For` 또는 `X-Real-IP`에서 실제 클라이언트 IP를 `r.RemoteAddr`에 설정 |
| `middleware.CORS()` | 설정 가능한 CORS 헤더 |
| `middleware.SecureHeaders` | 보안 헤더: CSP, HSTS, X-Frame-Options 등 |
| `middleware.RateLimit()` | IP당 고정 윈도우 요청 횟수 제한 |

`Logger`, `RequestID`, `RealIP`, 패닉 복구는 `core.NewApp()` 사용 시 자동으로 등록됩니다.

## 미들웨어 적용

```go
// 전역
app.Use(middleware.CORS())
app.Use(middleware.SecureHeaders)

// 컨트롤러 수준
c.UseControllerMiddleware(middleware.RateLimit())

// 라우트 수준
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

인자 없이 호출하면 관대한 기본값(모든 오리진, 일반 메서드)이 적용됩니다.

## 요청 횟수 제한 (Rate Limiting)

### 간단한 사용

```go
app.Use(middleware.RateLimit())  // IP당 분당 100회

app.Use(middleware.RateLimit(middleware.RateLimitConfig{
    Max:    50,
    Window: 30 * time.Second,
    KeyFunc: func(r *http.Request) string {
        return r.Header.Get("X-API-Key") // API 키 기준으로 제한
    },
    OnLimit: func(w http.ResponseWriter, r *http.Request) {
        core.Error(w, 429, "요청이 너무 많습니다")
    },
}))
```

### 라이프사이클 관리 포함 (테스트에 권장)

```go
rl := middleware.NewRateLimiter(middleware.RateLimitConfig{
    Max:    100,
    Window: time.Minute,
})
app.Use(rl.Middleware())
defer rl.Stop() // 백그라운드 정리 고루틴 종료
```

모든 요청에 설정되는 응답 헤더:
- `X-RateLimit-Limit` — 윈도우당 최대 요청 수
- `X-RateLimit-Remaining` — 남은 요청 수
- `X-RateLimit-Reset` — 윈도우 리셋 Unix 타임스탬프
- `Retry-After` — 리셋까지 남은 초 (한도 초과 시에만)

## 정적 파일 서빙

```go
app.UseStatic("/", "./public")
// ./public/index.html을 /에서 서빙
// 전역 접두사 자동 적용
```

## 요청 ID

`X-Request-Id`가 모든 요청에 주입됩니다. 핸들러에서 조회:

```go
id := middleware.GetRequestID(r.Context())
```

클라이언트가 `X-Request-Id` 헤더를 보내면 안전한 문자 허용 목록에 검증 후 그대로 사용합니다. 없으면 새 ID를 생성합니다.
