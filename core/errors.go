package core

import (
	"errors"
	"fmt"
	"net/http"

	zlog "github.com/zenqos/zenqo/internal/log"
	"github.com/zenqos/zenqo/middleware"
)

// HTTPError carries an HTTP status code and message.
// Return it from a HandlerFunc to send a specific HTTP error response.
//
// Example:
//
//	return nil, core.ErrNotFound("user not found")   // → 404
//	return nil, core.ErrBadRequest("invalid body")   // → 400
type HTTPError struct {
	Status  int
	Message string
}

func (e *HTTPError) Error() string { return e.Message }

func ErrBadRequest(msg string) error   { return &HTTPError{400, msg} }
func ErrUnauthorized(msg string) error { return &HTTPError{401, msg} }
func ErrForbidden(msg string) error    { return &HTTPError{403, msg} }
func ErrNotFound(msg string) error     { return &HTTPError{404, msg} }
func ErrInternal(msg string) error     { return &HTTPError{500, msg} }

// FieldError describes a single validation failure on a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError carries per-field validation failures.
// Returned by Bind when struct validation fails.
type ValidationError struct {
	Errors []FieldError
}

func (e *ValidationError) Error() string { return "validation failed" }

// ErrValidation creates a ValidationError from one or more FieldErrors.
func ErrValidation(errs ...FieldError) error {
	return &ValidationError{Errors: errs}
}

// ErrorHandlerFunc handles errors returned by HandlerFuncs.
// Register a custom one with app.SetErrorHandler() to override the default behavior.
// Call DefaultErrorHandler(w, r, err) inside your implementation to fall back to Zenqo's built-in logic.
type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)

// DefaultErrorHandler is Zenqo's built-in error handler used by adapt().
//   - *ValidationError → 400 with per-field error list
//   - *HTTPError       → matching HTTP status
//   - everything else  → 500 with the request ID logged
func DefaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	var ve *ValidationError
	if errors.As(err, &ve) {
		ValidationFailed(w, ve.Errors)
		return
	}
	var he *HTTPError
	if errors.As(err, &he) {
		Error(w, he.Status, he.Message)
		return
	}
	reqID := middleware.GetReqID(r.Context())
	zlog.Err("Handler", fmt.Sprintf("[%s] %s %s — %v", reqID, r.Method, r.URL.Path, err))
	InternalError(w, "internal server error")
}

// RFC9457ErrorHandler is an error handler that produces RFC 9457 Problem Details responses.
// Enable it via app.UseRFC9457() or pass it to app.SetErrorHandler().
//
//   - *ValidationError → 400 with per-field errors in "errors" array
//   - *HTTPError       → matching HTTP status with detail
//   - everything else  → 500 with the request ID logged
func RFC9457ErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	var ve *ValidationError
	if errors.As(err, &ve) {
		ProblemJSON(w, ProblemDetail{
			Status:   400,
			Detail:   "validation failed",
			Instance: r.URL.Path,
			Errors:   ve.Errors,
		})
		return
	}
	var he *HTTPError
	if errors.As(err, &he) {
		ProblemJSON(w, ProblemDetail{
			Status:   he.Status,
			Detail:   he.Message,
			Instance: r.URL.Path,
		})
		return
	}
	reqID := middleware.GetReqID(r.Context())
	zlog.Err("Handler", fmt.Sprintf("[%s] %s %s — %v", reqID, r.Method, r.URL.Path, err))
	ProblemJSON(w, ProblemDetail{
		Status:   500,
		Instance: r.URL.Path,
	})
}
