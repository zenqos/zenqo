package core

import (
	"encoding/json"
	"net/http"
)

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
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, ErrBadRequest("invalid request body")
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
