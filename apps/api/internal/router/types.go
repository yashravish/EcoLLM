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

// ModelCandidate holds the static properties of a model used for scoring.
type ModelCandidate struct {
	Name             string
	Size             string
	Quantization     string
	LatencyP95Ms     int
	EnergyKwh        float64
	CostUSD          float64
	QualityBenchmark float64 // 0–1, from benchmark suite
	FailureRate      float64 // Historical failure rate (0–1)
}

// ScoringWeights defines the weights for the routing score formula.
// Energy weight cannot go below 0.35 without explicit justification.
type ScoringWeights struct {
	Energy  float64 // 0.40 — non-negotiable minimum
	Cost    float64 // 0.30
	Quality float64 // 0.20
	Latency float64 // 0.10
}

// DefaultWeights are the canonical weights defined in the architecture spec.
var DefaultWeights = ScoringWeights{
	Energy:  0.40,
	Cost:    0.30,
	Quality: 0.20,
	Latency: 0.10,
}

// Reference maximums used for score normalization.
const (
	MaxEnergyKwh = 0.001  // Llama 70B worst-case
	MaxCostUSD   = 0.01   // Llama 70B worst-case
	MaxLatencyMs = 5000   // 5-second ceiling
	MaxRisk      = 0.10   // 10% failure rate ceiling
)

// RouteDecision is the output of the selector after scoring and constraint enforcement.
type RouteDecision struct {
	Model              string
	Fallback           string
	Score              float64
	TaskType           TaskType
	Complexity         int
	EstimatedEnergyKwh float64
	EstimatedCO2Grams  float64
	EstimatedCost      float64
	Confidence         float64
	EstimatedLatencyMs int
}

// ScoredCandidate is a scored model candidate returned by Selector.Candidates.
type ScoredCandidate struct {
	Name      string
	Score     float64
	EnergyKwh float64
	CostUSD   float64
	LatencyMs int
	Quality   float64
}

// Constraints are customer-supplied overrides applied after scoring.
type Constraints struct {
	MaxLatencyMs int
	MinQuality   float64
	MaxCostUSD   float64
	MaxEnergyKwh float64
}
