package chat

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ecollm/api/internal/carbon"
	"github.com/ecollm/api/internal/inference"
	"github.com/ecollm/api/internal/prompt"
	"github.com/ecollm/api/internal/router"
	"github.com/ecollm/api/internal/telemetry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const (
	cacheKeyPrefix = "cache:"
	dedupKeyPrefix = "dedup:"
	cacheTTL       = time.Hour
	dedupTTL       = 5 * time.Second

	// gpt4EnergyKwh is the published GPT-4 energy baseline per request used for
	// live-grid CO2e comparison. Distinct from carbon.GPT4BaselineCO2eGrams which
	// uses a fixed US-average grid intensity.
	gpt4EnergyKwh = 0.00008
)

var tracer = otel.Tracer("ecollm")

// ErrDuplicateRequest is returned when the same request body is submitted within dedupTTL.
var ErrDuplicateRequest = fmt.Errorf("duplicate request")

// ModelSummary is the response shape for each entry in GET /v1/models.
type ModelSummary struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Tasks               []string `json:"tasks"`
	MaxContext          int      `json:"max_context"`
	QualityBenchmark    float64  `json:"quality_benchmark"`
	LatencyP95Ms        float64  `json:"latency_p95_ms"`
	EnergyPerRequestKwh float64  `json:"energy_per_request_kwh"`
	Status              string   `json:"status"`
}

// requestRepository defines the persistence contract used by Service.
type requestRepository interface {
	Insert(ctx context.Context, rec *RequestRecord) error
}

// ServiceConfig groups feature flags passed from main config.
type ServiceConfig struct {
	EnableCache               bool
	EnableFallback            bool
	EnablePromptOptimization  bool
	EnableCarbonTracking      bool
}

// Service orchestrates the full inference pipeline:
// cache-check → optimize → classify → route → infer → carbon → assemble → persist
type Service struct {
	optimizer   *prompt.Optimizer
	classifier  *router.Classifier
	selector    *router.Selector
	gateway     *inference.Gateway
	estimator   *carbon.Estimator
	requestRepo requestRepository
	energyRepo  carbon.EnergyRepository
	carbonRepo  carbon.CarbonRepository
	redis       *redis.Client
	db          *pgxpool.Pool
	cfg         ServiceConfig
}

func NewService(
	optimizer *prompt.Optimizer,
	classifier *router.Classifier,
	selector *router.Selector,
	gateway *inference.Gateway,
	estimator *carbon.Estimator,
	repo requestRepository,
	energyRepo carbon.EnergyRepository,
	carbonRepo carbon.CarbonRepository,
	redisClient *redis.Client,
	cfg ServiceConfig,
	db *pgxpool.Pool,
) *Service {
	return &Service{
		optimizer:   optimizer,
		classifier:  classifier,
		selector:    selector,
		gateway:     gateway,
		estimator:   estimator,
		requestRepo: repo,
		energyRepo:  energyRepo,
		carbonRepo:  carbonRepo,
		redis:       redisClient,
		db:          db,
		cfg:         cfg,
	}
}

