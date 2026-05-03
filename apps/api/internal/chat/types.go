package chat

import "time"

// ChatMessage mirrors the OpenAI message format.
type ChatMessage struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

// EcoLLMOptions are the EcoLLM-specific extensions on the chat completion request.
type EcoLLMOptions struct {
	Prefer          string  `json:"prefer,omitempty"`           // efficiency | quality | speed
	MaxLatencyMs    int     `json:"max_latency_ms,omitempty"`
	MinQuality      float64 `json:"min_quality,omitempty"`
	IncludeMetadata bool    `json:"include_metadata,omitempty"`
}

// CompletionRequest is the OpenAI-compatible request body for POST /v1/chat/completions.
type CompletionRequest struct {
	Messages    []ChatMessage  `json:"messages"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Temperature float64        `json:"temperature,omitempty"`
	Model       string         `json:"model,omitempty"` // optional override
	EcoLLM      *EcoLLMOptions `json:"ecollm,omitempty"`
}

// CompletionResponse is the OpenAI-compatible response body.
type CompletionResponse struct {
	ID      string            `json:"id"`
	Object  string            `json:"object"`
	Created int64             `json:"created"`
	Model   string            `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   UsageInfo         `json:"usage"`
	EcoLLM  *EcoLLMMetadata   `json:"ecollm,omitempty"`
}

// CompletionChoice is a single generated response.
type CompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// UsageInfo mirrors the OpenAI usage object.
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// EcoLLMMetadata is appended to every response when include_metadata is true
// (or by default) to expose routing, energy, cost, and performance details.
type EcoLLMMetadata struct {
	Route       RouteMetadata       `json:"route"`
	Energy      EnergyMetadata      `json:"energy"`
	Cost        CostMetadata        `json:"cost"`
	Performance PerformanceMetadata `json:"performance"`
}

type RouteMetadata struct {
	TaskType      string  `json:"task_type"`
	Complexity    int     `json:"complexity"`
	ModelSelected string  `json:"model_selected"`
	FallbackModel string  `json:"fallback_model,omitempty"`
	RoutingScore  float64 `json:"routing_score"`
	Confidence    float64 `json:"confidence"`
	UsedFallback  bool    `json:"used_fallback"`
}

type EnergyMetadata struct {
	InferenceEnergyKwh  float64 `json:"inference_energy_kwh"`
	TotalEnergyKwh      float64 `json:"total_energy_kwh"`
	CO2eGrams           float64 `json:"co2e_grams"`
	GridCarbonIntensity float64 `json:"grid_carbon_intensity"`
	GridRegion          string  `json:"grid_region"`
}

type CostMetadata struct {
	InferenceCostUSD      float64 `json:"inference_cost_usd"`
	TotalCostUSD          float64 `json:"total_cost_usd"`
	SavingsVsGPT4Percent  float64 `json:"savings_vs_gpt4_percent"`
}

type PerformanceMetadata struct {
	LatencyMs          int     `json:"latency_ms"`
	TimeToFirstTokenMs int     `json:"time_to_first_token_ms"`
	TokensPerSecond    float64 `json:"tokens_per_second"`
}

// RoutePreviewResponse is returned by POST /v1/route/preview.
type RoutePreviewResponse struct {
	Route               RouteMetadata    `json:"route"`
	Candidates          []ModelCandidate `json:"candidates"`
	EstimatedEnergyKwh  float64          `json:"estimated_energy_kwh"`
	EstimatedCO2eGrams  float64          `json:"estimated_co2e_grams"`
	EstimatedCostUSD    float64          `json:"estimated_cost_usd"`
	EstimatedLatencyMs  int              `json:"estimated_latency_ms"`
}

// ModelCandidate is a scored candidate model returned in the preview.
type ModelCandidate struct {
	Model     string  `json:"model"`
	Score     float64 `json:"score"`
	EnergyKwh float64 `json:"energy_kwh"`
	CostUSD   float64 `json:"cost_usd"`
}

// RequestRecord is the domain type stored in the requests table.
type RequestRecord struct {
	ID                string
	OrgID             string
	RequestID         string
	PromptOriginal    string
	PromptOptimized   string
	TaskType          string
	Complexity        int
	ModelSelected     string
	ModelFallback     string
	RoutingScore      float64
	RoutingConfidence float64
	UsedFallback      bool
	CacheHit          bool
	ResponseText      string
	FinishReason      string
	PromptTokens      int
	CompletionTokens  int
	TotalTokens       int
	LatencyMs         int
	Status            string
	ErrorMessage      string
	CreatedAt         time.Time
}
