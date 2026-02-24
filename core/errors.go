package core

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
