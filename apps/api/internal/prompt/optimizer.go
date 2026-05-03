package prompt

import (
	"context"
	"strings"
)

const phi3ConfidenceThreshold = 0.7

// Optimizer applies rule-based prompt rewriting. If rule confidence falls
// below the threshold, it delegates to the Phi-3 sidecar via Phi3Client.
// The sidecar is used for < 10% of requests — keeping inference overhead minimal.
type Optimizer struct {
	phi3Client *Phi3Client
}

func NewOptimizer(phi3SidecarURL string) *Optimizer {
	return &Optimizer{
		phi3Client: NewPhi3Client(phi3SidecarURL),
	}
}

// Optimize returns the rewritten prompt string. It is a convenience wrapper
// around OptimizeWithResult for callers that only need the final text.
func (o *Optimizer) Optimize(ctx context.Context, raw string) (string, error) {
	result, err := o.OptimizeWithResult(ctx, raw)
	return result.Optimized, err
}

// OptimizeWithResult runs the full optimization pipeline and returns the
// detailed OptimizationResult, including which rules fired and the token delta.
func (o *Optimizer) OptimizeWithResult(ctx context.Context, raw string) (OptimizationResult, error) {
	result := o.applyRules(raw)

	if result.Confidence < phi3ConfidenceThreshold && o.phi3Client != nil {
		phi3Result, err := o.phi3Client.Refine(ctx, raw)
		if err == nil && phi3Result != "" {
			result.Optimized = phi3Result
			result.UsedPhi3 = true
			result.TokenDelta = estimateTokens(phi3Result) - estimateTokens(raw)
		}
		// Phi-3 failure is non-fatal: return rule-based result unchanged.
	}

	return result, nil
}

// applyRules runs all rules in order and returns an OptimizationResult.
func (o *Optimizer) applyRules(raw string) OptimizationResult {
	current := raw
	applied := make([]string, 0, 2)

	for _, rule := range rules {
		if rule.Applies(current) {
			current = rule.Optimize(current)
			applied = append(applied, rule.Name)
		}
	}

	return OptimizationResult{
		Original:     raw,
		Optimized:    current,
		RulesApplied: applied,
		TokenDelta:   estimateTokens(current) - estimateTokens(raw),
		UsedPhi3:     false,
		Confidence:   estimateConfidence(raw, current, applied),
	}
}

// estimateConfidence returns a heuristic confidence score for the optimization.
// Clear, specific prompts score high. Ambiguous short prompts score lower.
func estimateConfidence(original, optimized string, appliedRules []string) float64 {
	confidence := 0.8

	length := len(strings.TrimSpace(original))

	if length < 20 {
		confidence -= 0.2
	} else if length > 200 {
		confidence += 0.1
	}

	if len(appliedRules) > 2 {
		confidence -= 0.1
	}

	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}
	return confidence
}

// estimateTokens returns a rough token count: 1 token ≈ 4 bytes for English text.
func estimateTokens(text string) int {
	return (len(text) + 3) / 4
}
