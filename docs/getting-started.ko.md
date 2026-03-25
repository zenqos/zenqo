# 시작하기

> English: [Getting Started](./getting-started.md)

## 설치

### macOS

```bash
# Homebrew (권장)
brew install zenqos/tap/zenqo

# 확인
zenqo --help
```

### Linux

```bash
# 설치 스크립트 한 줄로 설치
curl -fsSL https://raw.githubusercontent.com/zenqos/zenqo/main/install.sh | sh

# 확인
zenqo --help
```

### Windows

**관리자 권한으로 PowerShell**을 열고 실행:

```powershell
irm https://raw.githubusercontent.com/zenqos/zenqo/main/install.ps1 | iex
```

바이너리가 `%LOCALAPPDATA%\zenqo`에 설치되고 PATH에 자동으로 추가됩니다.

```powershell
# 확인 (새 터미널에서)
zenqo --help
```

### go install (전 플랫폼)

Go 1.21 이상 필요. 바이너리는 `$GOPATH/bin`에 설치됩니다.

```bash
go install github.com/zenqos/zenqo/cmd/zenqo@latest

# PATH에 추가 (아직 없다면 ~/.zshrc 또는 ~/.bashrc에 추가)
export PATH="$PATH:$(go env GOPATH)/bin"

# 확인
zenqo --help
```

---

## 새 프로젝트 생성

```bash
zenqo new my-app
cd my-app && zenqo dev
```

권장 디렉토리 구조로 바로 실행 가능한 프로젝트를 생성합니다:

```
my-app/
├── main.go                    # 진입점
├── internal/
│   ├── app/app.go             # 컨트롤러 등록
│   └── config/config.go       # 환경 설정
├── go.mod
└── .gitignore
```

제너레이터로 기능 추가:

```bash
zenqo generate resource user   # internal/user/ 에 handler, service, dto, test 생성
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

    if err := app.Start(":3000"); err != nil {
        log.Fatal(err)
    }
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

## CLI 명령어

```bash
zenqo new <name>                 # 새 프로젝트 생성
zenqo dev                        # 개발 서버 실행
zenqo dev --watch                # 핫 리로드 (.go 파일 변경 시 자동 재시작)
zenqo generate resource <name>   # 컨트롤러 + 서비스 + DTO + 테스트
zenqo generate controller <name> # 컨트롤러만
zenqo generate guard <name>      # 가드 보일러플레이트
zenqo generate interceptor <name># 인터셉터 보일러플레이트
```

## 다음 단계

- [컨트롤러](./controllers.ko.md) — 기본 경로 아래에 라우트 묶기
- [요청 바인딩](./request-binding.ko.md) — 바디, 파라미터, 헤더, 파일 디코딩
- [에러 처리](./error-handling.ko.md) — 타입 에러와 커스텀 핸들러
- [가드 & 인터셉터](./guards-interceptors.ko.md) — 접근 제어와 라이프사이클 훅
- [미들웨어](./middleware.ko.md) — 내장 및 커스텀 미들웨어
- [OpenAPI](./openapi.ko.md) — 자동 스펙 생성 및 Swagger UI
