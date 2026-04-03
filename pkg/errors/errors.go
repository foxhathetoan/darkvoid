package errors

import (
	"errors"
	"fmt"
)

// AppError represents a structured application error
type AppError struct {
	Code       string         // Error code (e.g., "USER_NOT_FOUND")
	Message    string         // Human-readable message
	HTTPStatus int            // HTTP status code
	Details    map[string]any // Additional context
	Err        error          // Underlying error
}

// Error implements error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithDetail adds a detail to the error
func (e *AppError) WithDetail(key string, value any) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// New creates a new AppError
func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Details:    make(map[string]any),
	}
}

// Wrap wraps an existing error with context
func Wrap(err error, code, message string, httpStatus int) *AppError {
	if err == nil {
		return nil
	}

	// If already an AppError, preserve it
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Details:    make(map[string]any),
		Err:        err,
	}
}

// Is checks if error matches the target
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target
func As(err error, target any) bool {
	return errors.As(err, target)
}

// GetAppError extracts AppError from error chain
func GetAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// Standard library error functions
var (
	// Join returns an error that wraps the given errors
	Join = errors.Join

	// Unwrap returns the result of calling the Unwrap method on err
	Unwrap = errors.Unwrap
)
