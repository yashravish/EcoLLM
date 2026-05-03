package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ecollm/api/internal/carbon"
	"github.com/ecollm/api/internal/inference"
	"github.com/ecollm/api/internal/prompt"
	"github.com/ecollm/api/internal/router"
)

// ── Test doubles ─────────────────────────────────────────────────────────────

// nopRepository satisfies requestRepository without a real database.
// Insert is a no-op; any error can be injected via InsertErr.
type nopRepository struct {
	inserted []*RequestRecord
	InsertErr error
}

func (r *nopRepository) Insert(_ context.Context, rec *RequestRecord) error {
	r.inserted = append(r.inserted, rec)
	return r.InsertErr
}

type mockVLLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mockVLLMChoice struct {
	Index        int              `json:"index"`
	Message      mockVLLMMessage  `json:"message"`
	FinishReason string           `json:"finish_reason"`
}

type mockVLLMUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type mockVLLMResponse struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Choices []mockVLLMChoice `json:"choices"`
	Usage   mockVLLMUsage    `json:"usage"`
}

// vllmResponse returns a minimal valid OpenAI-compatible completion response.
func vllmResponse(content string) mockVLLMResponse {
	return mockVLLMResponse{
		ID:     "test-id",
		Object: "chat.completion",
		Choices: []mockVLLMChoice{
			{
				Index:        0,
				Message:      mockVLLMMessage{Role: "assistant", Content: content},
				FinishReason: "stop",
			},
		},
		Usage: mockVLLMUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}
}

// serveMock returns an httptest.Server that responds to vLLM /v1/chat/completions.
// The 2ms delay ensures latencyMs > 0 so energy calculations produce non-zero values.
func serveMock(t *testing.T, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(vllmResponse(content)); err != nil {
			t.Errorf("encode mock response: %v", err)
		}
	}))
}

func errMock(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
}

// buildService assembles a Service with controllable inference endpoint(s).
func buildService(t *testing.T, endpoints inference.InferenceEndpoints, repo requestRepository) *Service {
	t.Helper()
	gw := inference.NewGateway(endpoints, 5*time.Second)
	return NewService(
		prompt.NewOptimizer(""),
		router.NewClassifier(),
		router.NewSelector(nil),
		gw,
		carbon.NewEstimator("US-EAST"),
		repo,
		nil,                               // energyRepo — nil-guarded in service
		nil,                               // carbonRepo — nil-guarded in service
		nil,                               // redisClient — nil-guarded in service
		ServiceConfig{EnableFallback: true},
		nil,                               // db — nil-guarded in service
	)
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestService_HappyPath_ReturnsFullMetadata(t *testing.T) {
	srv := serveMock(t, "Quicksort is a divide-and-conquer sorting algorithm.")
	defer srv.Close()

	repo := &nopRepository{}
	svc := buildService(t, inference.InferenceEndpoints{
		Phi3URL:    srv.URL,
		MistralURL: srv.URL,
	}, repo)

	req := &CompletionRequest{
		Messages:    []ChatMessage{{Role: "user", Content: "What is quicksort?"}},
		MaxTokens:   256,
		Temperature: 0.7,
		EcoLLM:      &EcoLLMOptions{IncludeMetadata: true},
	}

	resp, err := svc.Complete(context.Background(), "org-1", "req-1", req)
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}

	if resp.ID == "" {
		t.Error("response ID should not be empty")
	}
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		t.Error("response should have a non-empty choice")
	}
	if resp.EcoLLM == nil {
		t.Fatal("EcoLLM metadata should be present")
	}
	if resp.EcoLLM.Route.ModelSelected == "" {
		t.Error("route.model_selected should not be empty")
	}
	if resp.EcoLLM.Route.TaskType == "" {
		t.Error("route.task_type should not be empty")
	}
	if resp.EcoLLM.Energy.TotalEnergyKwh <= 0 {
		t.Errorf("energy.total_energy_kwh should be positive, got %g", resp.EcoLLM.Energy.TotalEnergyKwh)
	}
	if resp.EcoLLM.Cost.TotalCostUSD <= 0 {
		t.Errorf("cost.total_cost_usd should be positive, got %g", resp.EcoLLM.Cost.TotalCostUSD)
	}
	if resp.EcoLLM.Performance.LatencyMs <= 0 {
		t.Errorf("performance.latency_ms should be positive, got %d", resp.EcoLLM.Performance.LatencyMs)
	}
	// Let async persist goroutine finish before checking.
	time.Sleep(50 * time.Millisecond)
}

