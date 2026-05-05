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
// The energy dimension now scores actual CO2 per request (energy × grid intensity)
// rather than raw kWh, so the router responds to real-time grid conditions:
//
//   - During clean-energy hours (low intensity), larger models score closer to
//     smaller ones — the carbon gap narrows.
//   - During coal-heavy hours (high intensity), the smallest model is strongly
//     preferred because the carbon gap widens.
//
// batchSize > 1 amortizes energy and cost across the batch so the router
// correctly scores batched inference as cheaper per request.
//
// Formula (from architecture doc Section 7.5, updated for carbon-first scoring):
//
//	carbon_score = 1 - (EnergyKwh/batchSize × gridIntensity × 1000) / MaxCarbonGrams
//	cost_score   = 1 - (CostUSD/batchSize) / MaxCostUSD
//	quality_score = QualityBenchmark           (with task-threshold penalty)
//	latency_score = 1 - LatencyP95Ms / MaxLatencyMs
//	risk_penalty  = (FailureRate / MaxRisk) × 0.05
//
//	score = 0.40×carbon + 0.30×cost + 0.20×quality + 0.10×latency − risk_penalty
func ScoreModel(candidate ModelCandidate, taskType TaskType, weights ScoringWeights, gridIntensityGCO2 float64, batchSize int) float64 {
	if gridIntensityGCO2 <= 0 {
		gridIntensityGCO2 = DefaultGridIntensity
	}
	if batchSize <= 0 {
		batchSize = 1
	}

	// Amortize energy and cost by batch size.
	effectiveEnergyKwh := candidate.EnergyKwh / float64(batchSize)
	effectiveCostUSD := candidate.CostUSD / float64(batchSize)

	// Carbon score: grams CO2 for this request on the current grid, normalized
	// against the absolute worst case (Llama 70B on coal-heavy grid).
	co2Grams := effectiveEnergyKwh * gridIntensityGCO2 * 1000
	carbonScore := clamp(1.0-co2Grams/MaxCarbonGrams, 0, 1)

	costScore := clamp(1.0-(effectiveCostUSD/MaxCostUSD), 0, 1)
	qualityScore := clamp(candidate.QualityBenchmark, 0, 1)
	latencyScore := clamp(1.0-(float64(candidate.LatencyP95Ms)/MaxLatencyMs), 0, 1)

	// Task-specific quality threshold: below it the model receives a 50% penalty
	// (not a hard disqualification — constraint enforcement handles that).
	minQuality := minQualityForTask(taskType)
	if qualityScore < minQuality {
		qualityScore *= 0.5
	}

	score := weights.Energy*carbonScore +
		weights.Cost*costScore +
		weights.Quality*qualityScore +
		weights.Latency*latencyScore

	// Risk penalty: subtract proportional to historical failure rate.
	riskPenalty := (candidate.FailureRate / MaxRisk) * 0.05
	score -= riskPenalty

	return score
}

// minQualityForTask returns the minimum acceptable quality score for a task type.
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
