package router

import "fmt"

const minEnergyWeight = 0.35 // environment-first hard floor

// ValidateWeights returns an error if the weight configuration violates the
// environment-first constraint or does not sum to 1.0 (±0.01 float tolerance).
func ValidateWeights(w ScoringWeights) error {
	if w.Energy < minEnergyWeight {
		return fmt.Errorf(
			"energy weight %.2f is below minimum %.2f (environment-first constraint)",
			w.Energy, minEnergyWeight,
		)
	}
	total := w.Energy + w.Cost + w.Quality + w.Latency
	if total < 0.99 || total > 1.01 {
		return fmt.Errorf("weights must sum to 1.0, got %.4f", total)
	}
	return nil
}

// ScoreModel computes the environment-first routing score for a model candidate.
//
// Formula (from architecture doc Section 7.5):
//
//	route_score =
//	    0.40 × (1 - energy_kwh / max_energy)   ← ENERGY — non-negotiable primary
//	  + 0.30 × (1 - cost_usd / max_cost)
//	  + 0.20 × quality_benchmark
//	  + 0.10 × (1 - latency_ms / max_latency)
//	  - 0.05 × (failure_rate / max_risk)
//
// Energy weight is 0.40 and cannot go below 0.35 without explicit justification.
func ScoreModel(candidate ModelCandidate, taskType TaskType, weights ScoringWeights) float64 {
	// Normalize each dimension to [0, 1] where higher = better
	energyScore := clamp(1.0-(candidate.EnergyKwh/MaxEnergyKwh), 0, 1)
	costScore := clamp(1.0-(candidate.CostUSD/MaxCostUSD), 0, 1)
	qualityScore := clamp(candidate.QualityBenchmark, 0, 1)
	latencyScore := clamp(1.0-(float64(candidate.LatencyP95Ms)/MaxLatencyMs), 0, 1)

	// Apply task-specific quality threshold penalty (not a disqualifier, but heavy)
	minQuality := minQualityForTask(taskType)
	if qualityScore < minQuality {
		qualityScore *= 0.5
	}

	// Composite score
	score := weights.Energy*energyScore +
		weights.Cost*costScore +
		weights.Quality*qualityScore +
		weights.Latency*latencyScore

	// Risk penalty: subtract proportional to historical failure rate
	riskPenalty := (candidate.FailureRate / MaxRisk) * 0.05
	score -= riskPenalty

	return score
}

// minQualityForTask returns the minimum acceptable quality score for a task type.
// Below this threshold the model receives a 50% quality score penalty.
func minQualityForTask(t TaskType) float64 {
	switch t {
	case TaskSimple:
		return 0.60
	case TaskMedium:
		return 0.75
	case TaskHard:
		return 0.85
	case TaskSpecialized:
		return 0.90
	default:
		return 0.70
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