// Complete runs the full EcoLLM inference pipeline and returns the response.
func (s *Service) Complete(ctx context.Context, orgID, requestID string, req *CompletionRequest) (*CompletionResponse, error) {
	ctx, span := tracer.Start(ctx, "chat.completion")
	defer span.End()
	span.SetAttributes(
		attribute.String("org_id", orgID),
		attribute.String("request_id", requestID),
	)

	start := time.Now()
	userPrompt := extractUserPrompt(req.Messages)

	// Reject identical body submitted within 5 seconds.
	if s.cfg.EnableCache && s.redis != nil {
		h := sha256.New()
		fmt.Fprintf(h, "%s|%d|%.2f", userPrompt, req.MaxTokens, req.Temperature)
		dedupKey := fmt.Sprintf("%s%x", dedupKeyPrefix, h.Sum(nil))
		set, err := s.redis.SetNX(ctx, dedupKey, requestID, dedupTTL).Result()
		if err == nil && !set {
			return nil, ErrDuplicateRequest
		}
	}

	// Check Redis prompt-response cache.
	if s.cfg.EnableCache && s.redis != nil {
		cacheKey := promptCacheKey(userPrompt, req.MaxTokens, req.Temperature)
		if cached, err := s.redis.Get(ctx, cacheKey).Bytes(); err == nil {
			var resp CompletionResponse
			if json.Unmarshal(cached, &resp) == nil {
				resp.ID = "eco-req-" + requestID
				log.Debug().Str("request_id", requestID).Msg("cache hit")
				span.SetAttributes(attribute.Bool("cache_hit", true))
				go s.persistCacheHit(orgID, requestID, userPrompt, &resp)
				return &resp, nil
			}
		}
	}

	optimized := userPrompt
	if s.cfg.EnablePromptOptimization {
		optCtx, optSpan := tracer.Start(ctx, "prompt.optimize")
		if opt, err := s.optimizer.Optimize(optCtx, userPrompt); err == nil {
			optimized = opt
		}
		optSpan.End()
	}

	_, classifySpan := tracer.Start(ctx, "task.classify")
	classification := s.classifier.Classify(optimized)
	classifySpan.SetAttributes(
		attribute.String("task_type", string(classification.TaskType)),
		attribute.Int("complexity", classification.Complexity),
	)
	classifySpan.End()

	_, selectSpan := tracer.Start(ctx, "router.select")
	var constraints *router.Constraints
	if req.EcoLLM != nil {
		constraints = &router.Constraints{
			MaxLatencyMs: req.EcoLLM.MaxLatencyMs,
			MinQuality:   req.EcoLLM.MinQuality,
		}
	}
	decision := s.selector.Select(classification, constraints)
	selectSpan.SetAttributes(
		attribute.String("model_selected", decision.Model),
		attribute.Float64("routing_score", decision.Score),
	)
	selectSpan.End()

	// Build messages with (possibly) optimized prompt.
	msgs := substituteOptimizedPrompt(req.Messages, userPrompt, optimized)

	// Convert []ChatMessage → []inference.InferenceMessage before gateway calls.
	infMsgs := make([]inference.InferenceMessage, len(msgs))
	for i, m := range msgs {
		infMsgs[i] = inference.InferenceMessage{Role: m.Role, Content: m.Content}
	}

	// Run inference (with fallback if enabled).
	var result *inference.InferenceResult
	var err error
	if s.cfg.EnableFallback {
		result, err = s.gateway.InferWithFallback(ctx, decision, infMsgs, req.MaxTokens, req.Temperature)
	} else {
		result, err = s.gateway.Infer(ctx, decision.Model, infMsgs, req.MaxTokens, req.Temperature)
	}
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return nil, fmt.Errorf("inference failed: %w", err)
	}

	latencyMs := int(time.Since(start).Milliseconds())
	tokensPerSec := 0.0
	if latencyMs > 0 {
		tokensPerSec = float64(result.CompletionTokens) / (float64(latencyMs) / 1000.0)
	}

	// Energy + carbon calculation.
	// When the inference-gateway measured real GPU watt-hours, pass them through
	// so CalculateEnergy skips the static estimate and uses the hardware reading.
	measurement := carbon.CalculateEnergy(carbon.EnergyInput{
		ModelName:        result.ModelUsed,
		InferenceMs:      int64(latencyMs),
		BatchSize:        1,
		MeasuredEnergyWh: result.MeasuredEnergyWh,
	})
	energyKwh := measurement.TotalEnergyKwh
	gridIntensity, gridRegion := s.estimator.GridIntensity()
	co2eGrams := energyKwh * gridIntensity * 1000

	gpt4CO2e := gpt4EnergyKwh * gridIntensity * 1000
	savingsPct := 0.0
	if gpt4CO2e > 0 {
		savingsPct = (gpt4CO2e - co2eGrams) / gpt4CO2e * 100
	}

	status := "success"
	if result.UsedFallback {
		status = "fallback"
	}
	telemetry.RecordRequest(telemetry.RequestResult{
		Model:         result.ModelUsed,
		OrgID:         orgID,
		TaskType:      string(decision.TaskType),
		Status:        status,
		LatencyMs:     latencyMs,
		EnergyKwh:     energyKwh,
		CO2eGrams:     co2eGrams,
		CostUSD:       decision.EstimatedCost,
		UsedFallback:  result.UsedFallback,
		FallbackModel: decision.Fallback,
	})

	resp := &CompletionResponse{
		ID:      "eco-req-" + requestID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   result.ModelUsed,
		Choices: []CompletionChoice{{
			Index:        0,
			Message:      ChatMessage{Role: "assistant", Content: result.Text},
			FinishReason: result.FinishReason,
		}},
		Usage: UsageInfo{
			PromptTokens:     result.PromptTokens,
			CompletionTokens: result.CompletionTokens,
			TotalTokens:      result.PromptTokens + result.CompletionTokens,
		},
		EcoLLM: &EcoLLMMetadata{
			Route: RouteMetadata{
				TaskType:      string(decision.TaskType),
				Complexity:    decision.Complexity,
				ModelSelected: decision.Model,
				FallbackModel: decision.Fallback,
				RoutingScore:  decision.Score,
				Confidence:    decision.Confidence,
				UsedFallback:  result.UsedFallback,
			},
			Energy: EnergyMetadata{
				InferenceEnergyKwh:  measurement.InferenceEnergyWh / 1000.0,
				TotalEnergyKwh:      energyKwh,
				CO2eGrams:           co2eGrams,
				GridCarbonIntensity: gridIntensity,
				GridRegion:          gridRegion,
				EnergySource:        measurement.EnergySource,
			},
			Cost: CostMetadata{
				InferenceCostUSD:     decision.EstimatedCost,
				TotalCostUSD:         decision.EstimatedCost,
				SavingsVsGPT4Percent: savingsVsGPT4(decision.EstimatedCost),
			},
			Performance: PerformanceMetadata{
				LatencyMs:       latencyMs,
				TokensPerSecond: tokensPerSec,
			},
		},
	}

	if s.cfg.EnableCache && s.redis != nil {
		if b, err := json.Marshal(resp); err == nil {
			cacheKey := promptCacheKey(userPrompt, req.MaxTokens, req.Temperature)
			s.redis.Set(ctx, cacheKey, b, cacheTTL)
		}
	}

	// Increment per-org daily usage counter in Redis (fast path for rate limiting).
	if s.redis != nil {
		counterKey := fmt.Sprintf("usage:%s:%s", orgID, time.Now().UTC().Format("2006-01-02"))
		s.redis.Incr(ctx, counterKey)
		s.redis.Expire(ctx, counterKey, 48*time.Hour)
	}

	// Persist request record + energy + carbon asynchronously.
	go s.persistRequest(orgID, requestID, userPrompt, optimized, decision, result, measurement, gridIntensity, gridRegion, co2eGrams, savingsPct, latencyMs)

	log.Info().
		Str("request_id", requestID).
		Str("org_id", orgID).
		Str("model", result.ModelUsed).
		Str("task_type", string(decision.TaskType)).
		Int("latency_ms", latencyMs).
		Float64("energy_kwh", energyKwh).
		Float64("co2e_grams", co2eGrams).
		Float64("cost_usd", decision.EstimatedCost).
		Msg("inference completed")

	return resp, nil
}

