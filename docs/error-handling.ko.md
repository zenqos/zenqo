# 에러 처리

> English: [Error Handling](./error-handling.md)

## 타입 에러

핸들러에서 타입 에러를 반환하면 해당 HTTP 상태 코드로 응답합니다:

```go
core.ErrBadRequest("잘못된 입력")         // 400
core.ErrUnauthorized("인증되지 않음")      // 401
core.ErrForbidden("접근 거부")             // 403
core.ErrNotFound("사용자를 찾을 수 없음")  // 404
core.ErrConflict("이메일이 이미 존재함")   // 409
core.ErrUnprocessable("유효성 검사 실패") // 422
core.ErrInternal("서버 내부 오류")         // 500
```

기본 에러 응답 형식:

```json
{ "code": 404, "message": "사용자를 찾을 수 없음" }
```

## 커스텀 에러 핸들러

모든 라우트의 기본 핸들러 재정의:

```go
app.SetErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
    var he *core.HTTPError
    if errors.As(err, &he) {
        core.JSON(w, he.Status, map[string]any{
            "error": he.Message,
            "path":  r.URL.Path,
        })
        return
    }
    // 알 수 없는 에러는 기본 처리로 넘기기
    core.DefaultErrorHandler(w, r, err)
})
```

## RFC 9457 Problem Details

모든 에러 응답에 `application/problem+json` 형식 활성화:

```go
app.UseRFC9457()
```

응답 형식:

```json
{
  "type":     "about:blank",
  "title":    "Not Found",
  "status":   404,
  "detail":   "사용자를 찾을 수 없음",
  "instance": "/users/42"
}
```

404 / 405 핸들러와 패닉 복구도 동일한 형식으로 변경됩니다.

## 패닉 복구

Zenqo는 기본적으로 모든 핸들러의 패닉을 복구하고 500 응답을 반환합니다. 별도 설정 불필요.

RFC 9457 활성화 시, 패닉은 problem+json 500으로 응답합니다.

## 에러 타입 확인

```go
var he *core.HTTPError
if errors.As(err, &he) {
    fmt.Println(he.Status)  // 예: 404
    fmt.Println(he.Message) // 예: "사용자를 찾을 수 없음"
}
```
