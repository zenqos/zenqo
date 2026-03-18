# 요청 바인딩

> English: [Request Binding](./request-binding.md)

## JSON 바디

`Bind[T]`는 요청 바디를 구조체로 디코딩하고 유효성 검사를 자동으로 실행합니다.

```go
type CreateUserDTO struct {
    Name  string `validate:"required,min=2,max=50"`
    Email string `validate:"required,email"`
    Age   int    `validate:"min=0,max=150"`
    Role  string `validate:"oneof=admin user guest"`
}

func (c *UserController) create(r *http.Request) (any, error) {
    dto, err := core.Bind[CreateUserDTO](r)
    if err \!= nil {
        return nil, err // 이미 400 Bad Request
    }
    return c.svc.Create(dto), nil
}
```

다음의 경우 자동으로 `ErrBadRequest`를 반환합니다:
- `Content-Type`이 `application/json`이 아닌 경우
- 바디가 없거나 잘못된 JSON인 경우
- 유효성 검사 실패

**바디 크기 제한** (기본 1 MB):

```go
core.SetMaxBodySize(5 << 20) // 5 MB; 0으로 설정하면 제한 없음
```

## 유효성 검사 규칙

| 태그 | 설명 |
|------|------|
| `required` | 필드가 존재하고 비어있지 않아야 함 |
| `min=N` | 최소 길이(문자열), 최솟값(숫자), 또는 최소 항목 수(슬라이스/맵) |
| `max=N` | 최대 길이(문자열), 최댓값(숫자), 또는 최대 항목 수(슬라이스/맵) |
| `len=N` | 정확한 길이 (문자열, 슬라이스, 배열, 맵) |
| `email` | 유효한 이메일 형식 |
| `url` | 유효한 HTTP/HTTPS URL |
| `uuid` | 유효한 UUID 형식 |
| `oneof=a\|b\|c` | 나열된 값 중 하나여야 함 (문자열 또는 정수) |
| `alpha` | 영문자만 |
| `alphanum` | 영문자와 숫자만 |
| `numeric` | 숫자만 |
| `lowercase` | 소문자여야 함 |
| `uppercase` | 대문자여야 함 |
| `contains=sub` | 부분 문자열 포함 |
| `startswith=pre` | 접두사로 시작 |
| `endswith=suf` | 접미사로 끝남 |
| `regex=pattern` | 정규식 패턴 매칭 |
| `dive` | 슬라이스/배열 각 요소 검증 |

중첩 구조체는 점(.) 구분 필드 경로로 재귀 검증됩니다 (예: `address.city`).

## 경로 파라미터

```go
// 지원 타입: string, int, int64, uint, uint64
id, err := core.Param[int64](r, "id")      // /users/{id}
name, err := core.Param[string](r, "slug")  // /posts/{slug}
```

파라미터가 없거나 요청한 타입으로 변환할 수 없으면 `ErrBadRequest`를 반환합니다.

## 쿼리 파라미터

단일 값:

```go
page := core.BindQuery(r, "page")   // GET /users?page=2  →  "2"
q    := core.BindQuery(r, "q")      // 없으면 "" 반환
```

구조체 바인딩 (유효성 검사 포함):

```go
type ListQuery struct {
    Page  int      `query:"page"`
    Limit int      `query:"limit" validate:"max=100"`
    Sort  string   `query:"sort"  validate:"oneof=asc|desc"`
    Tags  []string `query:"tag"`
}

func (c *Controller) list(r *http.Request) (any, error) {
    q, err := core.BindQueryStruct[ListQuery](r)
    if err != nil {
        return nil, err
    }
    // q.Page, q.Limit, q.Sort, q.Tags가 타입 변환 및 검증 완료
}
```

지원 타입: `string`, `int`, `int64`, `uint`, `uint64`, `float32`, `float64`, `bool`, `[]string`.

## 헤더

```go
token  := core.BindHeader(r, "Authorization")
accept := core.BindHeader(r, "Accept")
```

## 파일 업로드

### 단일 파일

```go
func (c *Controller) upload(r *http.Request) (any, error) {
    f, err := core.BindFile(r, "avatar")
    if err \!= nil {
        return nil, err
    }
    // f.Filename    — 정제된 파일명 (경로 구성요소 제거됨)
    // f.ContentType — 서버에서 감지한 MIME 타입 (클라이언트 선언값 아님)
    // f.Size        — 바이트 크기
    // f.Content     — []byte 파일 데이터
    // f.Header      — 커스텀 메타데이터 접근용 원본 multipart.FileHeader
    return map[string]any{"name": f.Filename, "size": f.Size}, nil
}
```

### 여러 파일

```go
files, err := core.BindFiles(r, "images")
if err \!= nil {
    return nil, err
}
for _, f := range files {
    // f.Content 처리
}
```

**업로드 제한:**

```go
core.SetMaxUploadSize(64 << 20)    // 전체 폼 크기 (기본 32 MB)
core.SetMaxFileCount(10)           // 요청당 최대 파일 수 (기본 20)
core.SetMaxSingleFileSize(5 << 20) // 파일당 최대 크기 (기본 10 MB)
```

> **보안 주의:** `Filename`은 서버에서 정제됩니다 (경로 구분자, null 바이트 제거). `ContentType`은 클라이언트 선언값이 아닌 파일 내용으로 감지됩니다. `Filename`을 파일 시스템 경로로 직접 사용하지 마세요.