// CompleteStream runs the routing pipeline and returns a channel of StreamChunks
// for SSE delivery to the caller. Streaming requests skip the response cache
// (streams are not repeatable) but still go through prompt optimization and routing.
func (s *Service) CompleteStream(ctx context.Context, orgID, requestID string, req *CompletionRequest) (<-chan StreamChunk, error) {
	userPrompt := extractUserPrompt(req.Messages)

	optimized := userPrompt
	if s.cfg.EnablePromptOptimization {
		if opt, err := s.optimizer.Optimize(ctx, userPrompt); err == nil {
			optimized = opt
		}
	}

	classification := s.classifier.Classify(optimized)

	var constraints *router.Constraints
	if req.EcoLLM != nil {
		constraints = &router.Constraints{
			MaxLatencyMs: req.EcoLLM.MaxLatencyMs,
			MinQuality:   req.EcoLLM.MinQuality,
		}
	}
	decision := s.selector.Select(classification, constraints)

	msgs := substituteOptimizedPrompt(req.Messages, userPrompt, optimized)
	infMsgs := make([]inference.InferenceMessage, len(msgs))
	for i, m := range msgs {
		infMsgs[i] = inference.InferenceMessage{Role: m.Role, Content: m.Content}
	}

	rawCh, err := s.gateway.InferStreamWithFallback(ctx, decision, infMsgs, req.MaxTokens, req.Temperature)
	if err != nil {
		return nil, err
	}

	out := make(chan StreamChunk, 64)
	go func() {
		defer close(out)
		for chunk := range rawCh {
			var finishReason *string
			if chunk.FinishReason != "" {
				fr := chunk.FinishReason
				finishReason = &fr
			}
			select {
			case <-ctx.Done():
				return
			case out <- StreamChunk{
				ID:      "eco-stream-" + requestID,
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   chunk.ModelUsed,
				Choices: []StreamChoice{{
					Index:        0,
					Delta:        StreamDelta{Content: chunk.Text},
					FinishReason: finishReason,
				}},
			}:
			}
		}
	}()

	return out, nil
}