func TestService_SimpleTask_RoutesToSmallModel(t *testing.T) {
	srv := serveMock(t, "42")
	defer srv.Close()

	svc := buildService(t, inference.InferenceEndpoints{
		Phi3URL:    srv.URL,
		MistralURL: srv.URL,
	}, &nopRepository{})

	req := &CompletionRequest{
		Messages:  []ChatMessage{{Role: "user", Content: "What is 2+2?"}},
		MaxTokens: 32,
	}

	resp, err := svc.Complete(context.Background(), "org-1", "req-2", req)
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}

	model := resp.EcoLLM.Route.ModelSelected
	if model != "phi_3" && model != "mistral_7b" {
		t.Errorf("simple task should route to phi_3 or mistral_7b, got %q", model)
	}
	// Must NOT route to llama_13b or llama_70b for such a trivial query.
	if model == "llama_13b" || model == "llama_70b" {
		t.Errorf("simple task must not route to large model, got %q", model)
	}
}

func TestService_FallbackPath_SetsUsedFallback(t *testing.T) {
	// Primary (phi_3) returns 500; fallback (mistral_7b) returns 200.
	primaryFail := errMock(t)
	defer primaryFail.Close()

	fallbackOK := serveMock(t, "recovered")
	defer fallbackOK.Close()

	// "What is X?" → TaskSimple → phi_3 (primary) + mistral_7b (fallback)
	svc := buildService(t, inference.InferenceEndpoints{
		Phi3URL:    primaryFail.URL,
		MistralURL: fallbackOK.URL,
	}, &nopRepository{})

	req := &CompletionRequest{
		Messages:  []ChatMessage{{Role: "user", Content: "What is TCP?"}},
		MaxTokens: 128,
	}

	resp, err := svc.Complete(context.Background(), "org-1", "req-3", req)
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}
	if !resp.EcoLLM.Route.UsedFallback {
		t.Error("UsedFallback should be true when primary model failed")
	}
}

func TestService_Validation_EmptyMessages(t *testing.T) {
	svc := buildService(t, inference.InferenceEndpoints{}, &nopRepository{})
	req := &CompletionRequest{Messages: []ChatMessage{}}

	_, err := svc.Complete(context.Background(), "org-1", "req-4", req)
	// Service should degrade gracefully: extractUserPrompt returns "", which is a valid
	// (empty) string — inference will fail at the gateway level.
	// The test verifies that at minimum the service does not panic.
	_ = err // error or nil both acceptable; panic is the bug we guard against.
}

func TestService_EnergyMetadata_NonZero(t *testing.T) {
	srv := serveMock(t, "hello")
	defer srv.Close()

	svc := buildService(t, inference.InferenceEndpoints{
		Phi3URL:    srv.URL,
		MistralURL: srv.URL,
		Llama13BURL: srv.URL,
		Llama70BURL: srv.URL,
	}, &nopRepository{})

	req := &CompletionRequest{
		Messages:  []ChatMessage{{Role: "user", Content: "What is Redis?"}},
		MaxTokens: 64,
	}

	resp, err := svc.Complete(context.Background(), "org-1", "req-5", req)
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}
	if resp.EcoLLM.Energy.TotalEnergyKwh == 0 {
		t.Error("TotalEnergyKwh must be non-zero")
	}
	if resp.EcoLLM.Energy.CO2eGrams == 0 {
		t.Error("CO2eGrams must be non-zero")
	}
	if resp.EcoLLM.Energy.GridRegion == "" {
		t.Error("GridRegion must not be empty")
	}
}

func TestService_SavingsVsGPT4_Reasonable(t *testing.T) {
	srv := serveMock(t, "answer")
	defer srv.Close()

	svc := buildService(t, inference.InferenceEndpoints{
		Phi3URL:    srv.URL,
		MistralURL: srv.URL,
	}, &nopRepository{})

	req := &CompletionRequest{
		Messages:  []ChatMessage{{Role: "user", Content: "list five colors"}},
		MaxTokens: 32,
	}

	resp, err := svc.Complete(context.Background(), "org-1", "req-6", req)
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}
	savings := resp.EcoLLM.Cost.SavingsVsGPT4Percent
	if savings < 0 || savings > 100 {
		t.Errorf("SavingsVsGPT4Percent = %.1f, expected 0-100%%", savings)
	}
	// Small models should show > 50% savings vs GPT-4.
	if savings < 50 {
		t.Errorf("SavingsVsGPT4Percent = %.1f, expected > 50%% for small model", savings)
	}
}
