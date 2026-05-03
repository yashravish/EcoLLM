package inference

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ecollm/api/internal/router"
)

// mockVLLMResponse returns a minimal valid vLLM chat completion response.
func mockVLLMResponse(content string) vLLMResponse {
	return vLLMResponse{
		ID:     "test-id",
		Object: "chat.completion",
		Choices: []vLLMChoice{
			{
				Index:        0,
				Message:      vLLMMessage{Role: "assistant", Content: content},
				FinishReason: "stop",
			},
		},
		Usage: vLLMUsage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
	}
}

// serveVLLM returns an httptest.Server that responds to /v1/chat/completions
// with the given response body and status code.
// The 2ms delay ensures LatencyMs > 0 in TestGateway_LatencyMeasured.
func serveVLLM(t *testing.T, status int, body vLLMResponse) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		time.Sleep(2 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
}

// serveHang returns an httptest.Server that never responds (simulates a hung vLLM)
// and a cancel function. Calling cancel unblocks all in-flight handlers so
// Close() can drain without deadlocking. The http.Client timeout does not
// reliably cancel server-side request contexts in tests (the TCP connection
// may stay open in the transport pool), so we need an explicit out.
func serveHang(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-ctx.Done():
		case <-r.Context().Done():
		}
	}))
	return srv, cancel
}

func newTestGateway(t *testing.T, endpoints InferenceEndpoints) *Gateway {
	t.Helper()
	return NewGateway(endpoints, 500*time.Millisecond)
}

func TestGateway_SuccessfulInference(t *testing.T) {
	server := serveVLLM(t, http.StatusOK, mockVLLMResponse("Paris"))
	defer server.Close()

	gw := newTestGateway(t, InferenceEndpoints{Phi3URL: server.URL})
	decision := router.RouteDecision{Model: "phi_3", Fallback: "mistral_7b"}
	msgs := []InferenceMessage{{Role: "user", Content: "What is the capital of France?"}}

	result, err := gw.InferWithFallback(context.Background(), decision, msgs, 256, 0.7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Paris" {
		t.Errorf("Text = %q, want %q", result.Text, "Paris")
	}
	if result.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", result.FinishReason, "stop")
	}
	if result.PromptTokens != 10 || result.CompletionTokens != 20 {
		t.Errorf("tokens = %d/%d, want 10/20", result.PromptTokens, result.CompletionTokens)
	}
	if result.UsedFallback {
		t.Error("UsedFallback should be false on primary success")
	}
}

func TestGateway_TimeoutTriggersFallback(t *testing.T) {
	primary, cancelHang := serveHang(t)
	defer func() {
		cancelHang()    // unblock the handler first so Close doesn't deadlock
		primary.Close()
	}()

	fallback := serveVLLM(t, http.StatusOK, mockVLLMResponse("fallback answer"))
	defer fallback.Close()

	gw := newTestGateway(t, InferenceEndpoints{
		Phi3URL:    primary.URL,
		MistralURL: fallback.URL,
	})
	decision := router.RouteDecision{Model: "phi_3", Fallback: "mistral_7b"}
	msgs := []InferenceMessage{{Role: "user", Content: "test"}}

	result, err := gw.InferWithFallback(context.Background(), decision, msgs, 256, 0.7)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}
	if !result.UsedFallback {
		t.Error("UsedFallback should be true when primary timed out")
	}
	if result.Text != "fallback answer" {
		t.Errorf("Text = %q, want %q", result.Text, "fallback answer")
	}
}

func TestGateway_FallbackSuccess_MarksUsedFallback(t *testing.T) {
	// Primary returns HTTP 500; fallback returns 200.
	primary := serveVLLM(t, http.StatusInternalServerError, vLLMResponse{})
	defer primary.Close()

	fallback := serveVLLM(t, http.StatusOK, mockVLLMResponse("recovered"))
	defer fallback.Close()

	gw := newTestGateway(t, InferenceEndpoints{
		Phi3URL:    primary.URL,
		MistralURL: fallback.URL,
	})
	decision := router.RouteDecision{Model: "phi_3", Fallback: "mistral_7b"}
	msgs := []InferenceMessage{{Role: "user", Content: "test"}}

	result, err := gw.InferWithFallback(context.Background(), decision, msgs, 100, 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.UsedFallback {
		t.Error("UsedFallback should be true when primary returned 500")
	}
}

func TestGateway_BothModelsFail_ReturnsError(t *testing.T) {
	primary := serveVLLM(t, http.StatusServiceUnavailable, vLLMResponse{})
	defer primary.Close()

	fallback := serveVLLM(t, http.StatusServiceUnavailable, vLLMResponse{})
	defer fallback.Close()

	gw := newTestGateway(t, InferenceEndpoints{
		Phi3URL:    primary.URL,
		MistralURL: fallback.URL,
	})
	decision := router.RouteDecision{Model: "phi_3", Fallback: "mistral_7b"}
	msgs := []InferenceMessage{{Role: "user", Content: "test"}}

	_, err := gw.InferWithFallback(context.Background(), decision, msgs, 100, 0.5)
	if err == nil {
		t.Error("expected error when both primary and fallback fail")
	}
	if !strings.Contains(err.Error(), "fallback") {
		t.Errorf("error should mention fallback model, got: %v", err)
	}
}

func TestGateway_UnknownModel_ImmediateError(t *testing.T) {
	gw := NewGateway(InferenceEndpoints{}, 5*time.Second)
	decision := router.RouteDecision{Model: "unknown_model_xyz", Fallback: ""}
	msgs := []InferenceMessage{{Role: "user", Content: "test"}}

	_, err := gw.InferWithFallback(context.Background(), decision, msgs, 100, 0.5)
	if err == nil {
		t.Error("expected error for unknown model, got nil")
	}
	// Error must come immediately — no network call should be attempted.
}

func TestGateway_NoFallbackAvailable_ReturnsError(t *testing.T) {
	// llama_70b has no fallback in the chain.
	primary := serveVLLM(t, http.StatusInternalServerError, vLLMResponse{})
	defer primary.Close()

	gw := newTestGateway(t, InferenceEndpoints{Llama70BURL: primary.URL})
	decision := router.RouteDecision{Model: "llama_70b", Fallback: ""}
	msgs := []InferenceMessage{{Role: "user", Content: "test"}}

	_, err := gw.InferWithFallback(context.Background(), decision, msgs, 100, 0.5)
	if err == nil {
		t.Error("expected error when primary fails and no fallback exists")
	}
}

func TestGateway_LatencyMeasured(t *testing.T) {
	server := serveVLLM(t, http.StatusOK, mockVLLMResponse("ok"))
	defer server.Close()

	gw := newTestGateway(t, InferenceEndpoints{Phi3URL: server.URL})
	decision := router.RouteDecision{Model: "phi_3", Fallback: ""}
	msgs := []InferenceMessage{{Role: "user", Content: "test"}}

	result, err := gw.InferWithFallback(context.Background(), decision, msgs, 100, 0.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.LatencyMs <= 0 {
		t.Errorf("LatencyMs should be positive, got %d", result.LatencyMs)
	}
}
