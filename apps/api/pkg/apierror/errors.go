package apierror

import (
	"encoding/json"
	"fmt"
)

// APIError is the canonical error envelope returned by all EcoLLM API endpoints.
// Callers unmarshal by inspecting the "type" field.
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
	TraceID string `json:"trace_id,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("[%s] %s (code=%d)", e.Type, e.Message, e.Code)
}

// MarshalJSON satisfies json.Marshaler so Fiber's c.JSON() renders the right shape.
func (e *APIError) MarshalJSON() ([]byte, error) {
	type alias APIError
	return json.Marshal((*alias)(e))
}

// Sentinel errors — instantiate once, never mutate.
var (
	ErrUnauthorized    = &APIError{Code: 401, Message: "Invalid API key", Type: "auth_error"}
	ErrForbidden       = &APIError{Code: 403, Message: "Insufficient permissions", Type: "auth_error"}
	ErrRateLimited     = &APIError{Code: 429, Message: "Rate limit exceeded", Type: "rate_limit_error"}
	ErrInvalidRequest  = &APIError{Code: 400, Message: "Invalid request body", Type: "validation_error"}
	ErrModelUnavailable = &APIError{Code: 503, Message: "No model available", Type: "service_error"}
	ErrInferenceFailed = &APIError{Code: 502, Message: "Inference failed", Type: "inference_error"}
	ErrNotFound        = &APIError{Code: 404, Message: "Resource not found", Type: "not_found_error"}
	ErrInternal        = &APIError{Code: 500, Message: "Internal server error", Type: "internal_error"}
	ErrRequestTooLarge = &APIError{Code: 413, Message: "Request body exceeds 1MB limit", Type: "validation_error"}
)

// Validation wraps a field-level validation message in the standard envelope.
func Validation(field, detail string) *APIError {
	return &APIError{
		Code:    400,
		Message: fmt.Sprintf("%s: %s", field, detail),
		Type:    "validation_error",
	}
}
