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
}

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
