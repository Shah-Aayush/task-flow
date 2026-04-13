package domain

import "errors"

// Sentinel domain errors returned by services.
// Handlers map these to HTTP status codes via response.ErrorToHTTP().
// Never use err.Error() string comparison — always use errors.Is().
var (
	ErrNotFound     = errors.New("not found")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrUnauthorized = errors.New("unauthorized")
)

// ValidationError carries structured field-level validation failures.
// Returned by the validator layer and mapped to a 400 response with
// {"error": "validation failed", "fields": {...}} body.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "validation failed"
}

// NewValidationError creates a ValidationError with the given field errors.
func NewValidationError(fields map[string]string) *ValidationError {
	return &ValidationError{Fields: fields}
}
