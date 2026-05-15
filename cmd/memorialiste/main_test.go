package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/inhuman/memorialiste/cliconfig"
	"github.com/inhuman/memorialiste/effective"
	"github.com/inhuman/memorialiste/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordedCall struct {
	model    string
	system   string
	language string
}

type recorder struct {
	mu    sync.Mutex
	calls []recordedCall
	// label correlates this provider instance with the entry it was built for.
	model    string
	system   string
	language string
}

func (r *recorder) Complete(_ context.Context, messages []provider.Message) (string, provider.TokenUsage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sys := ""
	for _, m := range messages {
		if m.Role == "system" {
			sys = m.Content
		}
	}
	r.calls = append(r.calls, recordedCall{model: r.model, system: sys, language: r.language})
	return "# Generated body", provider.TokenUsage{}, nil
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	repo, err := gogit.PlainInit(dir, false)
	require.NoError(t, err)
	wt, err := repo.Worktree()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# init\n"), 0o644))
	_, err = wt.Add("README.md")
	require.NoError(t, err)
	_, err = wt.Commit("init", &gogit.CommitOptions{
		Author: &object.Signature{Name: "x", Email: "x@x", When: time.Now()},
	})
	require.NoError(t, err)
	// Force a diff under "src/" for entries' covers.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src/foo.go"), []byte("package src\n\nfunc Foo() {}\n"), 0o644))
	_, err = wt.Add("src/foo.go")
	require.NoError(t, err)
	_, err = wt.Commit("add foo", &gogit.CommitOptions{
		Author: &object.Signature{Name: "x", Email: "x@x", When: time.Now()},
	})
	require.NoError(t, err)
	return dir
}

func writeManifest(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, ".docstructure.yaml")
	require.NoError(t, os.WriteFile(p, []byte(body), 0o644))
	return p
}

// withProviderFactory swaps the package-global newProviderFor for a given
// callback that returns a *recorder seeded with the per-entry settings.
func withProviderFactory(t *testing.T, factory func(cfg *cliconfig.Config, eff effective.Effective) provider.Provider) {
	t.Helper()
	prev := newProviderFor
	newProviderFor = factory
	t.Cleanup(func() { newProviderFor = prev })
}

func TestRun_PerEntryModelOverride(t *testing.T) {
	dir := initRepo(t)
	manifestPath := writeManifest(t, dir, `
docs:
  - path: docs/a.md
    covers: [src/]
    model: alpha-model
  - path: docs/b.md
    covers: [src/]
    model: beta-model
`)

	var mu sync.Mutex
	var calls []recordedCall

	withProviderFactory(t, func(cfg *cliconfig.Config, eff effective.Effective) provider.Provider {
		return &recorder{
			model:    eff.Model,
			system:   eff.SystemPrompt,
			language: eff.Language,
			calls:    nil,
			// share an aggregator via Complete that appends into outer slice
		}
	})
	// Wrap factory once more to capture into outer slice.
	withProviderFactory(t, func(cfg *cliconfig.Config, eff effective.Effective) provider.Provider {
		return &captureProvider{
			model:    eff.Model,
			system:   eff.SystemPrompt,
			language: eff.Language,
			mu:       &mu,
			out:      &calls,
		}
	})

	cfg := &cliconfig.Config{
		ProviderURL:        "http://localhost",
		Model:              "cfg-model",
		Language:           "english",
		DocStructure:       manifestPath,
		RepoPath:           dir,
		TokenBudget:        12000,
		DryRun:             true,
		BranchPrefix:       "docs/x-",
		RepoMeta:           "basic",
		CodeSearchMaxTurns: 10,
		Platform:           "gitlab",
		BaseBranch:         "main",
	}
	require.NoError(t, run(context.Background(), cfg))

	mu.Lock()
	defer mu.Unlock()
	models := map[string]bool{}
	for _, c := range calls {
		models[c.model] = true
	}
	assert.True(t, models["alpha-model"], "alpha-model not recorded; got %v", models)
	assert.True(t, models["beta-model"], "beta-model not recorded; got %v", models)
}