// PreviewRoute runs the routing pipeline without inference.
func (s *Service) PreviewRoute(ctx context.Context, orgID string, req *CompletionRequest) (*RoutePreviewResponse, error) {
	userPrompt := extractUserPrompt(req.Messages)
	optimized := userPrompt
	if s.cfg.EnablePromptOptimization {
		if opt, err := s.optimizer.Optimize(ctx, userPrompt); err == nil {
			optimized = opt
		}
	}
	classification := s.classifier.Classify(optimized)
	decision := s.selector.Select(classification, nil)
	energyKwh := s.estimator.EstimateEnergy(decision.Model, decision.EstimatedLatencyMs, 1)
	co2eGrams := s.estimator.EstimateCO2(energyKwh)

	// Build scored candidates for all models in the candidate pool.
	rawCandidates := s.selector.Candidates(classification)
	candidates := make([]ModelCandidate, len(rawCandidates))
	for i, c := range rawCandidates {
		candidates[i] = ModelCandidate{
			Model:     c.Name,
			Score:     c.Score,
			EnergyKwh: c.EnergyKwh,
			CostUSD:   c.CostUSD,
		}
	}

	return &RoutePreviewResponse{
		Route: RouteMetadata{
			TaskType:      string(decision.TaskType),
			Complexity:    decision.Complexity,
			ModelSelected: decision.Model,
			FallbackModel: decision.Fallback,
			RoutingScore:  decision.Score,
			Confidence:    decision.Confidence,
		},
		Candidates:          candidates,
		EstimatedEnergyKwh:  energyKwh,
		EstimatedCO2eGrams:  co2eGrams,
		EstimatedCostUSD:    decision.EstimatedCost,
		EstimatedLatencyMs:  decision.EstimatedLatencyMs,
	}, nil
}

// ListModels returns active models from model_registry with full metadata.
// Falls back to the gateway's static list if the DB is unavailable.
func (s *Service) ListModels(ctx context.Context) ([]ModelSummary, error) {
	if s.db == nil {
		return gatewayModelsToSummary(s.gateway.ListModels(ctx))
	}

	rows, err := s.db.Query(ctx, `
		SELECT name, display_name, max_context_len,
		       COALESCE(quality_benchmark, 0),
		       COALESCE(latency_p95_ms, 0),
		       COALESCE(energy_per_token_kwh, 0),
		       status,
		       COALESCE(quality_tasks::text, '{}')
		FROM model_registry
		WHERE status = 'active'
		ORDER BY name`)
	if err != nil {
		return gatewayModelsToSummary(s.gateway.ListModels(ctx))
	}
	defer rows.Close()

	var models []ModelSummary
	for rows.Next() {
		var name, displayName, status, qualityTasksJSON string
		var maxContext int
		var qualityBenchmark, latencyP95, energyPerToken float64
		if err := rows.Scan(&name, &displayName, &maxContext, &qualityBenchmark,
			&latencyP95, &energyPerToken, &status, &qualityTasksJSON); err != nil {
			continue
		}
		var qualityTasks map[string]float64
		if err := json.Unmarshal([]byte(qualityTasksJSON), &qualityTasks); err != nil {
			log.Warn().Err(err).Str("model", name).Msg("corrupt quality_tasks JSON in model registry")
		}
		tasks := make([]string, 0, len(qualityTasks))
		for k := range qualityTasks {
			tasks = append(tasks, k)
		}
		models = append(models, ModelSummary{
			ID:                  name,
			Name:                displayName,
			Tasks:               tasks,
			MaxContext:          maxContext,
			QualityBenchmark:    qualityBenchmark,
			LatencyP95Ms:        latencyP95,
			EnergyPerRequestKwh: energyPerToken,
			Status:              status,
		})
	}
	if err := rows.Err(); err != nil {
		return gatewayModelsToSummary(s.gateway.ListModels(ctx))
	}
	return models, nil
}

// gatewayModelsToSummary converts the gateway's lightweight ModelInfo slice
// (used when the DB is unavailable) into the unified ModelSummary type.
func gatewayModelsToSummary(infos []inference.ModelInfo, err error) ([]ModelSummary, error) {
	if err != nil {
		return nil, err
	}
	summaries := make([]ModelSummary, len(infos))
	for i, info := range infos {
		summaries[i] = ModelSummary{ID: info.ID, Status: "active"}
	}
	return summaries, nil
}

