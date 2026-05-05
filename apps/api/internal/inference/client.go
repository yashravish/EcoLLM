package inference

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// InferenceMessage is the internal message type passed to inference clients.
type InferenceMessage struct {
	Role    string
	Content string
}

// Client is an HTTP client for a single vLLM model runtime instance.
type Client struct {
	modelName     string
	externalModel string
	apiKey        string
	baseURL       string
	httpClient    *http.Client
}

func NewClient(modelName, baseURL, externalModel, apiKey string, timeout time.Duration) *Client {
	ext := externalModel
	if ext == "" {
		ext = modelName
	}
	return &Client{
		modelName:     modelName,
		externalModel: ext,
		apiKey:        apiKey,
		baseURL:       baseURL,
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
// When the inference-gateway injected an X-Measured-Energy-Wh header, the value
// is stored in InferenceResult.MeasuredEnergyWh so the carbon layer can use real
// GPU telemetry instead of a static power estimate.
func (c *Client) Infer(ctx context.Context, messages []InferenceMessage, maxTokens int, temperature float64) (*InferenceResult, error) {
	vllmMsgs := make([]vLLMMessage, len(messages))
	for i, m := range messages {
		vllmMsgs[i] = vLLMMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := vLLMRequest{
		Model:       c.externalModel,
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
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

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

	// Read measured energy injected by the inference-gateway (when DCGM is active).
	var measuredEnergyWh float64
	if h := resp.Header.Get("X-Measured-Energy-Wh"); h != "" {
		measuredEnergyWh, _ = strconv.ParseFloat(h, 64)
	}

	return &InferenceResult{
		Text:             vllmResp.Choices[0].Message.Content,
		FinishReason:     vllmResp.Choices[0].FinishReason,
		PromptTokens:     vllmResp.Usage.PromptTokens,
		CompletionTokens: vllmResp.Usage.CompletionTokens,
		LatencyMs:        latencyMs,
		ModelUsed:        c.modelName,
		MeasuredEnergyWh: measuredEnergyWh,
	}, nil
}

// InferStream sends a streaming chat completion request and returns a channel of
// token chunks. The channel is closed when the stream ends or the context is cancelled.
// Each chunk carries the delta content; the final chunk has a non-empty FinishReason.
func (c *Client) InferStream(ctx context.Context, messages []InferenceMessage, maxTokens int, temperature float64) (<-chan InferenceChunk, error) {
	vllmMsgs := make([]vLLMMessage, len(messages))
	for i, m := range messages {
		vllmMsgs[i] = vLLMMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := vLLMRequest{
		Model:       c.externalModel,
		Messages:    vllmMsgs,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Stream:      true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stream request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("vllm returned status %d", resp.StatusCode)
	}

	ch := make(chan InferenceChunk, 64)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return
			}
			var chunk vLLMStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) == 0 {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case ch <- InferenceChunk{
				Text:         chunk.Choices[0].Delta.Content,
				FinishReason: chunk.Choices[0].FinishReason,
				ModelUsed:    c.modelName,
			}:
			}
		}
	}()

	return ch, nil
}
