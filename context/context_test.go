package context_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
	mctx "github.com/inhuman/memorialiste/context"
	"github.com/inhuman/memorialiste/internal/fake"
	"github.com/inhuman/memorialiste/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Watermark tests (T012) ────────────────────────────────────────────────────

func TestReadWatermark_WithFrontmatter(t *testing.T) {
	f := writeTempFile(t, "---\ngenerated_at: abc1234\n---\n\n# Doc\n")
	sha, err := mctx.ReadWatermark(f)
	require.NoError(t, err)
	assert.Equal(t, "abc1234", sha)
}

func TestReadWatermark_NoFrontmatter(t *testing.T) {
	f := writeTempFile(t, "# Doc without frontmatter\n")
	sha, err := mctx.ReadWatermark(f)
	require.NoError(t, err)
	assert.Empty(t, sha)
}

func TestReadWatermark_MissingFile(t *testing.T) {
	sha, err := mctx.ReadWatermark(filepath.Join(t.TempDir(), "missing.md"))
	require.NoError(t, err)
	assert.Empty(t, sha)
}

func TestReadWatermark_FrontmatterWithoutKey(t *testing.T) {
	f := writeTempFile(t, "---\nauthor: someone\n---\n\n# Doc\n")
	sha, err := mctx.ReadWatermark(f)
	require.NoError(t, err)
	assert.Empty(t, sha)
}

func TestStripFrontmatter_Present(t *testing.T) {
	content := "---\ngenerated_at: abc\n---\n\n# Body\n"
	body := mctx.StripFrontmatter(content)
	assert.Equal(t, "# Body\n", body)
	assert.NotContains(t, body, "generated_at")
}

func TestStripFrontmatter_Absent(t *testing.T) {
	content := "# Just a doc\n"
	assert.Equal(t, content, mctx.StripFrontmatter(content))
}

func TestStripFrontmatter_SeparatorsInBody(t *testing.T) {
	content := "---\ngenerated_at: abc\n---\n\n# Doc\n\n---\n\nMore content\n"
	body := mctx.StripFrontmatter(content)
	assert.Contains(t, body, "---")
	assert.NotContains(t, body, "generated_at")
}

func TestWriteFrontmatter(t *testing.T) {
	result := mctx.WriteFrontmatter("# Body\n", "deadbeef")
	assert.True(t, strings.HasPrefix(result, "---\ngenerated_at: deadbeef\n---\n"), "result should start with frontmatter")
	assert.Contains(t, result, "# Body\n")
}

// ── Budget tests (T023) ───────────────────────────────────────────────────────

func TestApproxTokens(t *testing.T) {
	assert.Equal(t, 0, mctx.ApproxTokens(""))
	assert.Equal(t, 1, mctx.ApproxTokens("abcd"))
	assert.Equal(t, 25, mctx.ApproxTokens(string(make([]byte, 100))))
}

func TestAssemble_WithinBudget_NoSummariser(t *testing.T) {
	repo, repoPath := initTestRepo(t)
	commitFile(t, repo, repoPath, "internal/foo.go", "package foo\n", "add foo")

	entry := manifest.DocEntry{
		Path:   filepath.Join(repoPath, "docs/guide.md"),
		Covers: []string{"internal/"},
	}
	dc, err := mctx.Assemble(context.Background(), entry, mctx.Options{
		RepoPath:    repoPath,
		TokenBudget: 100000,
	})
	require.NoError(t, err)
	assert.False(t, dc.Summarised)
	assert.NotEmpty(t, dc.HeadSHA)
}

func TestAssemble_OverBudget_SummariserCalled(t *testing.T) {
	repo, repoPath := initTestRepo(t)
	// Create a diff that will exceed a tiny budget.
	content := string(make([]byte, 100))
	commitFile(t, repo, repoPath, "internal/big.go", "package foo\n\n// "+content, "add big")

	called := false
	fp := &fake.Provider{
		SummariseDiffFunc: func(_ context.Context, diff string) (string, error) {
			called = true
			return "summary", nil
		},
	}

	entry := manifest.DocEntry{
		Path:   filepath.Join(repoPath, "docs/guide.md"),
		Covers: []string{"internal/"},
	}
	dc, err := mctx.Assemble(context.Background(), entry, mctx.Options{
		RepoPath:    repoPath,
		TokenBudget: 1, // force summarisation
		Summariser:  fp,
	})
	require.NoError(t, err)
	assert.True(t, called)
	assert.True(t, dc.Summarised)
	assert.Equal(t, "summary", dc.Diff)
}

