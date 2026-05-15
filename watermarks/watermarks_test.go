package watermarks_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/inhuman/memorialiste/watermarks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Missing(t *testing.T) {
	f, err := watermarks.Load(filepath.Join(t.TempDir(), "nope.yaml"))
	require.NoError(t, err)
	assert.Empty(t, f.Records)
}

func TestLoad_Valid(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "w.yaml")
	require.NoError(t, os.WriteFile(p, []byte("- path: docs/a.md\n  generated_at: abc\n- path: docs/b.md\n  generated_at: def\n"), 0o644))
	f, err := watermarks.Load(p)
	require.NoError(t, err)
	require.Len(t, f.Records, 2)
	assert.Equal(t, "abc", f.Records[0].GeneratedAt)
}

func TestLoad_Malformed(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "w.yaml")
	require.NoError(t, os.WriteFile(p, []byte("not: [valid: yaml"), 0o644))
	_, err := watermarks.Load(p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse error")
	assert.Contains(t, err.Error(), p)
}

func TestUpsert_Insert(t *testing.T) {
	f := &watermarks.File{}
	f.Upsert("docs/a.md", "sha1")
	require.Len(t, f.Records, 1)
	assert.Equal(t, "sha1", f.Records[0].GeneratedAt)
}

func TestUpsert_Update(t *testing.T) {
	f := &watermarks.File{}
	f.Upsert("docs/a.md", "sha1")
	f.Upsert("docs/b.md", "sha2")
	f.Upsert("docs/a.md", "sha3")
	require.Len(t, f.Records, 2)
	v, ok := f.Lookup("docs/a.md")
	assert.True(t, ok)
	assert.Equal(t, "sha3", v)
	v2, ok := f.Lookup("docs/b.md")
	assert.True(t, ok)
	assert.Equal(t, "sha2", v2)
}

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "w.yaml")
	f := &watermarks.File{}
	f.Upsert("docs/a.md", "sha1")
	f.Upsert("docs/b.md", "sha2")
	require.NoError(t, f.Save(p))
	loaded, err := watermarks.Load(p)
	require.NoError(t, err)
	assert.Equal(t, f.Records, loaded.Records)
}

func TestSave_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "nested", "sub", "w.yaml")
	f := &watermarks.File{}
	f.Upsert("docs/a.md", "sha1")
	require.NoError(t, f.Save(p))
	_, err := os.Stat(p)
	require.NoError(t, err)
}
