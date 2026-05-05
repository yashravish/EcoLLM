package inference

// InferenceResult is returned by the inference gateway after a successful call.
type InferenceResult struct {
	Text             string
	FinishReason     string
	PromptTokens     int
	CompletionTokens int
	LatencyMs        int
	UsedFallback     bool
	ModelUsed        string
	// MeasuredEnergyWh is populated when the inference-gateway injected a
	// X-Measured-Energy-Wh header — meaning live DCGM/NVML telemetry was
	// available. Zero means the carbon layer should use its static estimate.
	MeasuredEnergyWh float64
}

// InferenceChunk is a single token chunk emitted during a streaming response.
type InferenceChunk struct {
	Text         string
	FinishReason string // empty until the final chunk
	ModelUsed    string
}

// ── vLLM wire types (non-streaming) ──────────────────────────────────────────

// vLLMRequest is the OpenAI-compatible request sent to the vLLM HTTP endpoint.
type vLLMRequest struct {
	Model       string        `json:"model"`
	Messages    []vLLMMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

type vLLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// vLLMResponse is the OpenAI-compatible response from vLLM.
type vLLMResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Choices []vLLMChoice `json:"choices"`
	Usage   vLLMUsage    `json:"usage"`
}

type vLLMChoice struct {
	Index        int         `json:"index"`
	Message      vLLMMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type vLLMUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ── vLLM wire types (streaming) ───────────────────────────────────────────────

type vLLMStreamChunk struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []vLLMStreamChoice  `json:"choices"`
}

type vLLMStreamChoice struct {
	Index        int        `json:"index"`
	Delta        vLLMDelta  `json:"delta"`
	FinishReason string     `json:"finish_reason"`
}

type vLLMDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content"`
}
