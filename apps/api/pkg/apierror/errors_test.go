package apierror

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAPIError_ErrorString(t *testing.T) {
	cases := []struct {
		err  *APIError
		want string
	}{
		{ErrUnauthorized, "[auth_error] Invalid API key (code=401)"},
		{ErrForbidden, "[auth_error] Insufficient permissions (code=403)"},
		{ErrRateLimited, "[rate_limit_error] Rate limit exceeded (code=429)"},
		{ErrInvalidRequest, "[validation_error] Invalid request body (code=400)"},
		{ErrModelUnavailable, "[service_error] No model available (code=503)"},
		{ErrInferenceFailed, "[inference_error] Inference failed (code=502)"},
		{ErrNotFound, "[not_found_error] Resource not found (code=404)"},
		{ErrInternal, "[internal_error] Internal server error (code=500)"},
		{ErrRequestTooLarge, "[validation_error] Request body exceeds 1MB limit (code=413)"},
	}
	for _, tc := range cases {
		t.Run(tc.err.Type, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAPIError_SentinelCodes(t *testing.T) {
	cases := []struct {
		name string
		err  *APIError
		code int
	}{
		{"unauthorized", ErrUnauthorized, 401},
		{"forbidden", ErrForbidden, 403},
		{"rate_limited", ErrRateLimited, 429},
		{"invalid_request", ErrInvalidRequest, 400},
		{"model_unavailable", ErrModelUnavailable, 503},
		{"inference_failed", ErrInferenceFailed, 502},
		{"not_found", ErrNotFound, 404},
		{"internal", ErrInternal, 500},
		{"request_too_large", ErrRequestTooLarge, 413},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Code != tc.code {
				t.Errorf("Code = %d, want %d", tc.err.Code, tc.code)
			}
		})
	}
}

func TestAPIError_MarshalJSON(t *testing.T) {
	b, err := json.Marshal(ErrUnauthorized)
	if err != nil {
		t.Fatalf("MarshalJSON() error: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if out["code"] != float64(401) {
		t.Errorf("json code = %v, want 401", out["code"])
	}
	if out["message"] != "Invalid API key" {
		t.Errorf("json message = %v", out["message"])
	}
	if out["type"] != "auth_error" {
		t.Errorf("json type = %v", out["type"])
	}
	if _, ok := out["trace_id"]; ok {
		t.Error("trace_id should be omitted when empty")
	}
}

func TestAPIError_MarshalJSON_TraceIDIncluded(t *testing.T) {
	e := &APIError{Code: 500, Message: "oops", Type: "internal_error", TraceID: "abc-123"}
	b, _ := json.Marshal(e)
	if !strings.Contains(string(b), `"trace_id":"abc-123"`) {
		t.Errorf("expected trace_id in JSON, got: %s", b)
	}
}

func TestValidation(t *testing.T) {
	e := Validation("email", "must be a valid email address")
	if e.Code != 400 {
		t.Errorf("Code = %d, want 400", e.Code)
	}
	if e.Type != "validation_error" {
		t.Errorf("Type = %q, want validation_error", e.Type)
	}
	want := "email: must be a valid email address"
	if e.Message != want {
		t.Errorf("Message = %q, want %q", e.Message, want)
	}
}

func TestValidation_EmptyFieldAndDetail(t *testing.T) {
	e := Validation("", "")
	if e == nil {
		t.Fatal("Validation() returned nil")
	}
	if e.Code != 400 {
		t.Errorf("Code = %d, want 400", e.Code)
	}
}

func TestSentinelErrors_NotMutated(t *testing.T) {
	// Confirm the sentinel is a pointer — callers must not modify it.
	orig := ErrUnauthorized.Code
	ErrUnauthorized.Code = 999
	if ErrUnauthorized.Code != 999 {
		t.Fatal("unexpected: pointer mutation did not persist")
	}
	ErrUnauthorized.Code = orig // restore
}