func TestRun_PerEntrySystemPrompt(t *testing.T) {
	dir := initRepo(t)
	p1 := filepath.Join(dir, "p1.md")
	p2 := filepath.Join(dir, "p2.md")
	require.NoError(t, os.WriteFile(p1, []byte("PROMPT-ONE for {language}"), 0o644))
	require.NoError(t, os.WriteFile(p2, []byte("PROMPT-TWO for {language}"), 0o644))
	manifestPath := writeManifest(t, dir, `
docs:
  - path: docs/a.md
    covers: [src/]
    system_prompt: "@`+p1+`"
  - path: docs/b.md
    covers: [src/]
    system_prompt: "@`+p2+`"
`)

	var mu sync.Mutex
	var calls []recordedCall
	withProviderFactory(t, func(cfg *cliconfig.Config, eff effective.Effective) provider.Provider {
		return &captureProvider{model: eff.Model, system: eff.SystemPrompt, language: eff.Language, mu: &mu, out: &calls}
	})

	cfg := &cliconfig.Config{
		Model: "m", Language: "english", DocStructure: manifestPath, RepoPath: dir,
		TokenBudget: 12000, DryRun: true, RepoMeta: "basic", CodeSearchMaxTurns: 10,
		Platform: "gitlab", BaseBranch: "main",
	}
	require.NoError(t, run(context.Background(), cfg))

	mu.Lock()
	defer mu.Unlock()
	systems := map[string]bool{}
	for _, c := range calls {
		systems[c.system] = true
	}
	assert.Contains(t, systems, "@"+p1)
	assert.Contains(t, systems, "@"+p2)
}

func TestMigration_FrontmatterToSidecar(t *testing.T) {
	dir := initRepo(t)
	docDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docDir, 0o755))
	// Pre-seed doc with a frontmatter watermark equal to HEAD so there's no diff.
	headSHA := resolveHEAD(t, dir)
	docContent := "---\ngenerated_at: " + headSHA + "\n---\n\n# A\n"
	require.NoError(t, os.WriteFile(filepath.Join(docDir, "a.md"), []byte(docContent), 0o644))

	manifestPath := writeManifest(t, dir, `
docs:
  - path: docs/a.md
    covers: [src/]
    watermarks_file: .w.yaml
`)

	calledWith := []string{}
	var mu sync.Mutex
	withProviderFactory(t, func(cfg *cliconfig.Config, eff effective.Effective) provider.Provider {
		return &captureProvider{model: eff.Model, mu: &mu, out: &[]recordedCall{}}
	})
	// Replace with a provider that records each Complete call.
	withProviderFactory(t, func(cfg *cliconfig.Config, eff effective.Effective) provider.Provider {
		return &funcProvider{onComplete: func(_ context.Context, _ []provider.Message) (string, provider.TokenUsage, error) {
			mu.Lock()
			calledWith = append(calledWith, eff.Model)
			mu.Unlock()
			return "X", provider.TokenUsage{}, nil
		}}
	})

	cfg := &cliconfig.Config{
		Model: "m", Language: "english", DocStructure: manifestPath, RepoPath: dir,
		TokenBudget: 12000, DryRun: true, RepoMeta: "basic", CodeSearchMaxTurns: 10,
		Platform: "gitlab", BaseBranch: "main",
	}
	require.NoError(t, run(context.Background(), cfg))

	mu.Lock()
	defer mu.Unlock()
	// The watermark came from frontmatter (sidecar didn't exist yet); diff is empty;
	// so the provider should NOT have been called for this entry.
	assert.Empty(t, calledWith, "no LLM call expected when watermark equals HEAD (frontmatter migration source)")
}

