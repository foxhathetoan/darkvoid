package errors

import "net/http"

// Common HTTP error codes

// Generic errors
var (
	ErrInternal        = New("INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
	ErrBadRequest      = New("BAD_REQUEST", "Bad request", http.StatusBadRequest)
	ErrUnauthorized    = New("UNAUTHORIZED", "Unauthorized", http.StatusUnauthorized)
	ErrForbidden       = New("FORBIDDEN", "Forbidden", http.StatusForbidden)
	ErrNotFound        = New("NOT_FOUND", "Resource not found", http.StatusNotFound)
	ErrConflict        = New("CONFLICT", "Resource conflict", http.StatusConflict)
	ErrTooManyRequests = New("TOO_MANY_REQUESTS", "Too many requests", http.StatusTooManyRequests)
)

// Validation errors
var (
	ErrValidation    = New("VALIDATION_ERROR", "Validation failed", http.StatusBadRequest)
	ErrInvalidInput  = New("INVALID_INPUT", "Invalid input", http.StatusBadRequest)
	ErrMissingField  = New("MISSING_FIELD", "Required field missing", http.StatusBadRequest)
	ErrInvalidFormat = New("INVALID_FORMAT", "Invalid format", http.StatusBadRequest)
)

// Helper functions to create common errors

// NewValidationError creates a validation error with field details
func NewValidationError(field, reason string) *AppError {
	err := New("VALIDATION_ERROR", "Validation failed", http.StatusBadRequest)
	return err.WithDetail("field", field).WithDetail("reason", reason)
}

// NewNotFoundError creates a not found error for a resource
func NewNotFoundError(resource string) *AppError {
	err := New("NOT_FOUND", "Resource not found", http.StatusNotFound)
	return err.WithDetail("resource", resource)
}

// NewConflictError creates a conflict error
func NewConflictError(resource string) *AppError {
	err := New("CONFLICT", "Resource conflict", http.StatusConflict)
	return err.WithDetail("resource", resource)
}

// NewUnauthorizedError creates an unauthorized error with reason
func NewUnauthorizedError(reason string) *AppError {
	err := New("UNAUTHORIZED", "Unauthorized", http.StatusUnauthorized)
	if reason != "" {
		return err.WithDetail("reason", reason)
	}
	return err
}

// NewForbiddenError creates a forbidden error with reason
func NewForbiddenError(reason string) *AppError {
	err := New("FORBIDDEN", "Forbidden", http.StatusForbidden)
	if reason != "" {
		return err.WithDetail("reason", reason)
	}
	return err
}

// NewInternalError wraps an internal error
func NewInternalError(err error) *AppError {
	return Wrap(err, "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
}

// NewBadRequestError creates a bad request error with custom message
func NewBadRequestError(message string) *AppError {
	return New("BAD_REQUEST", message, http.StatusBadRequest)
}
