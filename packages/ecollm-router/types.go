// Package router is the open-source EcoLLM routing library.
//
// It provides the task classifier and environment-first scoring function that
// power the EcoLLM managed service. Publishing this as a standalone module lets
// the community audit the routing logic, contribute improvements, and self-host
// the routing layer — fulfilling the open-source foundation mandate.
//
// Usage:
//
//	c := router.NewClassifier()
//	result := c.Classify(prompt)
//
//	score := router.ScoreModel(candidate, result.TaskType, router.DefaultWeights, gridIntensity, batchSize)
package router

// TaskType classifies the complexity of an incoming request.
type TaskType string

const (
	TaskSimple      TaskType = "simple"      // FAQ, classification, extraction
	TaskMedium      TaskType = "medium"      // Writing, basic coding, summarization
	TaskHard        TaskType = "hard"        // Complex reasoning, debugging, analysis
	TaskSpecialized TaskType = "specialized" // Domain-specific, long-form
)

// ClassificationResult is the output of the task classifier.
type ClassificationResult struct {
	TaskType   TaskType
	Complexity int      // 1–10
	Confidence float64  // 0–1
	Signals    []string // Which heuristics fired
}

// ModelCandidate holds the properties of a model used for scoring.
type ModelCandidate struct {
	Name             string
	Size             string  // e.g. "7B"
	Quantization     string  // e.g. "awq", "gptq"
	LatencyP95Ms     int     // p95 inference latency in milliseconds
	EnergyKwh        float64 // energy per request in kWh
	CostUSD          float64 // cost per request in USD
	QualityBenchmark float64 // 0–1; higher is better
	FailureRate      float64 // historical error rate (0–1)
}

// ScoringWeights defines the weights for the routing score formula.
// The Energy weight represents the carbon dimension and cannot go below 0.35
// without violating the environment-first constraint.
type ScoringWeights struct {
	Energy  float64 // 0.40 — non-negotiable minimum
	Cost    float64 // 0.30
	Quality float64 // 0.20
	Latency float64 // 0.10
}

// DefaultWeights are the canonical weights from the EcoLLM architecture spec.
var DefaultWeights = ScoringWeights{
	Energy:  0.40,
	Cost:    0.30,
	Quality: 0.20,
	Latency: 0.10,
}

// Reference maximums used for score normalization.
const (
	MaxCostUSD   = 0.01   // Llama 70B worst-case USD per request
	MaxLatencyMs = 5000.0 // 5-second ceiling

	// MaxCarbonGrams is the worst-case CO2 per request: Llama 70B (0.0008 kWh)
	// on a coal-heavy grid (850 gCO2/kWh × 1000 = g). Used to normalize the
	// carbon dimension so the score is always in [0, 1].
	MaxCarbonGrams = 680.0

	// DefaultGridIntensity is the US average grid carbon intensity in gCO2/kWh
	// (source: EPA eGRID). Used when no live grid reader is provided.
	DefaultGridIntensity = 450.0

	// MaxRisk is the failure-rate ceiling for the risk penalty calculation.
	MaxRisk = 0.10
)
