# Error Handling

> 한국어: [에러 처리](./error-handling.ko.md)

## Typed Errors

Return typed errors from any handler to send a specific HTTP status code:

```go
core.ErrBadRequest("invalid input")       // 400
core.ErrUnauthorized("not authenticated") // 401
core.ErrForbidden("access denied")        // 403
core.ErrNotFound("user not found")        // 404
core.ErrConflict("email already exists")  // 409
core.ErrUnprocessable("validation failed") // 422
core.ErrInternal("something went wrong")  // 500
```

Default error response format:

```json
{ "code": 404, "message": "user not found" }
```

## Custom Error Handler

Override the default handler for all routes:

```go
app.SetErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
    var he *core.HTTPError
    if errors.As(err, &he) {
        // handle known errors
        core.JSON(w, he.Status, map[string]any{
            "error": he.Message,
            "path":  r.URL.Path,
        })
        return
    }
    // fall back to default for unknown errors
    core.DefaultErrorHandler(w, r, err)
})
```

## RFC 9457 Problem Details

Enable the `application/problem+json` format for all error responses:

```go
app.UseRFC9457()
```

Response format:

```json
{
  "type":     "about:blank",
  "title":    "Not Found",
  "status":   404,
  "detail":   "user not found",
  "instance": "/users/42"
}
```

This also updates the 404 / 405 handlers and panic recoverer to use the same format.

## Panic Recovery

Zenqo recovers from panics in all handlers by default and returns a 500 response. No configuration needed.

With RFC 9457 enabled, panics return a problem+json 500.

## Checking Error Types

```go
var he *core.HTTPError
if errors.As(err, &he) {
    fmt.Println(he.Status)  // e.g. 404
    fmt.Println(he.Message) // e.g. "user not found"
}
```
