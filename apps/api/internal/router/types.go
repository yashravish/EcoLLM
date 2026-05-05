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
	Energy  float64 // 0.40 — non-negotiable minimum (scores carbon per request)
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
	MaxEnergyKwh = 0.001  // Llama 70B worst-case kWh per request
	MaxCostUSD   = 0.01   // Llama 70B worst-case USD per request
	MaxLatencyMs = 5000   // 5-second ceiling
	MaxRisk      = 0.10   // 10% failure rate ceiling

	// MaxCarbonGrams is the worst-case CO2 per request: Llama 70B (0.0008 kWh)
	// on a coal-heavy grid (850 gCO2/kWh). Used to normalize the carbon dimension.
	MaxCarbonGrams = 680.0

	// DefaultGridIntensity is the US average grid intensity used when no live
	// GridReader is configured (gCO2/kWh, source: EPA eGRID).
	DefaultGridIntensity = 450.0
)

// GridReader provides the current regional grid carbon intensity.
// The selector calls this once per routing decision so scores reflect real-time
// grid conditions rather than a fixed US-average default.
// Implement by wrapping carbon.Estimator.GridIntensity().
type GridReader interface {
	GridCarbonIntensity() float64
}

// RouteDecision is the output of the selector after scoring and constraint enforcement.
type RouteDecision struct {
	Model              string
	Fallback           string
	Score              float64
	TaskType           TaskType
	Complexity         int
	EstimatedEnergyKwh float64
	EstimatedCO2Grams  float64 // computed at routing time using live grid intensity
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
	// BatchSize > 1 enables batch-amortized scoring: energy and cost are divided
	// by batch size so the router correctly prefers larger models for big batches.
	BatchSize int
}
