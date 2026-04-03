package errors

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents the JSON error response structure
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details for API responses
type ErrorDetail struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// ToResponse converts AppError to ErrorResponse
func (e *AppError) ToResponse() *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    e.Code,
			Message: e.Message,
			Details: e.Details,
		},
	}
}

// WriteHTTP writes the error as JSON HTTP response
func (e *AppError) WriteHTTP(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.HTTPStatus)
	_ = json.NewEncoder(w).Encode(e.ToResponse())
}

// WriteErrorResponse writes any error as HTTP response
func WriteErrorResponse(w http.ResponseWriter, err error) {
	appErr := GetAppError(err)
	if appErr == nil {
		// Unknown error - return generic internal error
		appErr = NewInternalError(err)
	}
	appErr.WriteHTTP(w)
}

// WriteJSON is an alias for WriteErrorResponse for convenience
func WriteJSON(w http.ResponseWriter, err error) {
	WriteErrorResponse(w, err)
}

// ErrorHandler is a middleware that recovers from panics and returns error response
func ErrorHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				appErr := ErrInternal.WithDetail("panic", err)
				appErr.WriteHTTP(w)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
