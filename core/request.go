package core

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"

	zlog "github.com/zenqos/zenqo/internal/log"
)

var (
	maxBodySize   atomic.Int64
	maxUploadSize atomic.Int64
	maxFileCount  atomic.Int32
	maxSingleFile atomic.Int64
)

func init() {
	maxBodySize.Store(1 << 20)
	maxUploadSize.Store(32 << 20)
	maxFileCount.Store(20)
	maxSingleFile.Store(10 << 20)
}

// SetMaxBodySize sets the maximum allowed request body size for Bind.
// Default is 1 MB. Set to 0 to disable the limit.
func SetMaxBodySize(n int64) { maxBodySize.Store(n) }

// SetMaxUploadSize sets the maximum allowed multipart form size for file uploads.
// Default is 32 MB.
func SetMaxUploadSize(n int64) { maxUploadSize.Store(n) }

// SetMaxFileCount sets the maximum number of files accepted in a single BindFiles call.
// Default is 20. Set to 0 to disable the limit.
func SetMaxFileCount(n int) { maxFileCount.Store(int32(n)) }

// SetMaxSingleFileSize sets the maximum allowed size for a single uploaded file.
// Default is 10 MB. Set to 0 to disable the limit.
func SetMaxSingleFileSize(n int64) { maxSingleFile.Store(n) }

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
	if ct != "" {
		// Parse the media type properly so that "application/json; charset=utf-8"
		// is accepted but "application/json-patch+json" or "application/jsonl" are
		// correctly rejected.
		mediaType, _, err := mime.ParseMediaType(ct)
		if err != nil || mediaType != "application/json" {
			return v, ErrBadRequest("Content-Type must be application/json")
		}
	}
	body := io.Reader(r.Body)
	if limit := maxBodySize.Load(); limit > 0 {
		body = io.LimitReader(r.Body, limit+1)
	}
	err := json.NewDecoder(body).Decode(&v)
	_, _ = io.Copy(io.Discard, r.Body)
	if err != nil {
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

// BindQueryStruct binds URL query parameters to a typed struct.
// Fields must be tagged with `query:"key"` to map query keys to struct fields.
// Supports string, int, int64, uint, uint64, float32, float64, bool, and []string.
// Validation tags (`validate:"..."`) are applied after binding.
//
// Example:
//
//	type ListQuery struct {
//		Page  int    `query:"page"`
//		Limit int    `query:"limit" validate:"max=100"`
//		Sort  string `query:"sort"  validate:"oneof=asc|desc"`
//		Tags  []string `query:"tag"`
//	}
//
//	q, err := core.BindQueryStruct[ListQuery](r)
func BindQueryStruct[T any](r *http.Request) (T, error) {
	var v T
	rv := reflect.ValueOf(&v).Elem()
	rt := rv.Type()
	query := r.URL.Query()

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if !sf.IsExported() {
			continue
		}
		key := sf.Tag.Get("query")
		if key == "" || key == "-" {
			continue
		}

		fv := rv.Field(i)

		// []string: collect all values for the key
		if sf.Type == reflect.TypeOf([]string{}) {
			if vals, ok := query[key]; ok {
				fv.Set(reflect.ValueOf(vals))
			}
			continue
		}

		raw := query.Get(key)
		if raw == "" {
			continue
		}

		if err := setFieldFromString(fv, raw, key); err != nil {
			return v, err
		}
	}

	if err := validate(v); err != nil {
		return v, err
	}
	return v, nil
}

// setFieldFromString converts a string value and sets it on the reflect.Value.
func setFieldFromString(fv reflect.Value, raw, key string) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return ErrBadRequest(fmt.Sprintf("invalid query parameter %q: expected integer", key))
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return ErrBadRequest(fmt.Sprintf("invalid query parameter %q: expected unsigned integer", key))
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return ErrBadRequest(fmt.Sprintf("invalid query parameter %q: expected number", key))
		}
		fv.SetFloat(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return ErrBadRequest(fmt.Sprintf("invalid query parameter %q: expected boolean", key))
		}
		fv.SetBool(b)
	}
	return nil
}

// BindHeader reads a named header from the request.
//
// Example:
//
//	token := core.BindHeader(r, "Authorization")
func BindHeader(r *http.Request, key string) string {
	return r.Header.Get(key)
}

