package effective_test

import (
	"testing"
	"time"

	"github.com/inhuman/memorialiste/cliconfig"
	"github.com/inhuman/memorialiste/effective"
	"github.com/inhuman/memorialiste/manifest"
	"github.com/stretchr/testify/assert"
)

func ptrBool(b bool) *bool { return &b }
func ptrInt(i int) *int    { return &i }

func baseCfg() *cliconfig.Config {
	return &cliconfig.Config{
		Model:              "cfg-model",
		Language:           "english",
		TokenBudget:        12000,
		ASTContext:         false,
		CodeSearch:         false,
		CodeSearchMaxTurns: 10,
		RepoMeta:           "basic",
		LLMTimeout:         5 * time.Minute,
	}
}

func TestResolve_LLMTimeout_OnlyCfg(t *testing.T) {
	cfg := baseCfg()
	eff := effective.Resolve(cfg, nil, &manifest.Manifest{}, manifest.DocEntry{})
	assert.Equal(t, 5*time.Minute, eff.LLMTimeout)
}

func TestResolve_LLMTimeout_DefaultsApply(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{Defaults: manifest.Overrides{LLMTimeout: "30s"}}
	eff := effective.Resolve(cfg, nil, m, manifest.DocEntry{})
	assert.Equal(t, 30*time.Second, eff.LLMTimeout)
}

func TestResolve_LLMTimeout_PerDocOverridesDefaults(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{Defaults: manifest.Overrides{LLMTimeout: "30s"}}
	entry := manifest.DocEntry{Overrides: manifest.Overrides{LLMTimeout: "2m"}}
	eff := effective.Resolve(cfg, nil, m, entry)
	assert.Equal(t, 2*time.Minute, eff.LLMTimeout)
}

func TestResolve_LLMTimeout_EnvOverridesManifest(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{Defaults: manifest.Overrides{LLMTimeout: "30s"}}
	t.Setenv("MEMORIALISTE_LLM_TIMEOUT", "45s")
	eff := effective.Resolve(cfg, nil, m, manifest.DocEntry{})
	assert.Equal(t, 45*time.Second, eff.LLMTimeout)
}

func TestResolve_LLMTimeout_CLIExplicitWins(t *testing.T) {
	cfg := baseCfg()
	cfg.LLMTimeout = 10 * time.Minute
	m := &manifest.Manifest{Defaults: manifest.Overrides{LLMTimeout: "30s"}}
	entry := manifest.DocEntry{Overrides: manifest.Overrides{LLMTimeout: "2m"}}
	t.Setenv("MEMORIALISTE_LLM_TIMEOUT", "45s")
	eff := effective.Resolve(cfg, effective.CLIExplicit{"llm-timeout": true}, m, entry)
	assert.Equal(t, 10*time.Minute, eff.LLMTimeout)
}

func TestResolve_LLMTimeout_InvalidManifestFallsThrough(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{Defaults: manifest.Overrides{LLMTimeout: "garbage"}}
	eff := effective.Resolve(cfg, nil, m, manifest.DocEntry{})
	assert.Equal(t, 5*time.Minute, eff.LLMTimeout)
}

func TestDetectCLIExplicit(t *testing.T) {
	got := effective.DetectCLIExplicit([]string{"--model=m1", "--ast-context", "--language", "ru", "positional"})
	assert.True(t, got["model"])
	assert.True(t, got["ast-context"])
	assert.True(t, got["language"])
	assert.False(t, got["prompt"])
}

func TestResolve_Model_OnlyCfg(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{}
	eff := effective.Resolve(cfg, effective.CLIExplicit{}, m, manifest.DocEntry{})
	assert.Equal(t, "cfg-model", eff.Model)
}

func TestResolve_Model_DefaultsApply(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{Defaults: manifest.Overrides{Model: "m1"}}
	eff := effective.Resolve(cfg, effective.CLIExplicit{}, m, manifest.DocEntry{})
	assert.Equal(t, "m1", eff.Model)
}

func TestResolve_Model_PerDocOverrides(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{Defaults: manifest.Overrides{Model: "m1"}}
	entry := manifest.DocEntry{Overrides: manifest.Overrides{Model: "m2"}}
	eff := effective.Resolve(cfg, effective.CLIExplicit{}, m, entry)
	assert.Equal(t, "m2", eff.Model)
}

func TestResolve_Model_EnvOverrides(t *testing.T) {
	t.Setenv("MEMORIALISTE_MODEL", "m3")
	cfg := baseCfg()
	m := &manifest.Manifest{Defaults: manifest.Overrides{Model: "m1"}}
	entry := manifest.DocEntry{Overrides: manifest.Overrides{Model: "m2"}}
	eff := effective.Resolve(cfg, effective.CLIExplicit{}, m, entry)
	assert.Equal(t, "m3", eff.Model)
}

