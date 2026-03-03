package core

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

// MaxBodySize is the maximum allowed request body size for Bind.
// Default is 1 MB. Set to 0 to disable the limit.
var MaxBodySize int64 = 1 << 20

// Bind decodes the JSON request body into T.
// Returns ErrBadRequest automatically if the body is missing or malformed —
// no manual error handling needed in the handler.
//
// Example:
//
//	func (c *Controller) create(r *http.Request) (any, error) {
//		dto, err := core.Bind[CreateUserDTO](r)
//		if err != nil {
//			return nil, err  // already a 400 Bad Request
//		}
//		return c.svc.Create(dto), nil
//	}
func Bind[T any](r *http.Request) (T, error) {
	var v T
	ct := r.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(ct, "application/json") {
		return v, ErrBadRequest("Content-Type must be application/json")
	}
	body := io.Reader(r.Body)
	if MaxBodySize > 0 {
		body = io.LimitReader(r.Body, MaxBodySize)
	}
	if err := json.NewDecoder(body).Decode(&v); err != nil {
		return v, ErrBadRequest("invalid request body")
	}
	if err := validate(v); err != nil {
		return v, err
	}
	return v, nil
}

// BindQuery reads a named query parameter from the URL.
// Returns "" if the parameter is not present.
//
// Example:
//
//	page := core.BindQuery(r, "page")  // GET /users?page=2
func BindQuery(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

// BindHeader reads a named header from the request.
//
// Example:
//
//	token := core.BindHeader(r, "Authorization")
func BindHeader(r *http.Request, key string) string {
	return r.Header.Get(key)
}

// MaxUploadSize is the maximum allowed multipart form size for file uploads.
// Default is 32 MB. Override before calling BindFile / BindFiles if needed.
var MaxUploadSize int64 = 32 << 20

// UploadedFile holds metadata and content of a single uploaded file.
type UploadedFile struct {
	// Filename is the original file name reported by the client.
	Filename string
	// ContentType is the MIME type of the file (e.g. "image/jpeg").
	// Defaults to "application/octet-stream" when not provided by the client.
	ContentType string
	// Size is the byte length of Content.
	Size int64
	// Content is the raw file data. Available immediately; no need to close.
	Content []byte
	// Header exposes the raw multipart header for custom metadata access.
	Header *multipart.FileHeader
}

// BindFile parses a multipart/form-data request and returns the first file
// uploaded under the given form field name.
// Returns ErrBadRequest if the field is missing or the form cannot be parsed.
//
// Example:
//
//	func (c *Controller) upload(r *http.Request) (any, error) {
//		f, err := core.BindFile(r, "avatar")
//		if err != nil {
//			return nil, err
//		}
//		// f.Filename, f.ContentType, f.Size, f.Content
//		return map[string]any{"name": f.Filename, "size": f.Size}, nil
//	}
func BindFile(r *http.Request, field string) (*UploadedFile, error) {
	files, err := BindFiles(r, field)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, ErrBadRequest(fmt.Sprintf("no file uploaded for field %q", field))
	}
	return files[0], nil
}

// BindFiles parses a multipart/form-data request and returns all files
// uploaded under the given form field name.
// Returns a nil slice (and no error) when the field is absent.
//
// Example:
//
//	files, err := core.BindFiles(r, "images")
//	for _, f := range files {
//		// process f.Content
//	}
func BindFiles(r *http.Request, field string) ([]*UploadedFile, error) {
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		return nil, ErrBadRequest("failed to parse multipart form: " + err.Error())
	}
	fhs := r.MultipartForm.File[field]
	if len(fhs) == 0 {
		return nil, nil
	}
	result := make([]*UploadedFile, 0, len(fhs))
	for _, fh := range fhs {
		f, err := fh.Open()
		if err != nil {
			return nil, ErrInternal("failed to open uploaded file")
		}
		data, readErr := io.ReadAll(f)
		f.Close()
		if readErr != nil {
			return nil, ErrInternal("failed to read uploaded file")
		}
		ct := fh.Header.Get("Content-Type")
		if ct == "" {
			ct = "application/octet-stream"
		}
		result = append(result, &UploadedFile{
			Filename:    fh.Filename,
			ContentType: ct,
			Size:        fh.Size,
			Content:     data,
			Header:      fh,
		})
	}
	return result, nil
}

// Param extracts a named URL path parameter and converts it to the requested type.
// Supported types: string, int, int64, uint, uint64.
//
// Example:
//
//	id, err := core.Param[int64](r, "id")   // replaces strconv.ParseInt(core.URLParam(r, "id"), 10, 64)
//	name, err := core.Param[string](r, "name")
func Param[T interface{ string | int | int64 | uint | uint64 }](r *http.Request, key string) (T, error) {
	raw := URLParam(r, key)
	var zero T
	if raw == "" {
		return zero, ErrBadRequest(fmt.Sprintf("missing path parameter: %s", key))
	}
	switch any(zero).(type) {
	case string:
		return any(raw).(T), nil
	case int:
		v, err := strconv.Atoi(raw)
		if err != nil {
			return zero, ErrBadRequest(fmt.Sprintf("invalid path parameter %q: expected integer", key))
		}
		return any(v).(T), nil
	case int64:
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return zero, ErrBadRequest(fmt.Sprintf("invalid path parameter %q: expected integer", key))
		}
		return any(v).(T), nil
	case uint:
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return zero, ErrBadRequest(fmt.Sprintf("invalid path parameter %q: expected integer", key))
		}
		return any(uint(v)).(T), nil
	case uint64:
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return zero, ErrBadRequest(fmt.Sprintf("invalid path parameter %q: expected integer", key))
		}
		return any(v).(T), nil
	default:
		return zero, ErrBadRequest(fmt.Sprintf("unsupported parameter type for %q", key))
	}
}
