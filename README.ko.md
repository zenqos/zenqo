<p align="right">
  <a href="README.md">English</a> | <b>한국어</b>
</p>

<p align="center">
  <img src="./assets/logo-wordmark.svg" alt="Zenqo" width="260" />
</p>

<p align="center">반환값 기반의 <a href="https://go.dev" target="_blank">Go</a> 웹 프레임워크 — 깔끔하고 확장 가능한 서버 사이드 애플리케이션 구축을 위해</p>

<p align="center">
  <a href="https://github.com/zenqos/zenqo/actions/workflows/ci.yml"><img src="https://github.com/zenqos/zenqo/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <a href="https://pkg.go.dev/github.com/zenqos/zenqo"><img src="https://pkg.go.dev/badge/github.com/zenqos/zenqo.svg" alt="Go Reference" /></a>
  <a href="https://goreportcard.com/report/github.com/zenqos/zenqo"><img src="https://goreportcard.com/badge/github.com/zenqos/zenqo" alt="Go Report Card" /></a>
  <a href="https://github.com/zenqos/zenqo/blob/main/LICENSE"><img src="https://img.shields.io/github/license/zenqos/zenqo" alt="License" /></a>
</p>

## 소개

Zenqo는 효율적이고 유지보수하기 쉬운 Go 웹 애플리케이션을 위한 프레임워크입니다. 핸들러는 `http.ResponseWriter`에 직접 쓰는 대신 `(data, error)`를 반환하고, 프레임워크가 JSON 직렬화, 상태 코드, 에러 응답을 자동으로 처리합니다.

내부적으로 <a href="https://github.com/go-chi/chi" target="_blank">chi</a>를 라우터로 사용합니다.

## 철학

핸들러는 `(data, error)`를 반환하는 순수 함수입니다. 프레임워크가 JSON 직렬화, 상태 코드, 에러 응답을 처리하므로 코드는 비즈니스 로직에만 집중할 수 있습니다.

```go
// Zenqo
func getUser(r *http.Request) (any, error) {
    user, err := svc.FindByID(id)
    if err != nil {
        return nil, core.ErrNotFound("user not found")
    }
    return user, nil
}
```

## 시작하기

```bash
go install github.com/zenqos/zenqo/cmd/zenqo@latest
zenqo new my-app
cd my-app && go run .
```

→ 전체 가이드: [시작하기](./docs/getting-started.ko.md)

## 주요 기능

- **반환값 핸들러** — `(any, error)` 시그니처, JSON 및 상태 코드 자동 처리
- **컨트롤러** — `BaseController`로 라우트를 기본 경로 아래에 묶기
- **Bind & Validate** — `Bind[T]`로 JSON 디코딩과 유효성 검사를 한 번에
- **가드 & 인터셉터** — 라우트, 컨트롤러, 전역 수준의 접근 제어
- **에러 처리** — 패닉 복구, 타입 에러, 커스터마이즈 가능한 에러 핸들러
- **자동 camelCase** — `json` 태그 없이도 camelCase JSON 직렬화
- **내장 미들웨어** — CORS, 보안 헤더, 요청 ID, 패닉 복구

## 예제

- [`examples/basic`](./examples/basic) — 컨트롤러 없이 직접 라우트 등록
- [`examples/crud`](./examples/crud) — 컨트롤러 + 서비스 패턴의 CRUD API
- [`examples/auth`](./examples/auth) — 가드, 인터셉터, Bind+Validation을 활용한 JWT 인증

## 문서

| 주제 | EN | KO |
|------|----|----|
| 시작하기 | [→](./docs/getting-started.md) | [→](./docs/getting-started.ko.md) |
| 컨트롤러 | [→](./docs/controllers.md) | [→](./docs/controllers.ko.md) |
| 요청 바인딩 | [→](./docs/request-binding.md) | [→](./docs/request-binding.ko.md) |
| 에러 처리 | [→](./docs/error-handling.md) | [→](./docs/error-handling.ko.md) |
| 가드 & 인터셉터 | [→](./docs/guards-interceptors.md) | [→](./docs/guards-interceptors.ko.md) |
| 미들웨어 | [→](./docs/middleware.md) | [→](./docs/middleware.ko.md) |
| OpenAPI | [→](./docs/openapi.md) | [→](./docs/openapi.ko.md) |

- [CHANGELOG](./CHANGELOG.md) — 버전 히스토리
- [pkg.go.dev](https://pkg.go.dev/github.com/zenqos/zenqo) — API 레퍼런스 (GoDoc)

## 기여자

<table>
  <tbody>
    <tr>
      <td align="center" valign="top" width="14.28%">
        <a href="https://github.com/ftery0">
          <img src="https://avatars.githubusercontent.com/u/127281057?v=4" width="80px;" alt="ftery0"/><br />
          <sub><b>ftery0</b></sub>
        </a><br />
        <a title="Code">💻</a>
      </td>
      <td align="center" valign="top" width="14.28%">
        <a href="https://github.com/RogueTex">
          <img src="https://avatars.githubusercontent.com/u/218665445?v=4" width="80px;" alt="Raghu :)"/><br />
          <sub><b>Raghu :)</b></sub>
        </a><br />
        <a title="Code">💻</a>
      </td>
      <td align="center" valign="top" width="14.28%">
        <a href="https://github.com/aicontentcreate2023-star">
          <img src="https://avatars.githubusercontent.com/u/259026231?v=4" width="80px;" alt="aicontentcreate2023-star"/><br />
          <sub><b>aicontentcreate2023-star</b></sub>
        </a><br />
        <a title="Code">💻</a>
      </td>
    </tr>
  </tbody>
</table>

## 기여하기

기여를 환영합니다! PR 제출 전에 [기여 가이드](CONTRIBUTING.md)를 읽어주세요.

## 보안

보안 취약점 발견 시 **공개 이슈를 열지 마세요**. [보안 정책](SECURITY.md)의 안내에 따라 신고해주세요.

## 연락처

- 작성자 — [@ftery0](https://github.com/ftery0)

## 라이선스

Zenqo는 [MIT 라이선스](LICENSE)를 따릅니다.
