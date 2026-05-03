package router

import "testing"

func newTestSelector() *Selector {
	return NewSelector(nil) // scorer field unused by Select; nil is safe
}

func TestSelector_SimpleTaskRoutes_ToPhi3(t *testing.T) {
	s := newTestSelector()
	classification := ClassificationResult{TaskType: TaskSimple, Complexity: 1, Confidence: 0.8}

	decision := s.Select(classification, nil)

	if decision.Model != "phi_3" {
		t.Errorf("simple task should route to phi_3, got %q", decision.Model)
	}
	if decision.Fallback != "mistral_7b" {
		t.Errorf("phi_3 fallback should be mistral_7b, got %q", decision.Fallback)
	}
}

func TestSelector_MediumTaskRoutes_ToMistral(t *testing.T) {
	s := newTestSelector()
	classification := ClassificationResult{TaskType: TaskMedium, Complexity: 5, Confidence: 0.8}

	decision := s.Select(classification, nil)

	// mistral_7b scores higher than llama_13b for medium tasks because of energy weight.
	if decision.Model != "mistral_7b" {
		t.Errorf("medium task should route to mistral_7b, got %q", decision.Model)
	}
}

func TestSelector_HardTaskRoutes_ToLlama13B(t *testing.T) {
	s := newTestSelector()
	classification := ClassificationResult{TaskType: TaskHard, Complexity: 8, Confidence: 0.9}

	decision := s.Select(classification, nil)

	// llama_13b scores vastly higher than llama_70b (energy gap 5×) for hard tasks.
	if decision.Model != "llama_13b" {
		t.Errorf("hard task should route to llama_13b, got %q", decision.Model)
	}
	if decision.Fallback != "llama_70b" {
		t.Errorf("llama_13b fallback should be llama_70b, got %q", decision.Fallback)
	}
}

func TestSelector_ConstraintFiltering_FallsBackToGlobalSmallest(t *testing.T) {
	// Medium task candidates are mistral_7b (400ms) and llama_13b (800ms).
	// Both exceed MaxLatencyMs=200ms → all filtered → should fall back to phi_3.
	s := newTestSelector()
	classification := ClassificationResult{TaskType: TaskMedium, Complexity: 5, Confidence: 0.8}
	constraints := &Constraints{MaxLatencyMs: 200}

	decision := s.Select(classification, constraints)

	if decision.Model != "phi_3" {
		t.Errorf("all-filtered medium task with MaxLatencyMs=200 should return phi_3, got %q", decision.Model)
	}
}

func TestSelector_FallbackChainCorrect(t *testing.T) {
	s := newTestSelector()

	tests := []struct {
		taskType     TaskType
		wantModel    string
		wantFallback string
	}{
		{TaskSimple, "phi_3", "mistral_7b"},
		{TaskMedium, "mistral_7b", "llama_13b"},
		{TaskHard, "llama_13b", "llama_70b"},
		{TaskSpecialized, "llama_70b", ""},
	}

	for _, tt := range tests {
		classification := ClassificationResult{TaskType: tt.taskType, Complexity: 5, Confidence: 0.8}
		decision := s.Select(classification, nil)

		if decision.Model != tt.wantModel {
			t.Errorf("task=%s: model=%q, want=%q", tt.taskType, decision.Model, tt.wantModel)
		}
		if decision.Fallback != tt.wantFallback {
			t.Errorf("task=%s: fallback=%q, want=%q", tt.taskType, decision.Fallback, tt.wantFallback)
		}
	}
}

func TestSelector_AllFilteredReturnsSmallestNotError(t *testing.T) {
	// MaxCostUSD=0.000001 is below every model's cost → all filtered → phi_3 returned.
	s := newTestSelector()
	classification := ClassificationResult{TaskType: TaskHard, Complexity: 8, Confidence: 0.9}
	constraints := &Constraints{MaxCostUSD: 0.000001}

	decision := s.Select(classification, constraints)

	if decision.Model == "" {
		t.Error("all-filtered selection should return phi_3, not empty string")
	}
	if decision.Model != "phi_3" {
		t.Errorf("all-filtered fallback should be phi_3 (global smallest), got %q", decision.Model)
	}
}

func TestSelector_DecisionIncludesEnergyEstimate(t *testing.T) {
	s := newTestSelector()
	classification := ClassificationResult{TaskType: TaskSimple, Complexity: 1, Confidence: 0.8}

	decision := s.Select(classification, nil)

	if decision.EstimatedEnergyKwh <= 0 {
		t.Errorf("EstimatedEnergyKwh should be positive, got %g", decision.EstimatedEnergyKwh)
	}
	if decision.Score <= 0 || decision.Score > 1.0 {
		t.Errorf("Score should be in (0, 1], got %.4f", decision.Score)
	}
}

func TestSelector_QualityConstraintFilters(t *testing.T) {
	// MinQuality=0.90 on a simple task: phi_3 (0.65) and mistral_7b (0.85) both fail → phi_3 fallback.
	s := newTestSelector()
	classification := ClassificationResult{TaskType: TaskSimple, Complexity: 1, Confidence: 0.8}
	constraints := &Constraints{MinQuality: 0.90}

	decision := s.Select(classification, constraints)

	if decision.Model != "phi_3" {
		t.Errorf("quality-filtered simple task should return phi_3 (global smallest), got %q", decision.Model)
	}
}
