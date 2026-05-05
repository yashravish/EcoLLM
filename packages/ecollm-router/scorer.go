package router

import "fmt"

const minEnergyWeight = 0.35

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
// The energy dimension scores actual CO2 per request (energy × gridIntensityGCO2)
// rather than raw kWh so the router responds to real-time grid conditions:
//
//   - Low grid intensity (solar/wind): larger models become more competitive.
//   - High grid intensity (coal): smallest model is strongly preferred.
//
// batchSize > 1 amortizes energy and cost per request across the batch, making
// batched inference score better than single-request inference for the same model.
//
// Pass gridIntensityGCO2 = 0 to use the US-average default (450 gCO2/kWh).
// Pass batchSize = 0 or 1 for single-request scoring.
//
// Returns a score in approximately [0, 1]; higher is better.
func ScoreModel(candidate ModelCandidate, taskType TaskType, weights ScoringWeights, gridIntensityGCO2 float64, batchSize int) float64 {
	if gridIntensityGCO2 <= 0 {
		gridIntensityGCO2 = DefaultGridIntensity
	}
	if batchSize <= 0 {
		batchSize = 1
	}

	effectiveEnergyKwh := candidate.EnergyKwh / float64(batchSize)
	effectiveCostUSD := candidate.CostUSD / float64(batchSize)

	co2Grams := effectiveEnergyKwh * gridIntensityGCO2 * 1000
	carbonScore := clamp(1.0-co2Grams/MaxCarbonGrams, 0, 1)
	costScore := clamp(1.0-(effectiveCostUSD/MaxCostUSD), 0, 1)
	qualityScore := clamp(candidate.QualityBenchmark, 0, 1)
	latencyScore := clamp(1.0-(float64(candidate.LatencyP95Ms)/MaxLatencyMs), 0, 1)

	minQuality := minQualityForTask(taskType)
	if qualityScore < minQuality {
		qualityScore *= 0.5
	}

	score := weights.Energy*carbonScore +
		weights.Cost*costScore +
		weights.Quality*qualityScore +
		weights.Latency*latencyScore

	riskPenalty := (candidate.FailureRate / MaxRisk) * 0.05
	score -= riskPenalty

	return score
}

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
