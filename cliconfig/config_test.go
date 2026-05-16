package cliconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/kong"
)

func mapGetenv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestParse_DefaultsApply(t *testing.T) {
	cfg, err := Parse(nil, func(string) string { return "" })
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.ProviderURL != "http://localhost:11434" {
		t.Errorf("ProviderURL default: got %q", cfg.ProviderURL)
	}
	if cfg.Model != "qwen3-coder:30b" {
		t.Errorf("Model default: got %q", cfg.Model)
	}
	if cfg.TokenBudget != 12000 {
		t.Errorf("TokenBudget default: got %d", cfg.TokenBudget)
	}
	if !cfg.DryRun {
		t.Errorf("DryRun default: got false, expected true")
	}
	if cfg.Platform != "gitlab" {
		t.Errorf("Platform default: got %q", cfg.Platform)
	}
	if cfg.BaseBranch != "main" {
		t.Errorf("BaseBranch default: got %q", cfg.BaseBranch)
	}
	if cfg.ASTContext {
		t.Errorf("ASTContext default: got true, expected false")
	}
	if cfg.Language != "english" {
		t.Errorf("Language default: got %q", cfg.Language)
	}
	if cfg.LLMTimeout != 5*time.Minute {
		t.Errorf("LLMTimeout default: got %v, expected 5m", cfg.LLMTimeout)
	}
	if cfg.PlatformTimeout != 60*time.Second {
		t.Errorf("PlatformTimeout default: got %v, expected 60s", cfg.PlatformTimeout)
	}
	if cfg.ASTParseTimeout != 5*time.Second {
		t.Errorf("ASTParseTimeout default: got %v, expected 5s", cfg.ASTParseTimeout)
	}
}

func TestParse_TimeoutEnvOverrides(t *testing.T) {
	env := map[string]string{
		"MEMORIALISTE_LLM_TIMEOUT":       "90s",
		"MEMORIALISTE_PLATFORM_TIMEOUT":  "2m",
		"MEMORIALISTE_AST_PARSE_TIMEOUT": "1s",
	}
	cfg, err := Parse(nil, mapGetenv(env))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.LLMTimeout != 90*time.Second {
		t.Errorf("LLMTimeout: got %v", cfg.LLMTimeout)
	}
	if cfg.PlatformTimeout != 2*time.Minute {
		t.Errorf("PlatformTimeout: got %v", cfg.PlatformTimeout)
	}
	if cfg.ASTParseTimeout != time.Second {
		t.Errorf("ASTParseTimeout: got %v", cfg.ASTParseTimeout)
	}
}

func TestParse_EnvOverridesDefault(t *testing.T) {
	env := map[string]string{
		"MEMORIALISTE_PROVIDER_URL":  "http://provider.test",
		"MEMORIALISTE_MODEL":         "mymodel",
		"MEMORIALISTE_TOKEN_BUDGET":  "8000",
		"MEMORIALISTE_DRY_RUN":       "false",
		"MEMORIALISTE_AST_CONTEXT":   "true",
		"MEMORIALISTE_PLATFORM":      "github",
		"MEMORIALISTE_PROJECT_ID":    "acme/repo",
		"MEMORIALISTE_PLATFORM_TOKEN": "envtok",
	}
	cfg, err := Parse(nil, mapGetenv(env))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.ProviderURL != "http://provider.test" {
		t.Errorf("ProviderURL: got %q", cfg.ProviderURL)
	}
	if cfg.Model != "mymodel" {
		t.Errorf("Model: got %q", cfg.Model)
	}
	if cfg.TokenBudget != 8000 {
		t.Errorf("TokenBudget: got %d", cfg.TokenBudget)
	}
	if cfg.DryRun {
		t.Errorf("DryRun: got true, expected false")
	}
	if !cfg.ASTContext {
		t.Errorf("ASTContext: got false, expected true")
	}
	if cfg.Platform != "github" {
		t.Errorf("Platform: got %q", cfg.Platform)
	}
	if cfg.ProjectID != "acme/repo" {
		t.Errorf("ProjectID: got %q", cfg.ProjectID)
	}
	if cfg.PlatformToken != "envtok" {
		t.Errorf("PlatformToken: got %q", cfg.PlatformToken)
	}
}