func (s *Service) persistRequest(
	orgID, requestID, original, optimized string,
	decision router.RouteDecision,
	result *inference.InferenceResult,
	measurement carbon.EnergyMeasurement,
	gridIntensity float64,
	gridRegion string,
	co2eGrams float64,
	savingsPct float64,
	latencyMs int,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rec := &RequestRecord{
		ID:                uuid.NewString(),
		OrgID:             orgID,
		RequestID:         requestID,
		PromptOriginal:    original,
		PromptOptimized:   optimized,
		TaskType:          string(decision.TaskType),
		Complexity:        decision.Complexity,
		ModelSelected:     decision.Model,
		ModelFallback:     decision.Fallback,
		RoutingScore:      decision.Score,
		RoutingConfidence: decision.Confidence,
		UsedFallback:      result.UsedFallback,
		ResponseText:      result.Text,
		FinishReason:      result.FinishReason,
		PromptTokens:      result.PromptTokens,
		CompletionTokens:  result.CompletionTokens,
		TotalTokens:       result.PromptTokens + result.CompletionTokens,
		LatencyMs:         latencyMs,
		Status:            "completed",
	}
	if err := s.requestRepo.Insert(ctx, rec); err != nil {
		log.Error().Err(err).Str("request_id", requestID).Msg("failed to persist request")
		return // no point persisting energy/carbon without the parent row
	}

	if s.cfg.EnableCarbonTracking && s.energyRepo != nil {
		em := measurement
		em.RequestID = rec.ID
		em.OrgID = orgID
		if err := s.energyRepo.InsertMeasurement(ctx, &em); err != nil {
			log.Error().Err(err).Str("request_id", requestID).Msg("failed to persist energy measurement")
		}
	}

	if s.cfg.EnableCarbonTracking && s.carbonRepo != nil {
		gpt4CO2e := gpt4EnergyKwh * gridIntensity * 1000
		ce := &carbon.CarbonEstimate{
			RequestID:           rec.ID,
			GridRegion:          gridRegion,
			GridCarbonIntensity: gridIntensity,
			CarbonDataSource:    "static_fallback",
			CO2eGrams:           co2eGrams,
			GPT4EquivalentCO2e:  gpt4CO2e,
			SavingsPercent:      savingsPct,
		}
		if err := s.carbonRepo.InsertEstimate(ctx, ce); err != nil {
			log.Error().Err(err).Str("request_id", requestID).Msg("failed to persist carbon estimate")
		}
	}
}

// persistCacheHit records a minimal request row for cache hits (no inference).
func (s *Service) persistCacheHit(orgID, requestID, userPrompt string, resp *CompletionResponse) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	model := ""
	if len(resp.Choices) > 0 {
		model = resp.Model
	}

	rec := &RequestRecord{
		ID:           uuid.NewString(),
		OrgID:        orgID,
		RequestID:    requestID,
		PromptOriginal: userPrompt,
		ModelSelected: model,
		CacheHit:     true,
		PromptTokens: resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:  resp.Usage.TotalTokens,
		Status:       "completed",
	}
	if err := s.requestRepo.Insert(ctx, rec); err != nil {
		log.Error().Err(err).Str("request_id", requestID).Msg("failed to persist cache-hit request")
	}
}

// promptCacheKey returns a deterministic Redis key for a prompt + params pair.
func promptCacheKey(prompt string, maxTokens int, temperature float64) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%d|%.2f", prompt, maxTokens, temperature)
	return fmt.Sprintf("%s%x", cacheKeyPrefix, h.Sum(nil))
}

func extractUserPrompt(messages []ChatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	if len(messages) > 0 {
		return messages[len(messages)-1].Content
	}
	return ""
}

func substituteOptimizedPrompt(messages []ChatMessage, original, optimized string) []ChatMessage {
	if original == optimized {
		return messages
	}
	result := make([]ChatMessage, len(messages))
	copy(result, messages)
	for i := len(result) - 1; i >= 0; i-- {
		if result[i].Role == "user" && result[i].Content == original {
			result[i].Content = optimized
			break
		}
	}
	return result
}

const gpt4CostPerRequest = 0.01

func savingsVsGPT4(costUSD float64) float64 {
	if costUSD >= gpt4CostPerRequest {
		return 0
	}
	return (gpt4CostPerRequest - costUSD) / gpt4CostPerRequest * 100
}
