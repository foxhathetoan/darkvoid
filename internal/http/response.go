package httputil

import (
	"encoding/json"
	"net/http"
)

// RESTful API Response Guidelines:
// - Success (2xx): Return data directly at top-level
// - Error (4xx/5xx): Handled by pkg/errors with standard format

// WriteJSON writes JSON response with given status code
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log the error or handle it appropriately
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}

// MessageResponse represents a simple message response
// Used for operations that don't return data (e.g., delete, update without body)
type MessageResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// IDResponse represents a response containing a created resource ID
// Used after POST/creation operations
type IDResponse struct {
	ID string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// BulkOperationResponse represents response for bulk operations
type BulkOperationResponse struct {
	Total   int32  `json:"total" example:"100"`
	Success int32  `json:"success" example:"95"`
	Failed  int32  `json:"failed" example:"5"`
	Message string `json:"message,omitempty" example:"Bulk operation completed"`
}

// NewMessageResponse creates a simple message response
func NewMessageResponse(message string) MessageResponse {
	return MessageResponse{
		Message: message,
	}
}

// NewIDResponse creates an ID response
func NewIDResponse(id string) IDResponse {
	return IDResponse{
		ID: id,
	}
}

// NewBulkOperationResponse creates a bulk operation response
func NewBulkOperationResponse(total, success, failed int32, message string) BulkOperationResponse {
	return BulkOperationResponse{
		Total:   total,
		Success: success,
		Failed:  failed,
		Message: message,
	}
}
