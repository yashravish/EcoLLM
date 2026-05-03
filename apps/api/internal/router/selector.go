package router

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
	"llama_70b":  "", // terminal: no fallback, return error
}

// staticCandidates are the model properties loaded from model-configs at startup.
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
type Selector struct {
	scorer *Scorer
}

func NewSelector(scorer *Scorer) *Selector {
	return &Selector{scorer: scorer}
}

// Select applies the scoring function to all candidates, enforces customer
// constraints, and returns the RouteDecision with the highest score.
func (s *Selector) Select(classification ClassificationResult, constraints *Constraints) RouteDecision {
	candidateNames := modelCandidates[classification.TaskType]

	bestModel := ""
	bestScore := -1.0

	for _, name := range candidateNames {
		candidate, ok := staticCandidates[name]
		if !ok {
			continue
		}

		// Skip candidates that violate hard customer constraints
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

		score := ScoreModel(candidate, classification.TaskType, DefaultWeights)
		if score > bestScore {
			bestScore = score
			bestModel = name
		}
	}

	// If all candidates were filtered by constraints, degrade to the globally smallest
	// model (phi_3) rather than failing or selecting a larger model. This preserves the
	// energy-first principle even under tight customer constraints.
	if bestModel == "" {
		bestModel = "phi_3"
		bestScore = ScoreModel(staticCandidates[bestModel], classification.TaskType, DefaultWeights)
	}

	selected := staticCandidates[bestModel]

	return RouteDecision{
		Model:              bestModel,
		Fallback:           fallbackChain[bestModel],
		Score:              bestScore,
		TaskType:           classification.TaskType,
		Complexity:         classification.Complexity,
		EstimatedEnergyKwh: selected.EnergyKwh,
		EstimatedCost:      selected.CostUSD,
		Confidence:         classification.Confidence,
		EstimatedLatencyMs: selected.LatencyP95Ms,
	}
}

// Candidates returns all scored models in the candidate pool for the given classification.
func (s *Selector) Candidates(classification ClassificationResult) []ScoredCandidate {
	names := modelCandidates[classification.TaskType]
	result := make([]ScoredCandidate, 0, len(names))
	for _, name := range names {
		c, ok := staticCandidates[name]
		if !ok {
			continue
		}
		score := ScoreModel(c, classification.TaskType, DefaultWeights)
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
