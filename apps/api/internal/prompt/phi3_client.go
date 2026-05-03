package prompt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Phi3Client sends prompts to the Python Phi-3 FastAPI sidecar for refinement.
// It is only invoked when rule-based confidence falls below 0.7 — i.e., < 10%
// of requests in normal operation.
type Phi3Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewPhi3Client(baseURL string) *Phi3Client {
	return &Phi3Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Phi-3 inference is fast; fail quickly if slow
		},
	}
}

type phi3Request struct {
	Prompt string `json:"prompt"`
}

type phi3Response struct {
	OptimizedPrompt string `json:"optimized_prompt"`
}

// Refine sends the raw prompt to the Phi-3 sidecar and returns the rewritten text.
// Any network or parse error causes the caller to fall back to rule-based output.
func (c *Phi3Client) Refine(ctx context.Context, rawPrompt string) (string, error) {
	body, err := json.Marshal(phi3Request{Prompt: rawPrompt})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/optimize", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("phi3 request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("phi3 returned status %d", resp.StatusCode)
	}

	var result phi3Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.OptimizedPrompt, nil
}