func TestResolve_Model_CLIExplicitWins(t *testing.T) {
	t.Setenv("MEMORIALISTE_MODEL", "m3")
	cfg := baseCfg()
	cfg.Model = "m4"
	m := &manifest.Manifest{Defaults: manifest.Overrides{Model: "m1"}}
	entry := manifest.DocEntry{Overrides: manifest.Overrides{Model: "m2"}}
	eff := effective.Resolve(cfg, effective.CLIExplicit{"model": true}, m, entry)
	assert.Equal(t, "m4", eff.Model)
}

func TestResolve_DefaultsApply(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{Defaults: manifest.Overrides{ASTContext: ptrBool(true)}}
	eff := effective.Resolve(cfg, effective.CLIExplicit{}, m, manifest.DocEntry{})
	assert.True(t, eff.ASTContext)
}

func TestResolve_PerDocOverridesDefaults(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{Defaults: manifest.Overrides{ASTContext: ptrBool(true)}}
	entry := manifest.DocEntry{Overrides: manifest.Overrides{ASTContext: ptrBool(false)}}
	eff := effective.Resolve(cfg, effective.CLIExplicit{}, m, entry)
	assert.False(t, eff.ASTContext)
}

func TestResolve_DefaultsEmpty(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{}
	eff := effective.Resolve(cfg, effective.CLIExplicit{}, m, manifest.DocEntry{})
	assert.Equal(t, cfg.ASTContext, eff.ASTContext)
	assert.Equal(t, cfg.Model, eff.Model)
}

func TestResolve_FalseBoolIsExplicit(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{Defaults: manifest.Overrides{ASTContext: ptrBool(true)}}
	entry := manifest.DocEntry{Overrides: manifest.Overrides{ASTContext: ptrBool(false)}}
	eff := effective.Resolve(cfg, effective.CLIExplicit{}, m, entry)
	assert.False(t, eff.ASTContext)
}

func TestResolve_NilBoolIsAbsent(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{Defaults: manifest.Overrides{ASTContext: ptrBool(true)}}
	entry := manifest.DocEntry{}
	eff := effective.Resolve(cfg, effective.CLIExplicit{}, m, entry)
	assert.True(t, eff.ASTContext)
}

func TestResolve_BackwardCompat(t *testing.T) {
	cfg := baseCfg()
	cfg.ModelParams = ""
	cfg.SystemPrompt = ""
	cfg.Prompt = ""
	m := &manifest.Manifest{}
	entry := manifest.DocEntry{Path: "docs/x.md", Covers: []string{"a/"}}
	eff := effective.Resolve(cfg, effective.CLIExplicit{}, m, entry)
	assert.Equal(t, cfg.Model, eff.Model)
	assert.Equal(t, cfg.ModelParams, eff.ModelParams)
	assert.Equal(t, cfg.Language, eff.Language)
	assert.Equal(t, cfg.SystemPrompt, eff.SystemPrompt)
	assert.Equal(t, cfg.Prompt, eff.Prompt)
	assert.Equal(t, cfg.ASTContext, eff.ASTContext)
	assert.Equal(t, cfg.CodeSearch, eff.CodeSearch)
	assert.Equal(t, cfg.CodeSearchMaxTurns, eff.CodeSearchMaxTurns)
	assert.Equal(t, cfg.RepoMeta, eff.RepoMeta)
	assert.Equal(t, cfg.TokenBudget, eff.TokenBudget)
}

func TestResolve_LanguagePerDoc(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{}
	e1 := manifest.DocEntry{Overrides: manifest.Overrides{Language: "ru"}}
	e2 := manifest.DocEntry{Overrides: manifest.Overrides{Language: "en"}}
	assert.Equal(t, "ru", effective.Resolve(cfg, nil, m, e1).Language)
	assert.Equal(t, "en", effective.Resolve(cfg, nil, m, e2).Language)
}

func TestResolve_SystemPromptPerDoc(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{}
	entry := manifest.DocEntry{Overrides: manifest.Overrides{SystemPrompt: "@./prompt.md"}}
	eff := effective.Resolve(cfg, nil, m, entry)
	assert.Equal(t, "@./prompt.md", eff.SystemPrompt)
}

func TestResolve_TokenBudgetInt(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{Defaults: manifest.Overrides{TokenBudget: ptrInt(8000)}}
	entry := manifest.DocEntry{Overrides: manifest.Overrides{TokenBudget: ptrInt(16000)}}
	assert.Equal(t, 16000, effective.Resolve(cfg, nil, m, entry).TokenBudget)
	assert.Equal(t, 8000, effective.Resolve(cfg, nil, m, manifest.DocEntry{}).TokenBudget)
}

func TestDiff_NoChanges(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{}
	eff := effective.Resolve(cfg, nil, m, manifest.DocEntry{})
	assert.Equal(t, "(global)", eff.Diff(cfg))
}

func TestDiff_SeveralChanges(t *testing.T) {
	cfg := baseCfg()
	m := &manifest.Manifest{Defaults: manifest.Overrides{Model: "m1", ASTContext: ptrBool(true)}}
	eff := effective.Resolve(cfg, nil, m, manifest.DocEntry{})
	d := eff.Diff(cfg)
	assert.Contains(t, d, "model=m1")
	assert.Contains(t, d, "ast_context=true")
}
