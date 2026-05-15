package effective

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/inhuman/memorialiste/cliconfig"
	"github.com/inhuman/memorialiste/manifest"
)

// Effective is the fully-resolved per-entry configuration. Every field is
// final: no nils, no zero-as-absent ambiguity.
type Effective struct {
	Model              string
	ModelParams        string
	Language           string
	SystemPrompt       string
	Prompt             string
	ASTContext         bool
	CodeSearch         bool
	CodeSearchMaxTurns int
	RepoMeta           string
	TokenBudget        int
	WatermarksFile     string
}

// CLIExplicit names the flags explicitly supplied on the command line
// (without the leading "--"). Used to promote CLI values above manifest
// layers per FR-006.
type CLIExplicit map[string]bool

// DetectCLIExplicit inspects argv and returns the set of flag names that
// appeared as --name, --name=value, or --name value. Names are stored
// without the leading dashes.
func DetectCLIExplicit(argv []string) CLIExplicit {
	out := CLIExplicit{}
	for i := 0; i < len(argv); i++ {
		tok := argv[i]
		if !strings.HasPrefix(tok, "--") {
			continue
		}
		name := strings.TrimPrefix(tok, "--")
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			name = name[:eq]
		}
		if name == "" {
			continue
		}
		out[name] = true
	}
	return out
}

// Resolve merges five layers per FR-005 (hard-coded < manifest defaults <
// manifest per-doc < env var < CLI flag) into a single Effective.
func Resolve(cfg *cliconfig.Config, cliExplicit CLIExplicit, m *manifest.Manifest, entry manifest.DocEntry) Effective {
	defaults := manifest.Overrides{}
	if m != nil {
		defaults = m.Defaults
	}
	entryOv := entry.Overrides

	return Effective{
		Model:              resolveString("model", "MEMORIALISTE_MODEL", cfg.Model, cliExplicit, entryOv.Model, defaults.Model),
		ModelParams:        resolveString("model-params", "MEMORIALISTE_MODEL_PARAMS", cfg.ModelParams, cliExplicit, entryOv.ModelParams, defaults.ModelParams),
		Language:           resolveString("language", "MEMORIALISTE_LANGUAGE", cfg.Language, cliExplicit, entryOv.Language, defaults.Language),
		SystemPrompt:       resolveString("system-prompt", "MEMORIALISTE_SYSTEM_PROMPT", cfg.SystemPrompt, cliExplicit, entryOv.SystemPrompt, defaults.SystemPrompt),
		Prompt:             resolveString("prompt", "MEMORIALISTE_PROMPT", cfg.Prompt, cliExplicit, entryOv.Prompt, defaults.Prompt),
		ASTContext:         resolveBool("ast-context", "MEMORIALISTE_AST_CONTEXT", cfg.ASTContext, cliExplicit, entryOv.ASTContext, defaults.ASTContext),
		CodeSearch:         resolveBool("code-search", "MEMORIALISTE_CODE_SEARCH", cfg.CodeSearch, cliExplicit, entryOv.CodeSearch, defaults.CodeSearch),
		CodeSearchMaxTurns: resolveInt("code-search-max-turns", "MEMORIALISTE_CODE_SEARCH_MAX_TURNS", cfg.CodeSearchMaxTurns, cliExplicit, entryOv.CodeSearchMaxTurns, defaults.CodeSearchMaxTurns),
		RepoMeta:           resolveString("repo-meta", "MEMORIALISTE_REPO_META", cfg.RepoMeta, cliExplicit, entryOv.RepoMeta, defaults.RepoMeta),
		TokenBudget:        resolveInt("token-budget", "MEMORIALISTE_TOKEN_BUDGET", cfg.TokenBudget, cliExplicit, entryOv.TokenBudget, defaults.TokenBudget),
		WatermarksFile:     resolveString("watermarks-file", "MEMORIALISTE_WATERMARKS_FILE", cfg.WatermarksFile, cliExplicit, entryOv.WatermarksFile, defaults.WatermarksFile),
	}
}

func resolveString(flag, envName, cfgVal string, cli CLIExplicit, entry, def string) string {
	if cli[flag] {
		return cfgVal
	}
	if v := os.Getenv(envName); v != "" {
		return v
	}
	if entry != "" {
		return entry
	}
	if def != "" {
		return def
	}
	return cfgVal
}

func resolveBool(flag, envName string, cfgVal bool, cli CLIExplicit, entry, def *bool) bool {
	if cli[flag] {
		return cfgVal
	}
	if v := os.Getenv(envName); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	if entry != nil {
		return *entry
	}
	if def != nil {
		return *def
	}
	return cfgVal
}

func resolveInt(flag, envName string, cfgVal int, cli CLIExplicit, entry, def *int) int {
	if cli[flag] {
		return cfgVal
	}
	if v := os.Getenv(envName); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	if entry != nil {
		return *entry
	}
	if def != nil {
		return *def
	}
	return cfgVal
}

// Diff returns a space-separated list of `field=value` for every Effective
// field that differs from cfg's corresponding value. Returns "(global)" when
// nothing differs.
func (eff Effective) Diff(cfg *cliconfig.Config) string {
	var parts []string
	if eff.Model != cfg.Model {
		parts = append(parts, "model="+eff.Model)
	}
	if eff.ModelParams != cfg.ModelParams {
		parts = append(parts, "model_params="+eff.ModelParams)
	}
	if eff.Language != cfg.Language {
		parts = append(parts, "language="+eff.Language)
	}
	if eff.SystemPrompt != cfg.SystemPrompt {
		parts = append(parts, "system_prompt="+eff.SystemPrompt)
	}
	if eff.Prompt != cfg.Prompt {
		parts = append(parts, "prompt="+eff.Prompt)
	}
	if eff.ASTContext != cfg.ASTContext {
		parts = append(parts, fmt.Sprintf("ast_context=%v", eff.ASTContext))
	}
	if eff.CodeSearch != cfg.CodeSearch {
		parts = append(parts, fmt.Sprintf("code_search=%v", eff.CodeSearch))
	}
	if eff.CodeSearchMaxTurns != cfg.CodeSearchMaxTurns {
		parts = append(parts, fmt.Sprintf("code_search_max_turns=%d", eff.CodeSearchMaxTurns))
	}
	if eff.RepoMeta != cfg.RepoMeta {
		parts = append(parts, "repo_meta="+eff.RepoMeta)
	}
	if eff.TokenBudget != cfg.TokenBudget {
		parts = append(parts, fmt.Sprintf("token_budget=%d", eff.TokenBudget))
	}
	if eff.WatermarksFile != cfg.WatermarksFile {
		parts = append(parts, "watermarks_file="+eff.WatermarksFile)
	}
	if len(parts) == 0 {
		return "(global)"
	}
	return strings.Join(parts, " ")
}