func TestParse_FlagOverridesEnv(t *testing.T) {
	env := map[string]string{
		"MEMORIALISTE_MODEL":        "env-model",
		"MEMORIALISTE_TOKEN_BUDGET": "1000",
		"MEMORIALISTE_DRY_RUN":      "false",
	}
	cfg, err := Parse([]string{
		"--model=flag-model",
		"--token-budget=5000",
		"--dry-run=true",
	}, mapGetenv(env))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Model != "flag-model" {
		t.Errorf("Model: got %q", cfg.Model)
	}
	if cfg.TokenBudget != 5000 {
		t.Errorf("TokenBudget: got %d", cfg.TokenBudget)
	}
	if !cfg.DryRun {
		t.Errorf("DryRun: expected true (flag wins)")
	}
}

func TestParse_EmptyEnvTreatedAsUnset(t *testing.T) {
	env := map[string]string{"MEMORIALISTE_MODEL": ""}
	cfg, err := Parse(nil, mapGetenv(env))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.Model != "qwen3-coder:30b" {
		t.Errorf("expected default when env is empty, got %q", cfg.Model)
	}
}

func TestParse_BadIntEnv(t *testing.T) {
	env := map[string]string{"MEMORIALISTE_TOKEN_BUDGET": "not-a-number"}
	_, err := parseWithOptions(nil, mapGetenv(env), kong.Exit(func(int) {}))
	if err == nil {
		t.Fatal("expected error for non-integer token-budget env, got nil")
	}
	if !strings.Contains(err.Error(), "token-budget") && !strings.Contains(err.Error(), "TOKEN_BUDGET") {
		t.Errorf("expected error to mention the offending var, got %v", err)
	}
}

func TestParse_BadBoolEnv(t *testing.T) {
	env := map[string]string{"MEMORIALISTE_DRY_RUN": "invalid-bool"}
	_, err := parseWithOptions(nil, mapGetenv(env), kong.Exit(func(int) {}))
	if err == nil {
		t.Fatal("expected error for invalid bool, got nil")
	}
	if !strings.Contains(err.Error(), "dry-run") && !strings.Contains(err.Error(), "DRY_RUN") {
		t.Errorf("expected error to mention the offending var, got %v", err)
	}
}

func TestValidate_DryRunBypassesPlatformReqs(t *testing.T) {
	cfg := &Config{Platform: "gitlab", TokenBudget: 100, DryRun: true}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected nil error for dry-run with no token, got %v", err)
	}
}

func TestValidate_NonDryRunRequiresToken(t *testing.T) {
	cfg := &Config{Platform: "gitlab", TokenBudget: 100, DryRun: false, ProjectID: "x"}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--platform-token") || !strings.Contains(msg, "MEMORIALISTE_PLATFORM_TOKEN") {
		t.Errorf("expected message about platform-token, got: %s", msg)
	}
}

func TestValidate_NonDryRunRequiresProjectID(t *testing.T) {
	cfg := &Config{Platform: "gitlab", TokenBudget: 100, DryRun: false, PlatformToken: "t"}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--project-id") || !strings.Contains(msg, "MEMORIALISTE_PROJECT_ID") {
		t.Errorf("expected message about project-id, got: %s", msg)
	}
}

func TestValidate_AggregatesBoth(t *testing.T) {
	cfg := &Config{Platform: "gitlab", TokenBudget: 100, DryRun: false}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--platform-token") || !strings.Contains(msg, "--project-id") {
		t.Errorf("expected aggregated message, got: %s", msg)
	}
}

func TestParse_CodeSearchDefaults(t *testing.T) {
	cfg, err := Parse(nil, func(string) string { return "" })
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.CodeSearch {
		t.Errorf("CodeSearch default: want false")
	}
	if cfg.CodeSearchMaxTurns != 10 {
		t.Errorf("CodeSearchMaxTurns default: got %d, want 10", cfg.CodeSearchMaxTurns)
	}
}

func TestParse_CodeSearchEnvOverride(t *testing.T) {
	env := map[string]string{"MEMORIALISTE_CODE_SEARCH": "true"}
	cfg, err := Parse(nil, mapGetenv(env))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !cfg.CodeSearch {
		t.Errorf("expected CodeSearch=true from env")
	}
}

func TestParse_CodeSearchMaxTurnsFlag(t *testing.T) {
	cfg, err := Parse([]string{"--code-search-max-turns=5"}, func(string) string { return "" })
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.CodeSearchMaxTurns != 5 {
		t.Errorf("CodeSearchMaxTurns: got %d, want 5", cfg.CodeSearchMaxTurns)
	}
}

