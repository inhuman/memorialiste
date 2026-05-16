package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/inhuman/memorialiste/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeManifest(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "docstructure*.yaml")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func TestParse_ValidTwoEntries(t *testing.T) {
	path := writeManifest(t, `
docs:
  - path: docs/user-guide.md
    audience: end users
    covers:
      - internal/agent/
      - cmd/
    description: User guide.
  - path: docs/ops.md
    audience: operators
    covers:
      - internal/config/
`)
	m, err := manifest.Parse(path)
	require.NoError(t, err)
	require.Len(t, m.Docs, 2)

	assert.Equal(t, "docs/user-guide.md", m.Docs[0].Path)
	assert.Equal(t, []string{"internal/agent/", "cmd/"}, m.Docs[0].Covers)
	assert.Equal(t, "end users", m.Docs[0].Audience)

	assert.Equal(t, "docs/ops.md", m.Docs[1].Path)
	assert.Equal(t, []string{"internal/config/"}, m.Docs[1].Covers)
}

func TestParse_MissingPath(t *testing.T) {
	path := writeManifest(t, `
docs:
  - covers:
      - internal/agent/
`)
	_, err := manifest.Parse(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "entry[0].path is required")
}

func TestParse_EmptyCovers(t *testing.T) {
	path := writeManifest(t, `
docs:
  - path: docs/user-guide.md
    covers: []
`)
	_, err := manifest.Parse(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "entry[0].covers must not be empty")
}

func TestParse_NoDocs(t *testing.T) {
	path := writeManifest(t, `docs: []`)
	_, err := manifest.Parse(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no doc entries defined")
}

func TestParse_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.yaml")
	_, err := manifest.Parse(path)
	require.Error(t, err)
	assert.ErrorIs(t, err, manifest.ErrManifestNotFound)
	assert.Contains(t, err.Error(), path)
	assert.Contains(t, err.Error(), "--doc-structure")
}

// Backward compatibility for v0.3.x manifests is guaranteed by embedding
// Overrides into DocEntry with `yaml:",inline"` and adding a separate
// optional Defaults block — existing manifests parse unchanged.

func TestParse_DefaultsBlock(t *testing.T) {
	path := writeManifest(t, `
defaults:
  model: m1
  ast_context: true
  token_budget: 9000
docs:
  - path: docs/guide.md
    covers: [internal/]
`)
	m, err := manifest.Parse(path)
	require.NoError(t, err)
	assert.Equal(t, "m1", m.Defaults.Model)
	require.NotNil(t, m.Defaults.ASTContext)
	assert.True(t, *m.Defaults.ASTContext)
	require.NotNil(t, m.Defaults.TokenBudget)
	assert.Equal(t, 9000, *m.Defaults.TokenBudget)
}

func TestParse_PerDocOverrides(t *testing.T) {
	path := writeManifest(t, `
docs:
  - path: docs/guide.md
    covers: [internal/]
    model: m2
    ast_context: false
`)
	m, err := manifest.Parse(path)
	require.NoError(t, err)
	require.Len(t, m.Docs, 1)
	assert.Equal(t, "m2", m.Docs[0].Model)
	require.NotNil(t, m.Docs[0].ASTContext)
	assert.False(t, *m.Docs[0].ASTContext)
}

func TestParse_TypeMismatch(t *testing.T) {
	path := writeManifest(t, `
docs:
  - path: docs/guide.md
    covers: [internal/]
    ast_context: "true"
`)
	_, err := manifest.Parse(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse error")
}

func TestParse_InvalidRepoMeta(t *testing.T) {
	path := writeManifest(t, `
docs:
  - path: docs/guide.md
    covers: [internal/]
    repo_meta: verbose
`)
	_, err := manifest.Parse(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repo_meta")
	assert.Contains(t, err.Error(), "docs/guide.md")
}

func TestParse_InvalidTokenBudget(t *testing.T) {
	path := writeManifest(t, `
docs:
  - path: docs/guide.md
    covers: [internal/]
    token_budget: 0
`)
	_, err := manifest.Parse(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token_budget")
}

func TestParse_LLMTimeoutValid(t *testing.T) {
	path := writeManifest(t, `
defaults:
  llm_timeout: 30s
docs:
  - path: docs/guide.md
    covers: [internal/]
    llm_timeout: 2m
`)
	m, err := manifest.Parse(path)
	require.NoError(t, err)
	assert.Equal(t, "30s", m.Defaults.LLMTimeout)
	assert.Equal(t, "2m", m.Docs[0].LLMTimeout)
}

func TestParse_LLMTimeoutInvalid(t *testing.T) {
	path := writeManifest(t, `
docs:
  - path: docs/guide.md
    covers: [internal/]
    llm_timeout: not-a-duration
`)
	_, err := manifest.Parse(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "llm_timeout")
	assert.Contains(t, err.Error(), "docs/guide.md")
}

func TestParse_SystemPromptAtFile_Missing(t *testing.T) {
	path := writeManifest(t, `
docs:
  - path: docs/guide.md
    covers: [internal/]
    system_prompt: "@/nonexistent/path.md"
`)
	_, err := manifest.Parse(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "/nonexistent/path.md")
	assert.Contains(t, err.Error(), "docs/guide.md")
}

func TestParse_PerDocPromptMissingFile(t *testing.T) {
	path := writeManifest(t, `
docs:
  - path: docs/a.md
    covers: [internal/]
    system_prompt: "@./does-not-exist.md"
`)
	_, err := manifest.Parse(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "system_prompt")
}

func TestParse_UnknownFieldsIgnored(t *testing.T) {
	path := writeManifest(t, `
docs:
  - path: docs/guide.md
    covers: [internal/]
    models: m1
`)
	m, err := manifest.Parse(path)
	require.NoError(t, err)
	require.Len(t, m.Docs, 1)
	assert.Equal(t, "", m.Docs[0].Model)
}

func TestParse_UnknownKeysIgnored(t *testing.T) {
	path := writeManifest(t, `
docs:
  - path: docs/user-guide.md
    covers:
      - internal/
    unknown_field: ignored
`)
	m, err := manifest.Parse(path)
	require.NoError(t, err)
	assert.Len(t, m.Docs, 1)
}
