package core

import (
	"net/http"

	enc "github.com/zenqos/zenqo/internal/encoding"
	zlog "github.com/zenqos/zenqo/internal/log"
)

// SuccessResponse is the standard envelope for successful responses.
// Shape: { "success": true, "data": <payload> }
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

// ErrorResponse is the standard envelope for all error responses.
// Shape: { "code": <status>, "message": "<reason>" }
// Validation errors include an "errors" array: { "code": 400, "message": "...", "errors": [...] }
type ErrorResponse struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Errors  []FieldError `json:"errors,omitempty"`
}

// PaginatedResponse wraps a list payload with pagination metadata.
// Shape: { "success": true, "data": [...], "meta": { "total", "page", "per_page" } }
type PaginatedResponse struct {
	Success bool           `json:"success"`
	Data    interface{}    `json:"data"`
	Meta    PaginationMeta `json:"meta"`
}

// PaginationMeta holds pagination information for list responses.
type PaginationMeta struct {
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

// JSON encodes data and writes the response.
// Uses Zenqo's custom encoder — struct tags are optional,
// PascalCase field names are automatically converted to camelCase.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	b, err := enc.Marshal(data)
	if err != nil {
		zlog.Err("JSON", err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":500,"message":"internal server error"}`)) //nolint:errcheck
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(b) //nolint:errcheck
}

// OK sends a 200 response wrapped in SuccessResponse.
func OK(w http.ResponseWriter, data interface{}) { JSON(w, 200, SuccessResponse{true, data}) }

// Created sends a 201 response wrapped in SuccessResponse.
func Created(w http.ResponseWriter, data interface{}) { JSON(w, 201, SuccessResponse{true, data}) }

// Error sends an error response with the given status code and message.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, ErrorResponse{Code: status, Message: msg})
}

// BadRequest sends a 400 error response.
func BadRequest(w http.ResponseWriter, msg string) { Error(w, 400, msg) }

// NotFound sends a 404 error response.
func NotFound(w http.ResponseWriter, msg string) { Error(w, 404, msg) }

// InternalError sends a 500 error response.
func InternalError(w http.ResponseWriter, msg string) { Error(w, 500, msg) }

// ValidationFailed sends a 400 response with per-field validation errors.
func ValidationFailed(w http.ResponseWriter, errs []FieldError) {
	JSON(w, 400, ErrorResponse{Code: 400, Message: "validation failed", Errors: errs})
}

// Paginated sends a 200 response with list data and pagination metadata.
func Paginated(w http.ResponseWriter, data interface{}, total, page, perPage int) {
	JSON(w, 200, PaginatedResponse{true, data, PaginationMeta{total, page, perPage}})
}
