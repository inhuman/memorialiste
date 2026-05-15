package output_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/inhuman/memorialiste/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func initRepoWithCommit(t *testing.T) (*gogit.Repository, string) {
	t.Helper()
	dir := t.TempDir()
	repo, err := gogit.PlainInit(dir, false)
	require.NoError(t, err)
	wt, err := repo.Worktree()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# repo\n"), 0o644))
	_, err = wt.Add("README.md")
	require.NoError(t, err)
	_, err = wt.Commit("init", &gogit.CommitOptions{
		Author: &object.Signature{Name: "x", Email: "x@x", When: time.Now()},
	})
	require.NoError(t, err)
	return repo, dir
}

func fixedClock(ts string) func() time.Time {
	t, _ := time.Parse("20060102-150405", ts)
	return func() time.Time { return t }
}

// ── US1: write path ───────────────────────────────────────────────────────────

func TestApply_WritesFileWithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	res, err := output.Apply(context.Background(), output.Options{
		RepoPath: dir,
		DryRun:   true, // we only care about the write here
	}, []output.Entry{
		{Path: "docs/foo.md", Body: "# Body", HeadSHA: "abc1234"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"docs/foo.md"}, res.WrittenFiles)

	content, err := os.ReadFile(filepath.Join(dir, "docs/foo.md"))
	require.NoError(t, err)
	assert.Equal(t, "---\ngenerated_at: abc1234\n---\n\n# Body", string(content))
}

func TestApply_CreatesMissingParentDir(t *testing.T) {
	dir := t.TempDir()
	_, err := output.Apply(context.Background(), output.Options{
		RepoPath: dir,
		DryRun:   true,
	}, []output.Entry{
		{Path: "deeply/nested/dir/foo.md", Body: "x", HeadSHA: "sha"},
	})
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "deeply/nested/dir/foo.md"))
	require.NoError(t, err)
}

