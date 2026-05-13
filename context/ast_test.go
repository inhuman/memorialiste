package context_test

import (
	"context"
	"testing"

	mctx "github.com/inhuman/memorialiste/context"
	"github.com/inhuman/memorialiste/internal/fake"
	"github.com/inhuman/memorialiste/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sampleDiff is a minimal unified diff touching two files.
const sampleDiff = `diff --git a/internal/foo.go b/internal/foo.go
index 0000000..1111111 100644
--- a/internal/foo.go
+++ b/internal/foo.go
@@ -10,6 +10,7 @@ func Foo() {
+	x := 1
 }
diff --git a/internal/bar.go b/internal/bar.go
index 0000000..2222222 100644
--- a/internal/bar.go
+++ b/internal/bar.go
@@ -5,4 +5,5 @@ func Bar() {
+	y := 2
 }
`

// ── T010: US1 enrichment tests ────────────────────────────────────────────────

func TestEnrichDiff_Disabled_DiffUnchanged(t *testing.T) {
	repo, repoPath := initTestRepo(t)
	commitFile(t, repo, repoPath, "internal/foo.go", "package foo\nfunc Foo(){}\n", "add foo")

	entry := manifest.DocEntry{Path: repoPath + "/docs/guide.md", Covers: []string{"internal/"}}
	dc, err := mctx.Assemble(context.Background(), entry, mctx.Options{
		RepoPath:    repoPath,
		TokenBudget: 100000,
		ASTContext:  false,
	})
	require.NoError(t, err)
	assert.False(t, dc.ASTEnriched)
}

func TestEnrichDiff_Enabled_HeadersAdded(t *testing.T) {
	ann := &fake.Annotator{
		AnnotateFunc: func(_ context.Context, filePath string, _ []int) (mctx.ASTAnnotation, error) {
			return mctx.ASTAnnotation{FilePath: filePath, Scopes: []string{"FuncA"}}, nil
		},
	}

	enriched, ok, err := mctx.ExportedEnrichDiff(context.Background(), "/repo", sampleDiff, ann)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Contains(t, enriched, "=== internal/foo.go [FuncA] ===")
	assert.Contains(t, enriched, "=== internal/bar.go [FuncA] ===")
}

func TestEnrichDiff_FallbackAnnotation_PlainHeader(t *testing.T) {
	ann := &fake.Annotator{} // returns empty ASTAnnotation by default

	enriched, ok, err := mctx.ExportedEnrichDiff(context.Background(), "/repo", sampleDiff, ann)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Contains(t, enriched, "=== internal/foo.go ===")
	assert.NotContains(t, enriched, "[")
}

func TestEnrichDiff_FileLevel_PackageLevelAnnotation(t *testing.T) {
	ann := &fake.Annotator{
		AnnotateFunc: func(_ context.Context, filePath string, _ []int) (mctx.ASTAnnotation, error) {
			return mctx.ASTAnnotation{FilePath: filePath, FileLevel: true}, nil
		},
	}

	enriched, ok, err := mctx.ExportedEnrichDiff(context.Background(), "/repo", sampleDiff, ann)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Contains(t, enriched, "(package-level)")
}

func TestEnrichDiff_SameFunctionMultipleHunks_DeduplicatedInHeader(t *testing.T) {
	multiHunkDiff := `diff --git a/internal/foo.go b/internal/foo.go
index 0000000..1111111 100644
--- a/internal/foo.go
+++ b/internal/foo.go
@@ -5,3 +5,4 @@ func Foo() {
+	a := 1
@@ -20,3 +21,4 @@ func Foo() {
+	b := 2
`
	callCount := 0
	ann := &fake.Annotator{
		AnnotateFunc: func(_ context.Context, filePath string, _ []int) (mctx.ASTAnnotation, error) {
			callCount++
			return mctx.ASTAnnotation{FilePath: filePath, Scopes: []string{"Foo"}}, nil
		},
	}

	enriched, _, err := mctx.ExportedEnrichDiff(context.Background(), "/repo", multiHunkDiff, ann)
	require.NoError(t, err)
	// "Foo" should appear only once in the header, not twice.
	assert.Equal(t, 1, countOccurrences(enriched, "[Foo]"))
}

// ── T011: US2 fallback tests ──────────────────────────────────────────────────

func TestEnrichDiff_AnnotatorReturnsEmpty_NoError(t *testing.T) {
	ann := &fake.Annotator{
		AnnotateFunc: func(_ context.Context, filePath string, _ []int) (mctx.ASTAnnotation, error) {
			// Simulate grep-ast unavailable: return empty, no error.
			return mctx.ASTAnnotation{FilePath: filePath}, nil
		},
	}

	_, ok, err := mctx.ExportedEnrichDiff(context.Background(), "/repo", sampleDiff, ann)
	require.NoError(t, err)
	assert.False(t, ok, "should not be enriched when annotator returns empty")
}

// ── T012: US2 regression — ASTContext=false identical to baseline ─────────────

func TestAssemble_ASTContextFalse_IdenticalToBaseline(t *testing.T) {
	repo, repoPath := initTestRepo(t)
	commitFile(t, repo, repoPath, "internal/foo.go", "package foo\nfunc Foo(){}\n", "add foo")

	entry := manifest.DocEntry{Path: repoPath + "/docs/guide.md", Covers: []string{"internal/"}}
	opts := mctx.Options{RepoPath: repoPath, TokenBudget: 100000}

	baseline, err := mctx.Assemble(context.Background(), entry, opts)
	require.NoError(t, err)

	opts.ASTContext = false
	withFlag, err := mctx.Assemble(context.Background(), entry, opts)
	require.NoError(t, err)

	assert.Equal(t, baseline.Diff, withFlag.Diff)
	assert.Equal(t, baseline.ASTEnriched, withFlag.ASTEnriched)
	assert.False(t, withFlag.ASTEnriched)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}
