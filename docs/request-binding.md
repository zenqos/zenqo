# Request Binding

> 한국어: [요청 바인딩](./request-binding.ko.md)

## JSON Body

`Bind[T]` decodes the request body into a struct and automatically runs validation.

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
        return nil, err // already a 400 Bad Request
    }
    return c.svc.Create(dto), nil
}
```

Returns `ErrBadRequest` automatically if:
- `Content-Type` is not `application/json`
- Body is missing or malformed JSON
- Validation fails

**Body size limit** (default 1 MB):

```go
core.MaxBodySize = 5 << 20 // 5 MB; set 0 to disable
```

## Validation Rules

| Tag | Description |
|-----|-------------|
| `required` | Field must be present and non-zero |
| `min=N` | Minimum length (string) or value (number) |
| `max=N` | Maximum length (string) or value (number) |
| `email` | Valid email format |
| `url` | Valid URL format |
| `uuid` | Valid UUID format |
| `oneof=a\|b\|c` | Value must be one of the listed options |

## Path Parameters

```go
// Supported types: string, int, int64, uint, uint64
id, err := core.Param[int64](r, "id")     // /users/{id}
name, err := core.Param[string](r, "slug") // /posts/{slug}
```

Returns `ErrBadRequest` if the parameter is missing or cannot be converted to the requested type.

## Query Parameters

```go
page := core.BindQuery(r, "page")   // GET /users?page=2  →  "2"
q    := core.BindQuery(r, "q")      // returns "" if absent
```

## Headers

```go
token  := core.BindHeader(r, "Authorization")
accept := core.BindHeader(r, "Accept")
```

## File Uploads

### Single File

```go
func (c *Controller) upload(r *http.Request) (any, error) {
    f, err := core.BindFile(r, "avatar")
    if err \!= nil {
        return nil, err
    }
    // f.Filename    — sanitized base name (path components stripped)
    // f.ContentType — server-detected MIME type (not client-declared)
    // f.Size        — byte length
    // f.Content     — []byte file data
    // f.Header      — raw multipart.FileHeader for custom metadata
    return map[string]any{"name": f.Filename, "size": f.Size}, nil
}
```

### Multiple Files

```go
files, err := core.BindFiles(r, "images")
if err \!= nil {
    return nil, err
}
for _, f := range files {
    // process f.Content
}
```

**Upload size limit** (default 32 MB):

```go
core.MaxUploadSize = 64 << 20 // 64 MB
```

> **Security note:** `Filename` is sanitized server-side (path separators and null bytes removed). `ContentType` is detected from file content, not the client's declared value. Do not use `Filename` as a filesystem path without additional validation.
