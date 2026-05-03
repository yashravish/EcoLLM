package prompt

import (
	"context"
	"strings"
	"testing"
)

// newTestOptimizer creates an Optimizer with no Phi-3 sidecar.
// An empty URL means the sidecar client will always fail → rule-based path only.
func newTestOptimizer() *Optimizer {
	return NewOptimizer("")
}

func TestOptimizer_CodeRequestAddsLanguageGuidance(t *testing.T) {
	o := newTestOptimizer()
	result, err := o.OptimizeWithResult(context.Background(), "write code to sort a list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Optimized, "programming language") {
		t.Errorf("expected language guidance, got: %q", result.Optimized)
	}
	if !containsRule(result.RulesApplied, "add_language_guidance_for_code") {
		t.Errorf("expected rule 'add_language_guidance_for_code' in %v", result.RulesApplied)
	}
}

func TestOptimizer_CodeWithLanguageSpecified_NoGuidanceAdded(t *testing.T) {
	o := newTestOptimizer()
	prompt := "write code in Python to sort a list"
	result, err := o.OptimizeWithResult(context.Background(), prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if containsRule(result.RulesApplied, "add_language_guidance_for_code") {
		t.Error("should not add language guidance when language is already specified")
	}
}

func TestOptimizer_ShortExplanationAddsConciseness(t *testing.T) {
	o := newTestOptimizer()
	result, err := o.OptimizeWithResult(context.Background(), "what is quicksort")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Optimized, "concise") {
		t.Errorf("expected conciseness directive, got: %q", result.Optimized)
	}
	if !containsRule(result.RulesApplied, "add_format_guidance_for_explanations") {
		t.Errorf("expected rule 'add_format_guidance_for_explanations' in %v", result.RulesApplied)
	}
}

func TestOptimizer_TruncatesExcessiveContext(t *testing.T) {
	o := newTestOptimizer()
	// 5000-char prompt; truncation rule keeps first 2000 + last 2000 + separator (~31 chars).
	longPrompt := strings.Repeat("x", 5000)
	result, err := o.OptimizeWithResult(context.Background(), longPrompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	const maxLen = 4100
	if len(result.Optimized) > maxLen {
		t.Errorf("truncated prompt length %d exceeds limit %d", len(result.Optimized), maxLen)
	}
	if !strings.Contains(result.Optimized, "truncated") {
		t.Errorf("expected truncation marker in output: %q", result.Optimized[:100])
	}
	if !containsRule(result.RulesApplied, "truncate_excessive_context") {
		t.Errorf("expected rule 'truncate_excessive_context' in %v", result.RulesApplied)
	}
}

func TestOptimizer_WhitespaceNormalization(t *testing.T) {
	o := newTestOptimizer()
	// Triple newlines should collapse to double newline.
	prompt := "Line one\n\n\nLine two\n\n\n\nLine three"
	result, err := o.OptimizeWithResult(context.Background(), prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result.Optimized, "\n\n\n") {
		t.Errorf("triple newlines should be collapsed, got: %q", result.Optimized)
	}
	if !containsRule(result.RulesApplied, "remove_redundant_whitespace") {
		t.Errorf("expected rule 'remove_redundant_whitespace' in %v", result.RulesApplied)
	}
}

func TestOptimizer_NoOpPromptUnchanged(t *testing.T) {
	o := newTestOptimizer()
	// A well-formed prompt with explicit format guidance and a language already specified.
	// Only the add_conciseness_directive might fire — prevent that by including "Be concise."
	wellFormed := "Write a Python function that sorts a list in ascending order. Be concise."
	result, err := o.OptimizeWithResult(context.Background(), wellFormed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No structural rules (language guidance, explanation, truncation, whitespace) should fire.
	for _, rule := range []string{
		"add_language_guidance_for_code",
		"add_format_guidance_for_explanations",
		"truncate_excessive_context",
		"remove_redundant_whitespace",
		"add_conciseness_directive",
	} {
		if containsRule(result.RulesApplied, rule) {
			t.Errorf("rule %q should not fire on a well-formed prompt; fired rules: %v",
				rule, result.RulesApplied)
		}
	}
}

func TestOptimizer_TruncatedPromptHasNegativeTokenDelta(t *testing.T) {
	o := newTestOptimizer()
	longPrompt := strings.Repeat("analyze this enormous document carefully ", 200) // ~8000 chars
	result, err := o.OptimizeWithResult(context.Background(), longPrompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TokenDelta >= 0 {
		t.Errorf("truncated prompt should have negative TokenDelta, got %d", result.TokenDelta)
	}
}

func TestOptimizer_TokenDeltaReflectsAppendedRules(t *testing.T) {
	o := newTestOptimizer()
	// Short explanation → appends guidance → TokenDelta should be positive (more tokens added).
	result, err := o.OptimizeWithResult(context.Background(), "what is redis")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TokenDelta <= 0 {
		t.Errorf("appended guidance should increase token count (positive delta), got %d", result.TokenDelta)
	}
}

func TestOptimizer_OriginalPreservedInResult(t *testing.T) {
	o := newTestOptimizer()
	original := "write code to solve this problem"
	result, err := o.OptimizeWithResult(context.Background(), original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Original != original {
		t.Errorf("Original field modified: got %q, want %q", result.Original, original)
	}
}

// containsRule checks whether a specific rule name is in the applied list.
func containsRule(applied []string, name string) bool {
	for _, r := range applied {
		if r == name {
			return true
		}
	}
	return false
}
