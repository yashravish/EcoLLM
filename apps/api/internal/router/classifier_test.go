package router

import (
	"strings"
	"testing"
)

// TestClassifier_Classify is the primary table-driven test for task classification.
// Expected complexities are derived from the scoring rules in classifier.go:
//   base=1, hard keywords add 2-3, medium add 1, length>500 adds 2, >1500 adds 2 more,
//   multi-step adds 2. Confidence = 0.9 when 3+ signals fire, else 0.8.
func TestClassifier_Classify(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name      string
		prompt    string
		wantType  TaskType
		wantMinCx int
		wantMaxCx int
	}{
		// ── Simple ─────────────────────────────────────────────────────────
		{
			name:      "simple_question",
			prompt:    "What is the capital of France?",
			wantType:  TaskSimple,
			wantMinCx: 1, wantMaxCx: 3,
		},
		{
			name:      "classify_keyword",
			prompt:    "Classify this email as spam or not spam.",
			wantType:  TaskSimple,
			wantMinCx: 1, wantMaxCx: 3,
		},
		{
			name:      "extract_keyword",
			prompt:    "Extract all email addresses from this text.",
			wantType:  TaskSimple,
			wantMinCx: 1, wantMaxCx: 3,
		},
		{
			name:      "list_keyword",
			prompt:    "List all users in the database.",
			wantType:  TaskSimple,
			wantMinCx: 1, wantMaxCx: 3,
		},
		{
			name:      "short_explain",
			prompt:    "explain quicksort", // explain(+1) = cx 2
			wantType:  TaskSimple,
			wantMinCx: 1, wantMaxCx: 3,
		},

		// ── Medium ──────────────────────────────────────────────────────────
		{
			name: "code_generation_multi_keyword",
			// write(+1) + code(+1) + generate(+1) + summarize(+1) = cx 5
			prompt:    "Write code to generate and summarize a CSV report.",
			wantType:  TaskMedium,
			wantMinCx: 4, wantMaxCx: 6,
		},
		{
			name: "multi_step_first_then",
			// generate(+1) + first…then multi-step(+2) = cx 4
			prompt:    "First parse the CSV, then aggregate by region, then generate a chart.",
			wantType:  TaskMedium,
			wantMinCx: 4, wantMaxCx: 6,
		},
		{
			name: "step_by_step_not_double_counted",
			// explain(+1) + multi-step "step by step"(+2) = cx 4 (NOT cx 6)
			prompt:    "Please explain step by step how quicksort works.",
			wantType:  TaskMedium,
			wantMinCx: 4, wantMaxCx: 5,
		},
		{
			name: "refactor_and_code",
			// refactor(+2) + code(+1) = cx 4
			prompt:    "Refactor this code to be cleaner.",
			wantType:  TaskMedium,
			wantMinCx: 4, wantMaxCx: 6,
		},

		// ── Hard ────────────────────────────────────────────────────────────
		{
			name: "complex_debug_multiple_hard",
			// debug(+3) + analyze(+2) + optimize(+2) = cx 8
			prompt:    "Debug this distributed cache, analyze its failure modes, and optimize the eviction algorithm.",
			wantType:  TaskHard,
			wantMinCx: 7, wantMaxCx: 9,
		},
		{
			name: "long_complex_prompt",
			// length>500(+2) + length>1500(+2) + analyze(+2) = cx 7
			prompt:    strings.Repeat("Analyze this codebase task detail ", 50), // 1700 chars
			wantType:  TaskHard,
			wantMinCx: 7, wantMaxCx: 9,
		},
		{
			name: "hard_debug_plus_reason",
			// debug(+3) + reason about(+3) = cx 7
			prompt:    "Debug this service and reason about why the circuit breaker keeps tripping.",
			wantType:  TaskHard,
			wantMinCx: 7, wantMaxCx: 9,
		},

		// ── Specialized ─────────────────────────────────────────────────────
		{
			name: "architect_design_reason",
			// architect(+3) + design system(+3) + reason about(+3) = cx 10 (capped)
			prompt:    "Architect a multi-tenant solution and reason about the design system for all tradeoffs.",
			wantType:  TaskSpecialized,
			wantMinCx: 10, wantMaxCx: 10,
		},
		{
			name: "many_hard_keywords_capped",
			// debug+analyze+compare+refactor+optimize+reason about = 3+2+2+2+2+3 = 14 → capped 10
			prompt:    "Debug, analyze, compare, refactor, optimize the system and reason about the impact.",
			wantType:  TaskSpecialized,
			wantMinCx: 10, wantMaxCx: 10,
		},

		// ── Edge cases ──────────────────────────────────────────────────────
		{
			name:      "empty_string",
			prompt:    "",
			wantType:  TaskSimple,
			wantMinCx: 1, wantMaxCx: 1,
		},
		{
			name:      "single_word",
			prompt:    "Hi",
			wantType:  TaskSimple,
			wantMinCx: 1, wantMaxCx: 1,
		},
		{
			name: "single_hard_keyword",
			// debug(+3) = cx 4 → TaskMedium (not TaskSimple despite being one word)
			prompt:    "debug",
			wantType:  TaskMedium,
			wantMinCx: 4, wantMaxCx: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Classify(tt.prompt)

			if result.TaskType != tt.wantType {
				t.Errorf("TaskType = %q, want %q (complexity=%d, signals=%v)",
					result.TaskType, tt.wantType, result.Complexity, result.Signals)
			}
			if result.Complexity < tt.wantMinCx || result.Complexity > tt.wantMaxCx {
				t.Errorf("Complexity = %d, want [%d, %d] (taskType=%s, signals=%v)",
					result.Complexity, tt.wantMinCx, tt.wantMaxCx, result.TaskType, result.Signals)
			}
		})
	}
}

