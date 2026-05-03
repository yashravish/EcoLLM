package prompt

import (
	"strings"
	"unicode/utf8"
)

// OptimizationRule is a single rewriting rule applied to a prompt.
type OptimizationRule struct {
	Name     string
	Applies  func(prompt string) bool
	Optimize func(prompt string) string
}

// rules is the ordered list of rule-based optimizations.
// Rules are cheap (no ML), executed in order, and individually skippable.
var rules = []OptimizationRule{
	{
		// Fired when prompt requests code but doesn't specify the language.
		// Adding language specificity reduces back-and-forth = fewer total tokens.
		Name: "add_language_guidance_for_code",
		Applies: func(p string) bool {
			lower := strings.ToLower(p)
			return containsAny(lower, "write code", "code for", "script to", "function that") &&
				!containsAny(lower, "python", "go", "javascript", "typescript", "rust", "java", "c++")
		},
		Optimize: func(p string) string {
			return p + "\nPlease specify the programming language and include inline comments."
		},
	},
	{
		// Fired when a short explanation prompt lacks format guidance.
		// Appending "concise, structured" nudges shorter completions = less energy.
		Name: "add_format_guidance_for_explanations",
		Applies: func(p string) bool {
			lower := strings.ToLower(p)
			return containsAny(lower, "explain", "describe", "what is", "how does") &&
				utf8.RuneCountInString(p) < 50
		},
		Optimize: func(p string) string {
			return p + "\nProvide a concise, structured answer."
		},
	},
	{
		// Fired when the prompt exceeds 4000 chars (approximate 1000-token boundary).
		// Keeps intent at both ends; drops verbose middle context.
		Name: "truncate_excessive_context",
		Applies: func(p string) bool {
			return len(p) > 4000
		},
		Optimize: func(p string) string {
			return p[:2000] + "\n...[truncated for efficiency]...\n" + p[len(p)-2000:]
		},
	},
	{
		// Collapses triple+ newlines to double newline and trims leading/trailing whitespace.
		// Saves tokens and prevents vLLM from wasting context window on blank lines.
		Name: "remove_redundant_whitespace",
		Applies: func(p string) bool {
			return p != strings.TrimSpace(p) || strings.Contains(p, "\n\n\n")
		},
		Optimize: func(p string) string {
			for strings.Contains(p, "\n\n\n") {
				p = strings.ReplaceAll(p, "\n\n\n", "\n\n")
			}
			return strings.TrimSpace(p)
		},
	},
	{
		// Appended when the prompt carries no explicit length or format directive.
		// "Be concise." directly reduces output tokens, which is the primary energy lever.
		Name: "add_conciseness_directive",
		Applies: func(p string) bool {
			lower := strings.ToLower(p)
			// Don't fire if the prompt already contains a length/format signal.
			alreadyDirected := containsAny(lower,
				"be concise", "briefly", "in detail", "comprehensive",
				"word limit", "max words", "step by step", "explain thoroughly",
				"concise", "structured answer", "only the function", "return only",
				"just give", "only provide",
			)
			return !alreadyDirected
		},
		Optimize: func(p string) string {
			return p + "\nBe concise."
		},
	},
}

func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
