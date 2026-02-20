package core

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

// SuccessResponse is the standard envelope for successful responses.
// Shape: { "success": true, "data": <payload> }
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

// ErrorResponse is the standard envelope for error responses.
// Shape: { "code": <status>, "message": "<reason>" }
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
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

// JSON encodes data into a buffer first, then writes the response.
// If encoding fails before headers are sent, a 500 is returned instead
// of a partial JSON body.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		log.Printf("[Zenqo] JSON encoding error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":500,"message":"internal server error"}`)) //nolint:errcheck
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	buf.WriteTo(w) //nolint:errcheck // write errors are non-actionable after WriteHeader
}

// OK sends a 200 response wrapped in SuccessResponse.
func OK(w http.ResponseWriter, data interface{}) { JSON(w, 200, SuccessResponse{true, data}) }

// Created sends a 201 response wrapped in SuccessResponse.
func Created(w http.ResponseWriter, data interface{}) { JSON(w, 201, SuccessResponse{true, data}) }

// Error sends an error response with the given status code and message.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, ErrorResponse{status, msg})
}

// BadRequest sends a 400 error response.
func BadRequest(w http.ResponseWriter, msg string) { Error(w, 400, msg) }

// NotFound sends a 404 error response.
func NotFound(w http.ResponseWriter, msg string) { Error(w, 404, msg) }

// InternalError sends a 500 error response.
func InternalError(w http.ResponseWriter, msg string) { Error(w, 500, msg) }

// Paginated sends a 200 response with list data and pagination metadata.
func Paginated(w http.ResponseWriter, data interface{}, total, page, perPage int) {
	JSON(w, 200, PaginatedResponse{true, data, PaginationMeta{total, page, perPage}})
}