func TestValidate_BadPlatform(t *testing.T) {
	cfg := &Config{Platform: "bitbucket", TokenBudget: 100, DryRun: true}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "bitbucket") {
		t.Errorf("expected message to name value, got %s", err.Error())
	}
}

func TestValidate_TokenBudget(t *testing.T) {
	for _, v := range []int{-1, 0} {
		cfg := &Config{Platform: "gitlab", TokenBudget: v, DryRun: true}
		if err := cfg.Validate(); err == nil {
			t.Errorf("expected error for TokenBudget=%d", v)
		}
	}
}

func TestValidate_SystemPromptFileMissing(t *testing.T) {
	cfg := &Config{Platform: "gitlab", TokenBudget: 100, DryRun: true, SystemPrompt: "@/nonexistent/zzz/prompt.txt"}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing system-prompt file")
	}
	if !strings.Contains(err.Error(), "/nonexistent/zzz/prompt.txt") {
		t.Errorf("expected message to name path, got %s", err.Error())
	}
}

func TestValidate_SystemPromptFileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.txt")
	if err := os.WriteFile(path, []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{Platform: "gitlab", TokenBudget: 100, DryRun: true, SystemPrompt: "@" + path}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestParse_RepoMetaDefault(t *testing.T) {
	cfg, err := Parse(nil, func(string) string { return "" })
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.RepoMeta != "basic" {
		t.Errorf("RepoMeta default: got %q, want %q", cfg.RepoMeta, "basic")
	}
}

func TestParse_RepoMetaExtendedFlag(t *testing.T) {
	cfg, err := Parse([]string{"--repo-meta=extended"}, func(string) string { return "" })
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.RepoMeta != "extended" {
		t.Errorf("RepoMeta: got %q, want %q", cfg.RepoMeta, "extended")
	}
}

func TestParse_RepoMetaInvalidFlag(t *testing.T) {
	_, err := parseWithOptions([]string{"--repo-meta=garbage"}, func(string) string { return "" }, kong.Exit(func(int) {}), kong.Writers(nil, nil))
	if err == nil {
		t.Fatal("expected parse error for invalid --repo-meta value")
	}
}

func TestParse_RepoMetaEnv(t *testing.T) {
	env := map[string]string{"MEMORIALISTE_REPO_META": "extended"}
	cfg, err := Parse(nil, mapGetenv(env))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.RepoMeta != "extended" {
		t.Errorf("RepoMeta: got %q, want extended", cfg.RepoMeta)
	}
}

func TestValidate_NoTokenLeakage(t *testing.T) {
	const secret = "SUPER-SECRET-XYZ"
	cfg := &Config{
		Platform:      "gitlab",
		TokenBudget:   100,
		DryRun:        false,
		PlatformToken: secret,
		// ProjectID missing triggers error
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("token leaked in validation message: %s", err.Error())
	}

	// also verify with missing platform too
	cfg2 := &Config{Platform: "bogus", TokenBudget: -1, DryRun: false, PlatformToken: secret}
	err2 := cfg2.Validate()
	if err2 == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err2.Error(), secret) {
		t.Errorf("token leaked: %s", err2.Error())
	}
}

func TestParse_WatermarksFile_Default(t *testing.T) {
	cfg, err := Parse(nil, func(string) string { return "" })
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.WatermarksFile != "" {
		t.Errorf("WatermarksFile default: got %q, want empty", cfg.WatermarksFile)
	}
}

func TestParse_WatermarksFile_Env(t *testing.T) {
	env := map[string]string{"MEMORIALISTE_WATERMARKS_FILE": ".wm.yaml"}
	cfg, err := Parse(nil, mapGetenv(env))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.WatermarksFile != ".wm.yaml" {
		t.Errorf("WatermarksFile env: got %q, want %q", cfg.WatermarksFile, ".wm.yaml")
	}
}

func TestParse_WatermarksFile_Flag(t *testing.T) {
	cfg, err := Parse([]string{"--watermarks-file=.flag.yaml"}, func(string) string { return "" })
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.WatermarksFile != ".flag.yaml" {
		t.Errorf("WatermarksFile flag: got %q, want %q", cfg.WatermarksFile, ".flag.yaml")
	}
}