func TestApply_EmptyBody_Skipped(t *testing.T) {
	dir := t.TempDir()
	res, err := output.Apply(context.Background(), output.Options{
		RepoPath: dir,
		DryRun:   true,
	}, []output.Entry{
		{Path: "docs/empty.md", Body: "", HeadSHA: "sha"},
		{Path: "docs/ok.md", Body: "ok", HeadSHA: "sha"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"docs/ok.md"}, res.WrittenFiles)
	require.Len(t, res.SkippedEntries, 1)
	assert.Equal(t, "docs/empty.md", res.SkippedEntries[0].Path)
	assert.Equal(t, "empty body", res.SkippedEntries[0].Reason)
	_, err = os.Stat(filepath.Join(dir, "docs/empty.md"))
	assert.True(t, os.IsNotExist(err))
}

func TestApply_EmptyEntries_NoError(t *testing.T) {
	dir := t.TempDir()
	res, err := output.Apply(context.Background(), output.Options{
		RepoPath: dir,
		DryRun:   false,
	}, nil)
	require.NoError(t, err)
	assert.Empty(t, res.WrittenFiles)
	assert.Empty(t, res.BranchName)
	assert.Empty(t, res.CommitSHA)
}

func TestApply_DryRunWritesButDoesntCommit(t *testing.T) {
	repo, dir := initRepoWithCommit(t)
	headRef, _ := repo.Head()
	origHead := headRef.Hash()

	res, err := output.Apply(context.Background(), output.Options{
		RepoPath: dir,
		DryRun:   true,
	}, []output.Entry{
		{Path: "docs/x.md", Body: "x", HeadSHA: origHead.String()},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"docs/x.md"}, res.WrittenFiles)
	assert.Empty(t, res.BranchName)
	assert.Empty(t, res.CommitSHA)

	// HEAD should still be the original
	headAfter, _ := repo.Head()
	assert.Equal(t, origHead, headAfter.Hash())
	// File on disk exists
	_, err = os.Stat(filepath.Join(dir, "docs/x.md"))
	require.NoError(t, err)
}

// ── US2: commit path ──────────────────────────────────────────────────────────

func TestApply_NonDryRun_CreatesBranchAndCommit(t *testing.T) {
	repo, dir := initRepoWithCommit(t)
	headRef, _ := repo.Head()
	origHead := headRef.Hash()

	res, err := output.Apply(context.Background(), output.Options{
		RepoPath: dir,
		DryRun:   false,
		Now:      fixedClock("20260514-093412"),
	}, []output.Entry{
		{Path: "docs/a.md", Body: "A", HeadSHA: origHead.String(), Audience: "developers"},
		{Path: "docs/b.md", Body: "B", HeadSHA: origHead.String(), Audience: "developers"},
	})
	require.NoError(t, err)

	// Branch name format: <prefix><audience-slug>
	expected := output.DefaultBranchPrefix + "developers"
	assert.Equal(t, expected, res.BranchName)
	assert.NotEmpty(t, res.CommitSHA)
	assert.Equal(t, []string{"docs/a.md", "docs/b.md"}, res.WrittenFiles)

	// Worktree is on new branch, new commit
	headAfter, _ := repo.Head()
	assert.NotEqual(t, origHead, headAfter.Hash())
	assert.Equal(t, res.CommitSHA, headAfter.Hash().String())

	// Verify commit contains both files
	commit, err := repo.CommitObject(headAfter.Hash())
	require.NoError(t, err)
	tree, _ := commit.Tree()
	_, errA := tree.File("docs/a.md")
	_, errB := tree.File("docs/b.md")
	require.NoError(t, errA)
	require.NoError(t, errB)
}

func TestApply_CustomBranchPrefix(t *testing.T) {
	repo, dir := initRepoWithCommit(t)
	headRef, _ := repo.Head()

	res, err := output.Apply(context.Background(), output.Options{
		RepoPath:     dir,
		BranchPrefix: "chore/auto-docs-",
		Now:          fixedClock("20260514-100000"),
	}, []output.Entry{
		{Path: "docs/a.md", Body: "A", HeadSHA: headRef.Hash().String()},
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(res.BranchName, "chore/auto-docs-"),
		"branch=%q", res.BranchName)
}

func TestApply_BranchCollision_ErrBranchExists(t *testing.T) {
	repo, dir := initRepoWithCommit(t)
	headRef, _ := repo.Head()
	// Default audience "common" → branch <prefix>common
	expectedBranch := output.DefaultBranchPrefix + "common"

	require.NoError(t, repo.Storer.SetReference(
		plumbing.NewHashReference(plumbing.NewBranchReferenceName(expectedBranch), headRef.Hash()),
	))

	_, err := output.Apply(context.Background(), output.Options{
		RepoPath: dir,
	}, []output.Entry{
		{Path: "docs/a.md", Body: "A", HeadSHA: headRef.Hash().String()},
	})
	require.Error(t, err)
	var brErr *output.ErrBranchExists
	require.True(t, errors.As(err, &brErr))
	assert.Equal(t, expectedBranch, brErr.Name)
}

// ── Branch-name audience semantics ────────────────────────────────────────────

func TestApply_BranchName_SingleAudience(t *testing.T) {
	repo, dir := initRepoWithCommit(t)
	headRef, _ := repo.Head()

	res, err := output.Apply(context.Background(), output.Options{RepoPath: dir}, []output.Entry{
		{Path: "docs/a.md", Body: "A", HeadSHA: headRef.Hash().String(), Audience: "end users"},
		{Path: "docs/b.md", Body: "B", HeadSHA: headRef.Hash().String(), Audience: "end users"},
	})
	require.NoError(t, err)
	assert.Equal(t, output.DefaultBranchPrefix+"end-users", res.BranchName)
}

func TestApply_BranchName_MultiAudience(t *testing.T) {
	repo, dir := initRepoWithCommit(t)
	headRef, _ := repo.Head()

	res, err := output.Apply(context.Background(), output.Options{RepoPath: dir}, []output.Entry{
		{Path: "docs/a.md", Body: "A", HeadSHA: headRef.Hash().String(), Audience: "developers"},
		{Path: "docs/b.md", Body: "B", HeadSHA: headRef.Hash().String(), Audience: "end users"},
	})
	require.NoError(t, err)
	assert.Equal(t, output.DefaultBranchPrefix+"multi", res.BranchName)
}

func TestApply_BranchName_EmptyAudienceDefaultsToCommon(t *testing.T) {
	repo, dir := initRepoWithCommit(t)
	headRef, _ := repo.Head()

	res, err := output.Apply(context.Background(), output.Options{RepoPath: dir}, []output.Entry{
		{Path: "docs/a.md", Body: "A", HeadSHA: headRef.Hash().String()},
	})
	require.NoError(t, err)
	assert.Equal(t, output.DefaultBranchPrefix+"common", res.BranchName)
}

func TestApply_BranchName_AudienceSlugifiedAndDeduped(t *testing.T) {
	repo, dir := initRepoWithCommit(t)
	headRef, _ := repo.Head()

	res, err := output.Apply(context.Background(), output.Options{RepoPath: dir}, []output.Entry{
		{Path: "docs/a.md", Body: "A", HeadSHA: headRef.Hash().String(), Audience: "AI Assistants!"},
		{Path: "docs/b.md", Body: "B", HeadSHA: headRef.Hash().String(), Audience: "  AI   assistants  "},
	})
	require.NoError(t, err)
	assert.Equal(t, output.DefaultBranchPrefix+"ai-assistants", res.BranchName)
}

func TestApply_CommitMessageStructure(t *testing.T) {
	repo, dir := initRepoWithCommit(t)
	headRef, _ := repo.Head()
	shortSHA := headRef.Hash().String()[:7]

	_, err := output.Apply(context.Background(), output.Options{
		RepoPath: dir,
		Now:      fixedClock("20260514-120000"),
	}, []output.Entry{
		{Path: "docs/a.md", Body: "A", HeadSHA: headRef.Hash().String()},
		{Path: "docs/b.md", Body: "B", HeadSHA: headRef.Hash().String()},
	})
	require.NoError(t, err)

	headAfter, _ := repo.Head()
	commit, err := repo.CommitObject(headAfter.Hash())
	require.NoError(t, err)

	subject := "docs: update documentation to " + shortSHA
	assert.Contains(t, commit.Message, subject)
	assert.Contains(t, commit.Message, "Updated files:")
	assert.Contains(t, commit.Message, "- docs/a.md")
	assert.Contains(t, commit.Message, "- docs/b.md")
}

func TestApply_UnrelatedChangesPreserved(t *testing.T) {
	repo, dir := initRepoWithCommit(t)
	headRef, _ := repo.Head()

	// User makes an unrelated change before running memorialiste
	unrelatedPath := filepath.Join(dir, "src.go")
	require.NoError(t, os.WriteFile(unrelatedPath, []byte("package main\n"), 0o644))

	_, err := output.Apply(context.Background(), output.Options{
		RepoPath: dir,
		Now:      fixedClock("20260514-130000"),
	}, []output.Entry{
		{Path: "docs/a.md", Body: "A", HeadSHA: headRef.Hash().String()},
	})
	require.NoError(t, err)

	// Unrelated file still exists in working tree
	_, err = os.Stat(unrelatedPath)
	require.NoError(t, err)

	// Unrelated file is NOT in the new commit
	headAfter, _ := repo.Head()
	commit, _ := repo.CommitObject(headAfter.Hash())
	tree, _ := commit.Tree()
	_, errSrc := tree.File("src.go")
	assert.Error(t, errSrc, "src.go must NOT be in the commit")

	// Working tree still reports src.go as untracked/changed
	wt, _ := repo.Worktree()
	status, _ := wt.Status()
	assert.False(t, status.IsClean(), "unrelated change should remain in working tree")
}

// ── US5: sidecar watermark tests ──────────────────────────────────────────────

func TestApply_Sidecar_NoFrontmatterInDoc(t *testing.T) {
	dir := t.TempDir()
	_, err := output.Apply(context.Background(), output.Options{
		RepoPath: dir,
		DryRun:   true,
	}, []output.Entry{
		{Path: "docs/a.md", Body: "# Title", HeadSHA: "sha1", WatermarksFile: ".watermarks.yaml"},
	})
	require.NoError(t, err)
	got, err := os.ReadFile(filepath.Join(dir, "docs/a.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Title", string(got))
	assert.NotContains(t, string(got), "---")
}

func TestApply_Sidecar_RecordUpserted(t *testing.T) {
	dir := t.TempDir()
	_, err := output.Apply(context.Background(), output.Options{
		RepoPath: dir,
		DryRun:   true,
	}, []output.Entry{
		{Path: "docs/a.md", Body: "# A", HeadSHA: "sha-A", WatermarksFile: ".watermarks.yaml"},
	})
	require.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(dir, ".watermarks.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "docs/a.md")
	assert.Contains(t, string(data), "sha-A")
}

func TestApply_Sidecar_MultipleEntriesSameFile(t *testing.T) {
	dir := t.TempDir()
	_, err := output.Apply(context.Background(), output.Options{RepoPath: dir, DryRun: true}, []output.Entry{
		{Path: "docs/a.md", Body: "A", HeadSHA: "shaA", WatermarksFile: ".w.yaml"},
		{Path: "docs/b.md", Body: "B", HeadSHA: "shaB", WatermarksFile: ".w.yaml"},
	})
	require.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(dir, ".w.yaml"))
	require.NoError(t, err)
	s := string(data)
	assert.Contains(t, s, "docs/a.md")
	assert.Contains(t, s, "docs/b.md")
}

func TestApply_Sidecar_DifferentFilesPerDoc(t *testing.T) {
	dir := t.TempDir()
	_, err := output.Apply(context.Background(), output.Options{RepoPath: dir, DryRun: true}, []output.Entry{
		{Path: "docs/a.md", Body: "A", HeadSHA: "shaA", WatermarksFile: ".wA.yaml"},
		{Path: "docs/b.md", Body: "B", HeadSHA: "shaB", WatermarksFile: ".wB.yaml"},
	})
	require.NoError(t, err)
	a, err := os.ReadFile(filepath.Join(dir, ".wA.yaml"))
	require.NoError(t, err)
	b, err := os.ReadFile(filepath.Join(dir, ".wB.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(a), "docs/a.md")
	assert.NotContains(t, string(a), "docs/b.md")
	assert.Contains(t, string(b), "docs/b.md")
}

func TestApply_Sidecar_StagedInCommit(t *testing.T) {
	repo, dir := initRepoWithCommit(t)
	headRef, _ := repo.Head()
	res, err := output.Apply(context.Background(), output.Options{
		RepoPath: dir,
		Now:      fixedClock("20260514-130000"),
	}, []output.Entry{
		{Path: "docs/a.md", Body: "A", HeadSHA: headRef.Hash().String(), WatermarksFile: ".w.yaml"},
	})
	require.NoError(t, err)
	commit, err := repo.CommitObject(plumbing.NewHash(res.CommitSHA))
	require.NoError(t, err)
	tree, err := commit.Tree()
	require.NoError(t, err)
	_, err = tree.File(".w.yaml")
	require.NoError(t, err, "sidecar must be staged in commit")
	_, err = tree.File("docs/a.md")
	require.NoError(t, err)
}

func TestApply_FrontmatterMode_Unchanged(t *testing.T) {
	dir := t.TempDir()
	_, err := output.Apply(context.Background(), output.Options{RepoPath: dir, DryRun: true}, []output.Entry{
		{Path: "docs/a.md", Body: "# Body", HeadSHA: "abc"},
	})
	require.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(dir, "docs/a.md"))
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(data), "---\ngenerated_at: abc\n---"))
}
