package core

import (
	"encoding/json"
	"fmt"
	"io"
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
