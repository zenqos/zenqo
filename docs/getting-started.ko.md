# 시작하기

> English: [Getting Started](./getting-started.md)

## 설치

```bash
# Go
go install github.com/zenqos/zenqo/cmd/zenqo@latest

# Homebrew
brew install zenqos/tap/zenqo
```

## 새 프로젝트 생성

```bash
zenqo new my-app
cd my-app && go run .
```

권장 디렉토리 구조로 바로 실행 가능한 프로젝트를 생성합니다:

```
my-app/
├── cmd/my-app/main.go         # 진입점
├── internal/
│   ├── app/app.go             # 컨트롤러 등록
│   └── example/               # 생성된 기능 모듈
│       ├── controller.go
│       ├── service.go
│       └── dto.go
└── go.mod
```

## Hello World (CLI 없이)

```go
package main

import (
    "log"
    "net/http"
    "github.com/zenqos/zenqo/core"
)

func main() {
    app := core.NewApp()

    app.GET("/", func(r *http.Request) (any, error) {
        return map[string]string{"message": "Hello, Zenqo!"}, nil
    })

    log.Fatal(app.Start(":3000"))
}
```

```bash
curl http://localhost:3000/
# { "success": true, "data": { "message": "Hello, Zenqo!" } }
```

## 전역 경로 접두사

```go
app.SetGlobalPrefix("/api")
// 모든 라우트가 /api 하위에서 서빙됩니다
```

## 그레이스풀 셧다운

Zenqo는 `SIGINT` / `SIGTERM` 시그널을 자동으로 처리합니다. 대기 시간 조정:

```go
app.SetShutdownTimeout(10 * time.Second) // 기본값: 30초
```

## 보일러플레이트 생성

```bash
# 전체 리소스: 컨트롤러 + 서비스 + DTO + 테스트
zenqo generate resource user

# 개별 생성
zenqo generate controller order
zenqo generate guard auth
zenqo generate interceptor logging
```

## HTTP 핸들러로 사용 (테스트 / 임베딩)

```go
handler := app.Handler() // 라우트 빌드 후 http.Handler 반환
ts := httptest.NewServer(handler)
defer ts.Close()
```

## 다음 단계

- [컨트롤러](./controllers.ko.md) — 기본 경로 아래에 라우트 묶기
- [요청 바인딩](./request-binding.ko.md) — 바디, 파라미터, 헤더, 파일 디코딩
- [에러 처리](./error-handling.ko.md) — 타입 에러와 커스텀 핸들러
- [가드 & 인터셉터](./guards-interceptors.ko.md) — 접근 제어와 라이프사이클 훅
- [미들웨어](./middleware.ko.md) — 내장 및 커스텀 미들웨어
- [OpenAPI](./openapi.ko.md) — 자동 스펙 생성 및 Swagger UI