func TestMigration_SidecarToFrontmatter(t *testing.T) {
	dir := initRepo(t)
	docDir := filepath.Join(dir, "docs")
	require.NoError(t, os.MkdirAll(docDir, 0o755))
	// Doc has NO frontmatter; sidecar holds the watermark.
	require.NoError(t, os.WriteFile(filepath.Join(docDir, "a.md"), []byte("# A\n"), 0o644))
	headSHA := resolveHEAD(t, dir)
	sidecarPath := filepath.Join(dir, ".w.yaml")
	require.NoError(t, os.WriteFile(sidecarPath, []byte("- path: docs/a.md\n  generated_at: "+headSHA+"\n"), 0o644))

	// Manifest does NOT declare watermarks_file → must fall back to migrationSidecars.
	// But migrationSidecars is built from manifest entries. To exercise the reverse-
	// migration path we keep a *separate* entry that DOES declare the sidecar.
	manifestPath := writeManifest(t, dir, `
docs:
  - path: docs/a.md
    covers: [src/]
  - path: docs/b.md
    covers: [src/]
    watermarks_file: .w.yaml
`)

	var mu sync.Mutex
	called := map[string]bool{}
	withProviderFactory(t, func(cfg *cliconfig.Config, eff effective.Effective) provider.Provider {
		return &funcProvider{onComplete: func(_ context.Context, _ []provider.Message) (string, provider.TokenUsage, error) {
			mu.Lock()
			called[eff.Model] = true
			_ = called
			mu.Unlock()
			return "X", provider.TokenUsage{}, nil
		}}
	})
	// Track per-entry skip vs call by counting entries that pass through generate.
	calls := []string{}
	withProviderFactory(t, func(cfg *cliconfig.Config, eff effective.Effective) provider.Provider {
		return &funcProvider{onComplete: func(_ context.Context, _ []provider.Message) (string, provider.TokenUsage, error) {
			mu.Lock()
			calls = append(calls, eff.WatermarksFile)
			mu.Unlock()
			return "X", provider.TokenUsage{}, nil
		}}
	})

	cfg := &cliconfig.Config{
		Model: "m", Language: "english", DocStructure: manifestPath, RepoPath: dir,
		TokenBudget: 12000, DryRun: true, RepoMeta: "basic", CodeSearchMaxTurns: 10,
		Platform: "gitlab", BaseBranch: "main",
	}
	require.NoError(t, run(context.Background(), cfg))

	mu.Lock()
	defer mu.Unlock()
	// docs/a.md (frontmatter mode, missing frontmatter): the sidecar from docs/b.md
	// is consulted via migrationSidecars; record found, watermark == HEAD → no LLM call.
	// docs/b.md (sidecar mode): sidecar has no entry for it (just docs/a.md) → empty
	// watermark → LLM is called (this entry is brand-new).
	// So we expect exactly one call, and it is for the entry declaring the sidecar.
	require.Len(t, calls, 1, "expected exactly one LLM call (docs/b.md only); got %v", calls)
	assert.Equal(t, ".w.yaml", calls[0])
}

// ── helpers ───────────────────────────────────────────────────────────────────

type captureProvider struct {
	model, system, language string
	mu                      *sync.Mutex
	out                     *[]recordedCall
}

func (p *captureProvider) Complete(_ context.Context, messages []provider.Message) (string, provider.TokenUsage, error) {
	sys := ""
	for _, m := range messages {
		if m.Role == "system" {
			sys = m.Content
		}
	}
	_ = sys
	// Record the entry-bound system prompt (eff.SystemPrompt), not the loaded body,
	// so the test can assert routing rather than loadSystemPrompt's transformation.
	p.mu.Lock()
	*p.out = append(*p.out, recordedCall{model: p.model, system: p.system, language: p.language})
	p.mu.Unlock()
	return "# Body", provider.TokenUsage{}, nil
}

type funcProvider struct {
	onComplete func(ctx context.Context, messages []provider.Message) (string, provider.TokenUsage, error)
}

func (p *funcProvider) Complete(ctx context.Context, messages []provider.Message) (string, provider.TokenUsage, error) {
	return p.onComplete(ctx, messages)
}

func resolveHEAD(t *testing.T, dir string) string {
	t.Helper()
	repo, err := gogit.PlainOpen(dir)
	require.NoError(t, err)
	ref, err := repo.Head()
	require.NoError(t, err)
	return ref.Hash().String()
}