func TestAssemble_OverBudget_NilSummariser_Error(t *testing.T) {
	repo, repoPath := initTestRepo(t)
	commitFile(t, repo, repoPath, "internal/foo.go", "package foo\n", "add foo")

	entry := manifest.DocEntry{
		Path:   filepath.Join(repoPath, "docs/guide.md"),
		Covers: []string{"internal/"},
	}
	_, err := mctx.Assemble(context.Background(), entry, mctx.Options{
		RepoPath:    repoPath,
		TokenBudget: 1,
		Summariser:  nil,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Summariser provided")
}

// ── Diff tests (T017) ─────────────────────────────────────────────────────────

func TestDiff_ExcludesTestAndVendor(t *testing.T) {
	repo, repoPath := initTestRepo(t)
	commitFile(t, repo, repoPath, "internal/foo.go", "package foo\n", "add source")
	commitFile(t, repo, repoPath, "internal/foo_test.go", "package foo_test\n", "add test")
	commitFile(t, repo, repoPath, "vendor/lib/lib.go", "package lib\n", "add vendor")

	entry := manifest.DocEntry{
		Path:   filepath.Join(repoPath, "docs/guide.md"),
		Covers: []string{"internal/"},
	}
	dc, err := mctx.Assemble(context.Background(), entry, mctx.Options{
		RepoPath:    repoPath,
		TokenBudget: 100000,
	})
	require.NoError(t, err)
	assert.Contains(t, dc.Diff, "internal/foo.go")
	assert.NotContains(t, dc.Diff, "foo_test.go")
	assert.NotContains(t, dc.Diff, "vendor/")
}

func TestDiff_UnknownSHA(t *testing.T) {
	repo, repoPath := initTestRepo(t)
	// Need at least one commit so HEAD is valid.
	commitFile(t, repo, repoPath, "internal/foo.go", "package foo\n", "initial")

	// Write a doc file with a watermark SHA that doesn't exist in history.
	docPath := filepath.Join(repoPath, "docs/guide.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(docPath), 0o755))
	require.NoError(t, os.WriteFile(docPath, []byte("---\ngenerated_at: deadbeef00000000000000000000000000000000\n---\n\n# Doc\n"), 0o644))

	entry := manifest.DocEntry{
		Path:   docPath,
		Covers: []string{"internal/"},
	}
	_, err := mctx.Assemble(context.Background(), entry, mctx.Options{
		RepoPath:    repoPath,
		TokenBudget: 100000,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, mctx.ErrUnknownSHA)
}

func TestDiff_EmptyWatermark_FullHistory(t *testing.T) {
	repo, repoPath := initTestRepo(t)
	commitFile(t, repo, repoPath, "internal/foo.go", "package foo\n", "add source")

	entry := manifest.DocEntry{
		Path:   filepath.Join(repoPath, "docs/guide.md"),
		Covers: []string{"internal/"},
	}
	dc, err := mctx.Assemble(context.Background(), entry, mctx.Options{
		RepoPath:    repoPath,
		TokenBudget: 100000,
	})
	require.NoError(t, err)
	assert.Contains(t, dc.Diff, "internal/foo.go")
}

// ── helpers ───────────────────────────────────────────────────────────────────

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "doc*.md")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func initTestRepo(t *testing.T) (*gogit.Repository, string) {
	t.Helper()
	dir := t.TempDir()
	repo, err := gogit.PlainInit(dir, false)
	require.NoError(t, err)
	return repo, dir
}

func commitFile(t *testing.T, repo *gogit.Repository, repoPath, relPath, content, msg string) {
	t.Helper()
	absPath := filepath.Join(repoPath, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
	require.NoError(t, os.WriteFile(absPath, []byte(content), 0o644))

	wt, err := repo.Worktree()
	require.NoError(t, err)
	_, err = wt.Add(relPath)
	require.NoError(t, err)
	_, err = wt.Commit(msg, &gogit.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@test.com", When: time.Now()},
	})
	require.NoError(t, err)
}
