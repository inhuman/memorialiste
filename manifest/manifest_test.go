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
	_, err := manifest.Parse(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read")
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
