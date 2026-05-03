// Package types contains the canonical request/response types shared between
// Go services. These are generated from openapi.yaml — do not edit manually.
package types

// ChatCompletionResponse is the OpenAI-compatible response with EcoLLM metadata.
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
	EcoLLM  Metadata `json:"ecollm"`
}

// Choice is a single completion candidate.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage contains token counts for the request.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Metadata contains EcoLLM-specific per-request telemetry.
type Metadata struct {
	Route       RouteMetadata       `json:"route"`
	Energy      EnergyMetadata      `json:"energy"`
	Cost        CostMetadata        `json:"cost"`
	Performance PerformanceMetadata `json:"performance"`
}

// RouteMetadata describes how the request was routed.
type RouteMetadata struct {
	TaskType      string  `json:"task_type"`
	Complexity    float64 `json:"complexity"`
	ModelSelected string  `json:"model_selected"`
	FallbackModel string  `json:"fallback_model,omitempty"`
	RoutingScore  float64 `json:"routing_score"`
	Confidence    float64 `json:"confidence"`
	UsedFallback  bool    `json:"used_fallback"`
}

// EnergyMetadata describes the energy and carbon footprint.
type EnergyMetadata struct {
	TotalEnergyKWh       float64 `json:"total_energy_kwh"`
	CO2eGrams            float64 `json:"co2e_grams"`
	GridRegion           string  `json:"grid_region"`
	GridCarbonIntensity  float64 `json:"grid_carbon_intensity,omitempty"`
}

// CostMetadata describes the cost of the request.
type CostMetadata struct {
	TotalCostUSD         float64 `json:"total_cost_usd"`
	SavingsVsGPT4Percent float64 `json:"savings_vs_gpt4_percent"`
}

// PerformanceMetadata describes latency measurements.
type PerformanceMetadata struct {
	LatencyMs          int `json:"latency_ms"`
	TimeToFirstTokenMs int `json:"time_to_first_token_ms,omitempty"`
}