// UploadedFile holds metadata and content of a single uploaded file.
type UploadedFile struct {
	// Filename is the base name of the uploaded file (directory components stripped).
	// The value is sanitized: path separators and null bytes are removed.
	// Do NOT use this value as a file system path without additional validation.
	Filename string
	// ContentType is the MIME type detected from the file content (first 512 bytes).
	// This is determined server-side and is NOT the value declared by the client.
	ContentType string
	// Size is the byte length of Content.
	Size int64
	// Content is the raw file data. Available immediately; no need to close.
	Content []byte
	// Header exposes the raw multipart header for custom metadata access,
	// including the original (unsanitized) client-declared filename and content type.
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
// Per-request limits (configurable via package-level vars):
//   - Total form size: MaxUploadSize (default 32 MB)
//   - Maximum files per request: MaxFileCount (default 20)
//   - Maximum size per file: MaxSingleFileSize (default 10 MB)
//
// Example:
//
//	files, err := core.BindFiles(r, "images")
//	for _, f := range files {
//		// process f.Content
//	}
func BindFiles(r *http.Request, field string) ([]*UploadedFile, error) {
	if err := r.ParseMultipartForm(maxUploadSize.Load()); err != nil {
		zlog.Err("BindFiles", err.Error())
		return nil, ErrBadRequest("invalid multipart form")
	}
	if r.MultipartForm == nil {
		return nil, nil
	}
	fhs := r.MultipartForm.File[field]
	if len(fhs) == 0 {
		return nil, nil
	}
	fileCount := int(maxFileCount.Load())
	if fileCount > 0 && len(fhs) > fileCount {
		return nil, ErrBadRequest(fmt.Sprintf("too many files: maximum %d files per request", fileCount))
	}
	singleLimit := maxSingleFile.Load()
	result := make([]*UploadedFile, 0, len(fhs))
	for _, fh := range fhs {
		if singleLimit > 0 && fh.Size > singleLimit {
			return nil, ErrBadRequest(fmt.Sprintf("file %q exceeds maximum size of %d bytes", fh.Filename, singleLimit))
		}
		f, err := fh.Open()
		if err != nil {
			return nil, ErrInternal("failed to open uploaded file")
		}
		data, readErr := io.ReadAll(f)
		f.Close()
		if readErr != nil {
			return nil, ErrInternal("failed to read uploaded file")
		}
		result = append(result, &UploadedFile{
			Filename:    sanitizeFilename(fh.Filename),
			ContentType: http.DetectContentType(data),
			Size:        int64(len(data)),
			Content:     data,
			Header:      fh,
		})
	}
	return result, nil
}

// sanitizeFilename strips directory components and null bytes from a client-provided filename.
func sanitizeFilename(name string) string {
	// Remove null bytes that could truncate paths on some systems.
	name = strings.ReplaceAll(name, "\x00", "")
	// filepath.Base strips any leading path components (e.g. "../../etc/passwd" → "passwd").
	return filepath.Base(name)
}

// Param extracts a named URL path parameter and converts it to the requested type.
// Supported types: string, int, int64, uint, uint64.
//
// Example:
//
//	id, err := core.Param[int64](r, "id")   // replaces strconv.ParseInt(core.URLParam(r, "id"), 10, 64)
//	name, err := core.Param[string](r, "name")
func Param[T interface {
	string | int | int64 | uint | uint64
}](r *http.Request, key string) (T, error) {
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

// MustParam extracts a named URL path parameter, converts it to the requested type,
// and writes a 400 Bad Request response if the parameter is missing or invalid.
// Returns the parsed value and true on success, zero value and false on failure.
//
// This is a convenience wrapper around Param that handles the error response
// automatically, avoiding repetitive error-checking boilerplate.
//
// Example:
//
//	func (c *Controller) get(w http.ResponseWriter, r *http.Request) {
//		id, ok := core.MustParam[int64](w, r, "id")
//		if !ok {
//			return  // 400 already written
//		}
//		// use id safely
//	}
func MustParam[T interface {
	string | int | int64 | uint | uint64
}](w http.ResponseWriter, r *http.Request, key string) (T, bool) {
	val, err := Param[T](r, key)
	if err != nil {
		BadRequest(w, err.Error())
		return val, false
	}
	return val, true
}
