package router

import (
	"math"
	"testing"
)

// phi3Candidate and mistralCandidate mirror staticCandidates values.
// Defined here so scorer tests are self-contained and don't depend on selector.go globals.
var (
	phi3Candidate = ModelCandidate{
		Name:             "phi_3",
		EnergyKwh:        0.00001,
		CostUSD:          0.0001,
		QualityBenchmark: 0.65,
		LatencyP95Ms:     80,
		FailureRate:      0.01,
	}
	mistralCandidate = ModelCandidate{
		Name:             "mistral_7b",
		EnergyKwh:        0.00008,
		CostUSD:          0.0005,
		QualityBenchmark: 0.85,
		LatencyP95Ms:     400,
		FailureRate:      0.02,
	}
	llama13bCandidate = ModelCandidate{
		Name:             "llama_13b",
		EnergyKwh:        0.00015,
		CostUSD:          0.001,
		QualityBenchmark: 0.92,
		LatencyP95Ms:     800,
		FailureRate:      0.02,
	}
)

func TestScoreModel_OrderingSimpleTask(t *testing.T) {
	// For simple tasks the energy weight (0.40) drives Phi-3 above Mistral 7B
	// because Phi-3 is 8× more energy-efficient.
	phi3Score := ScoreModel(phi3Candidate, TaskSimple, DefaultWeights)
	mistralScore := ScoreModel(mistralCandidate, TaskSimple, DefaultWeights)

	if phi3Score <= mistralScore {
		t.Errorf("Phi-3 score (%.4f) should exceed Mistral score (%.4f) for simple tasks",
			phi3Score, mistralScore)
	}
}

func TestScoreModel_OrderingHardTask(t *testing.T) {
	// For hard tasks (quality threshold 0.85) Phi-3 is penalised (quality 0.65 < 0.85).
	// Mistral should outscore Phi-3 on hard tasks.
	phi3Score := ScoreModel(phi3Candidate, TaskHard, DefaultWeights)
	mistralScore := ScoreModel(mistralCandidate, TaskHard, DefaultWeights)

	if phi3Score >= mistralScore {
		t.Errorf("Mistral score (%.4f) should exceed Phi-3 score (%.4f) for hard tasks",
			mistralScore, phi3Score)
	}
}

func TestScoreModel_QualityThresholdPenalty(t *testing.T) {
	// A model whose quality falls below the task threshold receives a 0.5× quality
	// multiplier. The penalised score must be measurably lower than the unpenalised.
	//
	// Phi-3 quality=0.65, hard-task threshold=0.85 → penalty applies.
	penalised := ScoreModel(phi3Candidate, TaskHard, DefaultWeights)

	// Construct an identical candidate with quality just above the threshold.
	phi3AboveThreshold := phi3Candidate
	phi3AboveThreshold.QualityBenchmark = 0.90

	unpenalised := ScoreModel(phi3AboveThreshold, TaskHard, DefaultWeights)

	if penalised >= unpenalised {
		t.Errorf("penalised score (%.4f) should be lower than unpenalised (%.4f)",
			penalised, unpenalised)
	}

	// Quantify: penalty removes 0.20 × (quality × 0.5 - quality) from the score.
	expectedDiff := DefaultWeights.Quality * (phi3AboveThreshold.QualityBenchmark - phi3Candidate.QualityBenchmark*0.5)
	actualDiff := unpenalised - penalised
	if math.Abs(actualDiff-expectedDiff) > 0.0001 {
		t.Errorf("score diff %.4f, expected %.4f", actualDiff, expectedDiff)
	}
}

func TestScoreModel_KnownValue(t *testing.T) {
	// Manual calculation for Phi-3 on a simple task with DefaultWeights:
	//
	//   energyScore  = 1 - (0.00001 / 0.001)  = 0.99
	//   costScore    = 1 - (0.0001  / 0.01)   = 0.99
	//   qualityScore = 0.65  (>= 0.60 threshold — no penalty)
	//   latencyScore = 1 - (80 / 5000)         = 0.984
	//   riskPenalty  = (0.01 / 0.10) × 0.05   = 0.005
	//
	//   score = 0.40×0.99 + 0.30×0.99 + 0.20×0.65 + 0.10×0.984 − 0.005
	//         = 0.396 + 0.297 + 0.130 + 0.0984 − 0.005
	//         = 0.9164
	const want = 0.9164
	got := ScoreModel(phi3Candidate, TaskSimple, DefaultWeights)
	if math.Abs(got-want) > 0.0001 {
		t.Errorf("ScoreModel(phi3, simple) = %.6f, want %.4f", got, want)
	}
}

