# OpenAPI

> English: [OpenAPI](./openapi.md)

Zenqo는 OpenAPI 3.1 스펙을 자동 생성하고 Swagger UI를 제공합니다 — 수동 YAML 작성 불필요.

## 설정

모든 컨트롤러 등록 후, `app.Start` 전에 `openapi.Mount` 호출:

```go
import "github.com/zenqos/zenqo/openapi"

openapi.Mount(app, openapi.Config{
    Title:       "My API",
    Version:     "1.0.0",
    Description: "선택적 설명",
    SpecPath:    "/openapi.json", // 기본값
    YAMLPath:    "/openapi.yaml", // 기본값 ("-"로 비활성화)
    DocsPath:    "/docs",         // 기본값
    Security:    []openapi.SecurityDef{openapi.BearerAuth()}, // 선택
})

app.Start(":3000")
// Swagger UI: http://localhost:3000/docs
// JSON 스펙:  http://localhost:3000/openapi.json
// YAML 스펙:  http://localhost:3000/openapi.yaml
```

## 라우트 어노테이션

`*RouteDefinition`에 빌더 메서드를 체이닝해서 스펙을 풍부하게 만들 수 있습니다:

```go
c.GET("/users", c.list).
    Summary("사용자 목록 조회").
    Description("페이지네이션된 사용자 목록을 반환합니다.").
    Tags("users").
    Response(200, []UserDTO{})

c.POST("/users", c.create).
    Summary("사용자 생성").
    Tags("users").
    Body(CreateUserDTO{}).
    Response(201, UserDTO{}).
    Response(400, nil).
    Response(409, nil)

c.DELETE("/users/{id}", c.remove).
    Summary("사용자 삭제").
    Tags("users").
    Deprecated()
```

| 빌더 | 설명 |
|------|------|
| `.Summary(s)` | 한 줄 요약 |
| `.Description(s)` | 상세 설명 |
| `.Tags(t...)` | 하나 이상의 태그로 그룹화 |
| `.Body(v)` | 요청 바디 스키마 추론용 제로값 구조체 전달 |
| `.Response(status, v)` | 응답 스키마용 제로값 구조체 또는 빈 바디는 `nil` |
| `.Deprecated()` | 사용 중단 표시 |
| `.NoSecurity()` | 글로벌 보안 오버라이드 (공개 라우트 표시) |

## 스키마 자동 추론

Go 구조체 타입에서 스키마가 자동으로 추론됩니다:

| Go 타입 | OpenAPI 타입 |
|---------|--------------|
| `string` | `string` |
| `int`, `int64` | `integer` |
| `float32`, `float64` | `number` |
| `bool` | `boolean` |
| `[]T` | `array` |
| `time.Time` | `string / date-time` |
| `[]byte` | `string / byte` |
| `map[K]V` | `additionalProperties`가 있는 `object` |
| 이름 있는 구조체 | `$ref` → `#/components/schemas/이름` |

`validate` 태그의 제약 조건도 스펙에 반영됩니다:

| 태그 | 스키마 제약 |
|------|------------|
| `required` | 부모 `required` 배열에 추가 |
| `min=N` | `minLength` (문자열) 또는 `minimum` (숫자) |
| `max=N` | `maxLength` (문자열) 또는 `maximum` (숫자) |
| `email` | `format: email` |
| `url` | `format: uri` |
| `uuid` | `format: uuid` |
| `oneof=a\|b` | `enum: [a, b]` |

## 경로 파라미터

경로 파라미터는 라우트 경로에서 자동으로 추출됩니다:

```
/users/{id}  →  parameter: { name: "id", in: "path", required: true }
```

## 보안 스킴

`Config.Security`로 글로벌 보안 스킴 추가:

```go
openapi.Mount(app, openapi.Config{
    Title:   "My API",
    Version: "1.0.0",
    Security: []openapi.SecurityDef{
        openapi.BearerAuth(),              // Authorization: Bearer <token>
        openapi.APIKeyHeader("X-API-Key"), // 커스텀 헤더
    },
})
```

| 헬퍼 | 설명 |
|------|------|
| `openapi.BearerAuth()` | Bearer JWT 인증 |
| `openapi.APIKeyHeader(name)` | 커스텀 헤더 API 키 |
| `openapi.APIKeyCookie(name)` | 쿠키 API 키 |
| `openapi.OAuth2AuthCode(cfg)` | OAuth2 인가 코드 플로우 |

개별 라우트를 `.NoSecurity()`로 공개 표시:

```go
c.GET("/public", c.publicHandler).NoSecurity()
```

## 설정 옵션

| 필드 | 기본값 | 설명 |
|------|--------|------|
| `Title` | (필수) | Swagger UI에 표시할 API 이름 |
| `Version` | `"1.0.0"` | API 버전 문자열 |
| `Description` | `""` | Markdown 설명 |
| `SpecPath` | `"/openapi.json"` | JSON 스펙 엔드포인트 |
| `YAMLPath` | `"/openapi.yaml"` | YAML 스펙 엔드포인트 (`"-"`로 비활성화) |
| `DocsPath` | `"/docs"` | Swagger UI 엔드포인트 |
| `Security` | `nil` | 글로벌 보안 스킴 |
| `AutoErrorResponses` | `true` | 400/404/422/500 응답 자동 주입 |
| `TryItOutEnabled` | `false` | Swagger UI "Try it out" 자동 활성화 |
| `UseRFC9457` | `false` | 에러 응답에 ProblemDetail 스키마 사용 |

## 스펙 출력 예시

```json
{
  "openapi": "3.1.0",
  "info": { "title": "My API", "version": "1.0.0" },
  "paths": {
    "/users": {
      "get": {
        "summary": "사용자 목록 조회",
        "tags": ["users"],
        "responses": {
          "200": {
            "content": {
              "application/json": {
                "schema": { "type": "array", "items": { "$ref": "#/components/schemas/UserDTO" } }
              }
            }
          }
        }
      }
    }
  }
}
```
