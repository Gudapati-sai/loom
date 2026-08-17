package llm

// Static is the built-in question set used whenever no local model is
// available, or when the model's output fails validation twice in a row.
// This is what the PRD's fallback acceptance criterion depends on: the
// wizard is never blocked on the model being present or correct.
var Static = map[string]Question{
	"welcome_name": {
		Prompt:      "What's the project called?",
		FreeText:    true,
		AllowCustom: true,
	},
	"stack": {
		Prompt: "What's the primary language/stack?",
		Options: []Option{
			{Label: "Go", Explanation: "Compiled, single-binary, good fit for CLIs and services."},
			{Label: "Python", Explanation: "Fast to write, huge ecosystem, good fit for data/ML/scripting."},
			{Label: "TypeScript/Node", Explanation: "Good fit for web backends and full-stack JS projects."},
			{Label: "Other", Explanation: "Anything else — CI templates default to a generic shape you can adapt."},
		},
	},
	"feature_name": {
		Prompt:   "Name the first feature you're building (short, kebab-case is fine):",
		FreeText: true,
	},
	"feature_problem": {
		Prompt:   "In 1-2 sentences, what problem does this feature solve?",
		FreeText: true,
	},
	"feature_criteria": {
		Prompt:   "Give one concrete, testable acceptance criterion for it:",
		FreeText: true,
	},
}