func TestScoreModel_ZeroEnergy(t *testing.T) {
	c := phi3Candidate
	c.EnergyKwh = 0
	score := ScoreModel(c, TaskSimple, DefaultWeights)
	// energyScore must be 1.0 (best possible), so total score is higher than baseline.
	baseline := ScoreModel(phi3Candidate, TaskSimple, DefaultWeights)
	if score <= baseline {
		t.Errorf("zero-energy model should score higher (%.4f) than baseline (%.4f)", score, baseline)
	}
}

func TestScoreModel_MaxEnergy(t *testing.T) {
	c := phi3Candidate
	c.EnergyKwh = MaxEnergyKwh
	score := ScoreModel(c, TaskSimple, DefaultWeights)
	// energyScore = 1 - (MaxEnergyKwh / MaxEnergyKwh) = 0 → energy contribution = 0.
	// Full score = 0 + 0.30×costScore + 0.20×quality + 0.10×latency - risk
	maxEnergyFloor := 0.30*clamp(1.0-(phi3Candidate.CostUSD/MaxCostUSD), 0, 1) +
		0.20*phi3Candidate.QualityBenchmark +
		0.10*clamp(1.0-(float64(phi3Candidate.LatencyP95Ms)/MaxLatencyMs), 0, 1) -
		(phi3Candidate.FailureRate/MaxRisk)*0.05
	if math.Abs(score-maxEnergyFloor) > 0.0001 {
		t.Errorf("max-energy score %.4f, expected floor %.4f", score, maxEnergyFloor)
	}
}

func TestScoreModel_MaxLatency(t *testing.T) {
	c := phi3Candidate
	c.LatencyP95Ms = MaxLatencyMs
	score := ScoreModel(c, TaskSimple, DefaultWeights)
	// latencyScore = 0 → latency contributes nothing.
	withMaxLatency := score
	c.LatencyP95Ms = 0
	withZeroLatency := ScoreModel(c, TaskSimple, DefaultWeights)
	diff := withZeroLatency - withMaxLatency
	if math.Abs(diff-DefaultWeights.Latency) > 0.0002 {
		t.Errorf("latency diff = %.4f, expected ~%.2f (latency weight)", diff, DefaultWeights.Latency)
	}
}

func TestScoreModel_FullRiskPenalty(t *testing.T) {
	c := phi3Candidate
	c.FailureRate = MaxRisk // 10% — maximum risk
	score := ScoreModel(c, TaskSimple, DefaultWeights)
	// riskPenalty = (MaxRisk / MaxRisk) × 0.05 = 0.05
	noRisk := phi3Candidate
	noRisk.FailureRate = 0
	noRiskScore := ScoreModel(noRisk, TaskSimple, DefaultWeights)
	if math.Abs((noRiskScore-score)-0.05) > 0.0001 {
		t.Errorf("full risk penalty should be exactly 0.05, got %.4f", noRiskScore-score)
	}
}

func TestScoreModel_EnergyScoreClampedAboveMax(t *testing.T) {
	c := phi3Candidate
	c.EnergyKwh = MaxEnergyKwh * 10 // deliberately exceed maximum
	score := ScoreModel(c, TaskSimple, DefaultWeights)
	if score < 0 {
		t.Errorf("score should be >= 0 even with over-max energy, got %.4f", score)
	}
	// energyScore must be clamped to 0, not negative.
	// Verify by checking score equals a direct calculation with energyScore=0.
	clamped := phi3Candidate
	clamped.EnergyKwh = MaxEnergyKwh
	clampedScore := ScoreModel(clamped, TaskSimple, DefaultWeights)
	if math.Abs(score-clampedScore) > 0.0001 {
		t.Errorf("over-max energy score (%.4f) should equal clamped score (%.4f)", score, clampedScore)
	}
}

// ── ValidateWeights tests ─────────────────────────────────────────────────────

func TestValidateWeights_DefaultWeightsValid(t *testing.T) {
	if err := ValidateWeights(DefaultWeights); err != nil {
		t.Errorf("DefaultWeights should be valid, got: %v", err)
	}
}

func TestValidateWeights_EnergyBelowMinimumRejected(t *testing.T) {
	w := ScoringWeights{Energy: 0.30, Cost: 0.35, Quality: 0.25, Latency: 0.10}
	if err := ValidateWeights(w); err == nil {
		t.Error("expected error for energy weight 0.30, got nil")
	}
}

func TestValidateWeights_EnergyAtExactMinimumAccepted(t *testing.T) {
	w := ScoringWeights{Energy: 0.35, Cost: 0.35, Quality: 0.20, Latency: 0.10}
	if err := ValidateWeights(w); err != nil {
		t.Errorf("energy weight at minimum 0.35 should be valid, got: %v", err)
	}
}

func TestValidateWeights_WeightsDontSumToOneRejected(t *testing.T) {
	w := ScoringWeights{Energy: 0.40, Cost: 0.30, Quality: 0.20, Latency: 0.20} // sums to 1.10
	if err := ValidateWeights(w); err == nil {
		t.Error("expected error for weights summing to 1.10, got nil")
	}
}
