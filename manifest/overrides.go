package manifest

// Overrides is the shared shape used by both Manifest.Defaults and per-doc
// override fields embedded into DocEntry. Pointer types distinguish "absent"
// (nil) from "explicitly set to zero value" (e.g. *false).
type Overrides struct {
	Model              string `yaml:"model,omitempty"`
	ModelParams        string `yaml:"model_params,omitempty"`
	Language           string `yaml:"language,omitempty"`
	SystemPrompt       string `yaml:"system_prompt,omitempty"`
	Prompt             string `yaml:"prompt,omitempty"`
	ASTContext         *bool  `yaml:"ast_context,omitempty"`
	CodeSearch         *bool  `yaml:"code_search,omitempty"`
	CodeSearchMaxTurns *int   `yaml:"code_search_max_turns,omitempty"`
	RepoMeta           string `yaml:"repo_meta,omitempty"`
	TokenBudget        *int   `yaml:"token_budget,omitempty"`
	WatermarksFile     string `yaml:"watermarks_file,omitempty"`
}
