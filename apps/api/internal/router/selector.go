package router

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Candidate pools per task type — ordered smallest → largest (energy-first preference).
var modelCandidates = map[TaskType][]string{
	TaskSimple:      {"phi_3", "mistral_7b"},
	TaskMedium:      {"mistral_7b", "llama_13b"},
	TaskHard:        {"llama_13b", "llama_70b"},
	TaskSpecialized: {"llama_70b"},
}

// Fallback chain: on primary failure, escalate to next larger model.
var fallbackChain = map[string]string{
	"phi_3":      "mistral_7b",
	"mistral_7b": "llama_13b",
	"llama_13b":  "llama_70b",
	"llama_70b":  "", // terminal: no further fallback
}

// staticCandidates are the baseline model properties seeded at startup.
var staticCandidates = map[string]ModelCandidate{
	"phi_3": {
		Name: "phi_3", Size: "3.8B", Quantization: "awq",
		LatencyP95Ms: 80, EnergyKwh: 0.00001, CostUSD: 0.0001,
		QualityBenchmark: 0.65, FailureRate: 0.01,
	},
	"mistral_7b": {
		Name: "mistral_7b", Size: "7B", Quantization: "awq",
		LatencyP95Ms: 400, EnergyKwh: 0.00008, CostUSD: 0.0005,
		QualityBenchmark: 0.85, FailureRate: 0.02,
	},
	"llama_13b": {
		Name: "llama_13b", Size: "13B", Quantization: "awq",
		LatencyP95Ms: 800, EnergyKwh: 0.00015, CostUSD: 0.001,
		QualityBenchmark: 0.92, FailureRate: 0.02,
	},
	"llama_70b": {
		Name: "llama_70b", Size: "70B", Quantization: "gptq",
		LatencyP95Ms: 3000, EnergyKwh: 0.0008, CostUSD: 0.005,
		QualityBenchmark: 0.98, FailureRate: 0.03,
	},
}

// Selector chooses the best model from the candidate pool for a given classification.
// It supports three optional extensions beyond the static baseline:
//
//  1. WithGridReader — live grid intensity so the carbon score uses actual gCO2/kWh
//     rather than the US-average default.
//  2. WithDB + StartScoreRefresh — quality scores are periodically overridden with
//     EWMA values computed from real customer feedback by the learning worker.
//  3. BatchSize in Constraints — energy and cost are amortized across the batch.
type Selector struct {
	scorer        *Scorer
	db            *pgxpool.Pool
	gridReader    GridReader
	mu            sync.RWMutex
	learnedScores map[string]float64 // key: "model:task_type" → quality 0–1
}

func NewSelector(scorer *Scorer) *Selector {
	return &Selector{
		scorer:        scorer,
		learnedScores: make(map[string]float64),
	}
}

// WithDB enables dynamic quality score loading from the model_quality_scores table.
// Must be called before StartScoreRefresh.
func (s *Selector) WithDB(db *pgxpool.Pool) *Selector {
	s.db = db
	return s
}

// WithGridReader plugs in a live grid intensity source. When set, every routing
// decision uses the current gCO2/kWh value rather than the 450 gCO2/kWh default.
func (s *Selector) WithGridReader(g GridReader) *Selector {
	s.gridReader = g
	return s
}

// StartScoreRefresh launches a background goroutine that reloads learned quality
// scores from model_quality_scores on the given interval. Call once from main.
func (s *Selector) StartScoreRefresh(ctx context.Context, interval time.Duration) {
	if s.db == nil {
		return
	}
	go func() {
		s.refreshScores(ctx)
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.refreshScores(ctx)
			}
		}
	}()
}

func (s *Selector) refreshScores(ctx context.Context) {
	rows, err := s.db.Query(ctx,
		`SELECT model_name, task_type, quality_score FROM model_quality_scores`)
	if err != nil {
		// Table may not exist yet (first run before the learning worker creates it).
		log.Debug().Err(err).Msg("selector: could not load learned quality scores")
		return
	}
	defer rows.Close()

	fresh := make(map[string]float64)
	for rows.Next() {
		var model, taskType string
		var score float64
		if err := rows.Scan(&model, &taskType, &score); err == nil {
			fresh[model+":"+taskType] = score
		}
	}

	s.mu.Lock()
	s.learnedScores = fresh
	s.mu.Unlock()
	log.Debug().Int("entries", len(fresh)).Msg("selector: learned quality scores refreshed")
}

