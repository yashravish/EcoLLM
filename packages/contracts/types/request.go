// Package types contains the canonical request/response types shared between
// Go services. These are generated from openapi.yaml — do not edit manually.
package types

// ChatCompletionRequest is the OpenAI-compatible inference request body.
type ChatCompletionRequest struct {
	Messages    []Message              `json:"messages"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	EcoLLM      *EcoLLMRequestOptions  `json:"ecollm,omitempty"`
}

// Message is a single turn in the conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// EcoLLMRequestOptions extends the standard request with routing hints.
type EcoLLMRequestOptions struct {
	Prefer          string  `json:"prefer,omitempty"`
	MaxLatencyMs    int     `json:"max_latency_ms,omitempty"`
	MinQuality      float64 `json:"min_quality,omitempty"`
	IncludeMetadata bool    `json:"include_metadata,omitempty"`
}
