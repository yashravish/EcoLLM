package router

import "strings"

// Classifier performs lightweight heuristic task classification.
// No ML model is used — keyword matching + length heuristics keep classification
// overhead under 0.5 ms per request, making it suitable for the hot path.
type Classifier struct{}

func NewClassifier() *Classifier {
	return &Classifier{}
}

var hardKeywords = map[string]int{
	"debug":         3,
	"architect":     3,
	"design system": 3,
	"reason about":  3,
	"analyze":       2,
	"compare":       2,
	"optimize":      2,
	"refactor":      2,
}

var mediumKeywords = map[string]int{
	"write":     1,
	"code":      1,
	"generate":  1,
	"summarize": 1,
	"translate": 1,
	"explain":   1,
}

var simpleKeywords = []string{
	"classify", "extract", "detect", "list", "find", "what is", "who is",
}

// Classify returns a ClassificationResult for the given prompt.
// Safe for concurrent use from multiple goroutines.
func (c *Classifier) Classify(prompt string) ClassificationResult {
	complexity := 1
	signals := make([]string, 0, 4)
	lower := strings.ToLower(prompt)

	if len(prompt) > 500 {
		complexity += 2
		signals = append(signals, "long_prompt")
	}
	if len(prompt) > 1500 {
		complexity += 2
		signals = append(signals, "very_long_prompt")
	}

	for keyword, weight := range hardKeywords {
		if strings.Contains(lower, keyword) {
			complexity += weight
			signals = append(signals, "hard:"+keyword)
		}
	}

	for keyword, weight := range mediumKeywords {
		if strings.Contains(lower, keyword) {
			complexity += weight
			signals = append(signals, "medium:"+keyword)
		}
	}

	for _, keyword := range simpleKeywords {
		if strings.Contains(lower, keyword) {
			signals = append(signals, "simple:"+keyword)
		}
	}

	if strings.Contains(lower, "step by step") ||
		(strings.Contains(lower, "first") && strings.Contains(lower, "then")) {
		complexity += 2
		signals = append(signals, "multi_step")
	}

	if complexity > 10 {
		complexity = 10
	}

	taskType := complexityToTaskType(complexity)

	confidence := 0.8
	if len(signals) >= 3 {
		confidence = 0.9
	}

	return ClassificationResult{
		TaskType:   taskType,
		Complexity: complexity,
		Confidence: confidence,
		Signals:    signals,
	}
}

func complexityToTaskType(complexity int) TaskType {
	switch {
	case complexity <= 3:
		return TaskSimple
	case complexity <= 6:
		return TaskMedium
	case complexity <= 9:
		return TaskHard
	default:
		return TaskSpecialized
	}
}
