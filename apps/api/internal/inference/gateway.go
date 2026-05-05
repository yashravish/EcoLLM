package inference

import (
	"context"
	"fmt"
	"time"

	"github.com/ecollm/api/internal/router"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("ecollm")

// Gateway routes inference requests to the correct vLLM client and implements
// the fallback chain: phi_3 → mistral_7b → llama_13b → llama_70b → error.
type Gateway struct {
	clients map[string]*Client
}

// ModelInfo holds the identifier for a configured model.
type ModelInfo struct {
	ID string `json:"id"`
}

// InferenceEndpoints maps model names to their vLLM base URLs.
type InferenceEndpoints struct {
	Phi3URL     string
	MistralURL  string
	Llama13BURL string
	Llama70BURL string

	// ExternalModelNames maps internal model IDs to the model name sent upstream.
	// When empty the internal name is used (local vLLM / Ollama mode).
	Phi3ExternalModel     string
	MistralExternalModel  string
	Llama13BExternalModel string
	Llama70BExternalModel string

	// APIKey is the Bearer token for external APIs (Groq, OpenRouter, etc.).
	APIKey string
}

func NewGateway(endpoints InferenceEndpoints, timeout time.Duration) *Gateway {
	clients := map[string]*Client{
		"phi_3":      NewClient("phi_3", endpoints.Phi3URL, endpoints.Phi3ExternalModel, endpoints.APIKey, timeout),
		"mistral_7b": NewClient("mistral_7b", endpoints.MistralURL, endpoints.MistralExternalModel, endpoints.APIKey, timeout),
		"llama_13b":  NewClient("llama_13b", endpoints.Llama13BURL, endpoints.Llama13BExternalModel, endpoints.APIKey, timeout),
		"llama_70b":  NewClient("llama_70b", endpoints.Llama70BURL, endpoints.Llama70BExternalModel, endpoints.APIKey, timeout),
	}
	return &Gateway{clients: clients}
}

// InferWithFallback attempts inference on the primary model and falls back
// to the decision's Fallback model on any error.
func (g *Gateway) InferWithFallback(
	ctx context.Context,
	decision router.RouteDecision,
	messages []InferenceMessage,
	maxTokens int,
	temperature float64,
) (*InferenceResult, error) {
	ctx, span := tracer.Start(ctx, "inference.dispatch")
	defer span.End()
	span.SetAttributes(
		attribute.String("model.primary", decision.Model),
		attribute.String("model.fallback", decision.Fallback),
	)

	result, err := g.infer(ctx, decision.Model, messages, maxTokens, temperature)
	if err == nil {
		return result, nil
	}

	if decision.Fallback == "" {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, fmt.Errorf("primary model %s failed and no fallback: %w", decision.Model, err)
	}

	log.Warn().
		Str("primary", decision.Model).
		Str("fallback", decision.Fallback).
		Err(err).
		Msg("primary model failed, trying fallback")

	result, err = g.infer(ctx, decision.Fallback, messages, maxTokens, temperature)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, fmt.Errorf("fallback model %s also failed: %w", decision.Fallback, err)
	}

	result.UsedFallback = true
	span.SetAttributes(attribute.Bool("used_fallback", true))
	return result, nil
}

// Infer dispatches a single inference request to the named model.
func (g *Gateway) Infer(ctx context.Context, modelName string, messages []InferenceMessage, maxTokens int, temperature float64) (*InferenceResult, error) {
	return g.infer(ctx, modelName, messages, maxTokens, temperature)
}

func (g *Gateway) infer(ctx context.Context, modelName string, messages []InferenceMessage, maxTokens int, temperature float64) (*InferenceResult, error) {
	ctx, span := tracer.Start(ctx, "inference.model")
	defer span.End()
	span.SetAttributes(
		attribute.String("model", modelName),
		attribute.Int("max_tokens", maxTokens),
	)

	client, ok := g.clients[modelName]
	if !ok {
		return nil, fmt.Errorf("no client for model %s", modelName)
	}
	result, err := client.Infer(ctx, messages, maxTokens, temperature)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
	}
	return result, err
}

// InferStreamWithFallback starts a streaming inference on the primary model and
// falls back to the decision's Fallback model if the primary stream cannot start.
func (g *Gateway) InferStreamWithFallback(
	ctx context.Context,
	decision router.RouteDecision,
	messages []InferenceMessage,
	maxTokens int,
	temperature float64,
) (<-chan InferenceChunk, error) {
	ch, err := g.inferStream(ctx, decision.Model, messages, maxTokens, temperature)
	if err == nil {
		return ch, nil
	}

	if decision.Fallback == "" {
		return nil, fmt.Errorf("primary model %s streaming failed and no fallback: %w", decision.Model, err)
	}

	log.Warn().
		Str("primary", decision.Model).
		Str("fallback", decision.Fallback).
		Err(err).
		Msg("primary model streaming failed, trying fallback")

	return g.inferStream(ctx, decision.Fallback, messages, maxTokens, temperature)
}

// InferStream starts a streaming inference request on the named model.
func (g *Gateway) InferStream(ctx context.Context, modelName string, messages []InferenceMessage, maxTokens int, temperature float64) (<-chan InferenceChunk, error) {
	return g.inferStream(ctx, modelName, messages, maxTokens, temperature)
}

func (g *Gateway) inferStream(ctx context.Context, modelName string, messages []InferenceMessage, maxTokens int, temperature float64) (<-chan InferenceChunk, error) {
	client, ok := g.clients[modelName]
	if !ok {
		return nil, fmt.Errorf("no client for model %s", modelName)
	}
	return client.InferStream(ctx, messages, maxTokens, temperature)
}

// ListModels returns the set of configured model names.
func (g *Gateway) ListModels(ctx context.Context) ([]ModelInfo, error) {
	models := make([]ModelInfo, 0, len(g.clients))
	for name := range g.clients {
		models = append(models, ModelInfo{ID: name})
	}
	return models, nil
}
