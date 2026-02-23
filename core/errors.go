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
