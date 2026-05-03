package prompt

// OptimizationResult is returned by Optimizer.OptimizeWithResult.
type OptimizationResult struct {
	Original     string
	Optimized    string
	RulesApplied []string
	TokenDelta   int     // estimated token change; negative = fewer tokens = energy reduction
	UsedPhi3     bool    // true if the Phi-3 sidecar was invoked
	Confidence   float64
}