func TestClassifier_ConfidenceThreshold(t *testing.T) {
	c := NewClassifier()

	t.Run("high_confidence_three_or_more_signals", func(t *testing.T) {
		// debug(signal) + analyze(signal) + optimize(signal) = 3 signals → 0.9
		result := c.Classify("Debug this service, analyze the failure, and optimize the query.")
		if result.Confidence != 0.9 {
			t.Errorf("Confidence = %.2f, want 0.9 (signals=%v)", result.Confidence, result.Signals)
		}
		if len(result.Signals) < 3 {
			t.Errorf("expected >= 3 signals, got %d: %v", len(result.Signals), result.Signals)
		}
	})

	t.Run("base_confidence_fewer_than_three_signals", func(t *testing.T) {
		// "what is" = 1 simple signal → 0.8
		result := c.Classify("What is the capital of France?")
		if result.Confidence != 0.8 {
			t.Errorf("Confidence = %.2f, want 0.8 (signals=%v)", result.Confidence, result.Signals)
		}
	})
}

func TestClassifier_SignalsReported(t *testing.T) {
	c := NewClassifier()

	t.Run("simple_keyword_signal_present", func(t *testing.T) {
		result := c.Classify("What is the capital of France?")
		if !containsSignal(result.Signals, "simple:what is") {
			t.Errorf("expected signal 'simple:what is' in %v", result.Signals)
		}
	})

	t.Run("hard_keyword_signal_present", func(t *testing.T) {
		result := c.Classify("Debug this service.")
		if !containsSignal(result.Signals, "hard:debug") {
			t.Errorf("expected signal 'hard:debug' in %v", result.Signals)
		}
	})

	t.Run("multi_step_signal_present", func(t *testing.T) {
		result := c.Classify("First do A, then do B.")
		if !containsSignal(result.Signals, "multi_step") {
			t.Errorf("expected signal 'multi_step' in %v", result.Signals)
		}
	})

	t.Run("length_signal_present_for_long_prompts", func(t *testing.T) {
		result := c.Classify(strings.Repeat("a", 501))
		if !containsSignal(result.Signals, "long_prompt") {
			t.Errorf("expected signal 'long_prompt' in %v", result.Signals)
		}
	})

	t.Run("no_double_count_step_by_step", func(t *testing.T) {
		// After the fix, "step by step" should produce exactly one signal (multi_step),
		// not two (hard:step by step + multi_step).
		result := c.Classify("step by step")
		hardCount := 0
		for _, s := range result.Signals {
			if strings.HasPrefix(s, "hard:step") {
				hardCount++
			}
		}
		if hardCount > 0 {
			t.Errorf("'step by step' must not appear as a hard keyword signal, got %v", result.Signals)
		}
		if !containsSignal(result.Signals, "multi_step") {
			t.Errorf("expected 'multi_step' signal, got %v", result.Signals)
		}
	})
}

func TestClassifier_Concurrent(t *testing.T) {
	c := NewClassifier()
	prompt := "Debug and analyze this distributed system."

	done := make(chan struct{}, 50)
	for i := 0; i < 50; i++ {
		go func() {
			result := c.Classify(prompt)
			if result.TaskType == "" {
				t.Errorf("concurrent Classify returned empty TaskType")
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 50; i++ {
		<-done
	}
}

func TestClassifier_ComplexityBoundary(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name      string
		prompt    string
		wantMinCx int
		wantMaxCx int
		wantType  TaskType
	}{
		{
			name: "boundary_simple_to_medium",
			// complexity exactly 3 → TaskSimple; complexity 4 → TaskMedium
			// "explain"(+1) + step_by_step(+2) = 4 → TaskMedium
			prompt:    "Explain step by step.",
			wantMinCx: 4, wantMaxCx: 4,
			wantType:  TaskMedium,
		},
		{
			name: "boundary_medium_to_hard",
			// complexity exactly 7 → TaskHard
			// debug(+3) + reason about(+3) = 7
			prompt:    "Debug this and reason about the fix.",
			wantMinCx: 7, wantMaxCx: 7,
			wantType:  TaskHard,
		},
		{
			name: "complexity_capped_at_10",
			// many hard keywords → must not exceed 10
			prompt:    strings.Repeat("debug architect analyze compare refactor optimize reason about design system ", 5),
			wantMinCx: 10, wantMaxCx: 10,
			wantType:  TaskSpecialized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Classify(tt.prompt)
			if result.Complexity < tt.wantMinCx || result.Complexity > tt.wantMaxCx {
				t.Errorf("Complexity = %d, want [%d, %d]", result.Complexity, tt.wantMinCx, tt.wantMaxCx)
			}
			if result.TaskType != tt.wantType {
				t.Errorf("TaskType = %s, want %s", result.TaskType, tt.wantType)
			}
		})
	}
}

// containsSignal is a helper that checks if a specific signal string is present.
func containsSignal(signals []string, target string) bool {
	for _, s := range signals {
		if s == target {
			return true
		}
	}
	return false
}