// gridIntensity returns the current carbon intensity or the default if no reader.
func (s *Selector) gridIntensity() float64 {
	if s.gridReader != nil {
		return s.gridReader.GridCarbonIntensity()
	}
	return DefaultGridIntensity
}

// learnedQuality returns the feedback-derived quality score for a model+task pair,
// or the static benchmark if no learned score exists yet.
func (s *Selector) learnedQuality(modelName string, taskType TaskType, staticBenchmark float64) float64 {
	s.mu.RLock()
	score, ok := s.learnedScores[modelName+":"+string(taskType)]
	s.mu.RUnlock()
	if ok {
		return score
	}
	return staticBenchmark
}

// Select applies the scoring function to all candidates, enforces customer
// constraints, and returns the RouteDecision with the highest score.
func (s *Selector) Select(classification ClassificationResult, constraints *Constraints) RouteDecision {
	candidateNames := modelCandidates[classification.TaskType]

	intensity := s.gridIntensity()
	batchSize := 1
	if constraints != nil && constraints.BatchSize > 1 {
		batchSize = constraints.BatchSize
	}

	bestModel := ""
	bestScore := -1.0

	for _, name := range candidateNames {
		candidate, ok := staticCandidates[name]
		if !ok {
			continue
		}

		// Override quality with feedback-learned score when available.
		candidate.QualityBenchmark = s.learnedQuality(name, classification.TaskType, candidate.QualityBenchmark)

		// Skip candidates that violate hard customer constraints.
		if constraints != nil {
			if constraints.MaxLatencyMs > 0 && candidate.LatencyP95Ms > constraints.MaxLatencyMs {
				continue
			}
			if constraints.MinQuality > 0 && candidate.QualityBenchmark < constraints.MinQuality {
				continue
			}
			if constraints.MaxCostUSD > 0 && candidate.CostUSD > constraints.MaxCostUSD {
				continue
			}
			if constraints.MaxEnergyKwh > 0 && candidate.EnergyKwh > constraints.MaxEnergyKwh {
				continue
			}
		}

		score := ScoreModel(candidate, classification.TaskType, DefaultWeights, intensity, batchSize)
		if score > bestScore {
			bestScore = score
			bestModel = name
		}
	}

	// If all candidates were filtered by constraints, degrade to phi_3 (smallest).
	// Preserves environment-first principle even under tight customer constraints.
	if bestModel == "" {
		bestModel = "phi_3"
		bestScore = ScoreModel(staticCandidates[bestModel], classification.TaskType, DefaultWeights, intensity, batchSize)
	}

	selected := staticCandidates[bestModel]

	return RouteDecision{
		Model:              bestModel,
		Fallback:           fallbackChain[bestModel],
		Score:              bestScore,
		TaskType:           classification.TaskType,
		Complexity:         classification.Complexity,
		EstimatedEnergyKwh: selected.EnergyKwh,
		EstimatedCO2Grams:  selected.EnergyKwh * intensity * 1000,
		EstimatedCost:      selected.CostUSD,
		Confidence:         classification.Confidence,
		EstimatedLatencyMs: selected.LatencyP95Ms,
	}
}

// Candidates returns all scored models in the candidate pool for the given
// classification. Used by PreviewRoute to show the full scoring breakdown.
func (s *Selector) Candidates(classification ClassificationResult) []ScoredCandidate {
	names := modelCandidates[classification.TaskType]
	intensity := s.gridIntensity()
	result := make([]ScoredCandidate, 0, len(names))
	for _, name := range names {
		c, ok := staticCandidates[name]
		if !ok {
			continue
		}
		c.QualityBenchmark = s.learnedQuality(name, classification.TaskType, c.QualityBenchmark)
		score := ScoreModel(c, classification.TaskType, DefaultWeights, intensity, 1)
		result = append(result, ScoredCandidate{
			Name:      c.Name,
			Score:     score,
			EnergyKwh: c.EnergyKwh,
			CostUSD:   c.CostUSD,
			LatencyMs: c.LatencyP95Ms,
			Quality:   c.QualityBenchmark,
		})
	}
	return result
}
