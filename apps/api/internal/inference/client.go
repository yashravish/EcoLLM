package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client is an HTTP client for a single vLLM model runtime instance.
type Client struct {
	modelName  string
	baseURL    string
	httpClient *http.Client
}

func NewClient(modelName, baseURL string, timeout time.Duration) *Client {
	return &Client{
		modelName: modelName,
		baseURL:   baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        20,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// Infer sends a chat completion request to the vLLM instance and returns the result.
func (c *Client) Infer(ctx context.Context, messages []InferenceMessage, maxTokens int, temperature float64) (*InferenceResult, error) {
	vllmMsgs := make([]vLLMMessage, len(messages))
	for i, m := range messages {
		vllmMsgs[i] = vLLMMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := vLLMRequest{
		Model:       c.modelName,
		Messages:    vllmMsgs,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Stream:      false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inference request: %w", err)
	}
	defer resp.Body.Close()

	latencyMs := int(time.Since(start).Milliseconds())

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vllm returned status %d", resp.StatusCode)
	}

	var vllmResp vLLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&vllmResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(vllmResp.Choices) == 0 {
		return nil, fmt.Errorf("vllm returned no choices")
	}

	return &InferenceResult{
		Text:             vllmResp.Choices[0].Message.Content,
		FinishReason:     vllmResp.Choices[0].FinishReason,
		PromptTokens:     vllmResp.Usage.PromptTokens,
		CompletionTokens: vllmResp.Usage.CompletionTokens,
		LatencyMs:        latencyMs,
		ModelUsed:        c.modelName,
	}, nil
}

// InferenceMessage is the internal message type passed to inference clients.
type InferenceMessage struct {
	Role    string
	Content string
}
