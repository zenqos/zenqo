package core

import (
	"net/http"
	"strconv"

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

// ProblemDetail represents an RFC 9457 Problem Details object.
// When UseRFC9457 is enabled, all error responses use this format
// with Content-Type: application/problem+json.
//
// Reference: https://www.rfc-editor.org/rfc/rfc9457
type ProblemDetail struct {
	Type     string       `json:"type"`               // URI reference; default "about:blank"
	Title    string       `json:"title"`              // short human-readable summary
	Status   int          `json:"status"`             // HTTP status code
	Detail   string       `json:"detail,omitempty"`   // specific explanation
	Instance string       `json:"instance,omitempty"` // URI of the specific occurrence
	Errors   []FieldError `json:"errors,omitempty"`   // validation errors
}

// ProblemJSON writes a ProblemDetail as application/problem+json.
func ProblemJSON(w http.ResponseWriter, pd ProblemDetail) {
	if pd.Type == "" {
		pd.Type = "about:blank"
	}
	if pd.Title == "" {
		pd.Title = httpStatusTitle(pd.Status)
	}
	b, err := enc.Marshal(pd)
	if err != nil {
		zlog.Err("ProblemJSON", err.Error())
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"type":"about:blank","title":"Internal Server Error","status":500}`)) //nolint:errcheck
		return
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(pd.Status)
	w.Write(b) //nolint:errcheck
}

// httpStatusTitle returns the standard HTTP status title for common codes.
func httpStatusTitle(code int) string {
	switch code {
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 409:
		return "Conflict"
	case 422:
		return "Unprocessable Entity"
	case 429:
		return "Too Many Requests"
	case 500:
		return "Internal Server Error"
	case 502:
		return "Bad Gateway"
	case 503:
		return "Service Unavailable"
	default:
		return "HTTP " + strconv.Itoa(code)
	}
}
