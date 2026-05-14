package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initRepoWithTags(t *testing.T, tagNames []string) (*gogit.Repository, string) {
	t.Helper()
	dir := t.TempDir()
	repo, err := gogit.PlainInit(dir, false)
	require.NoError(t, err)
	wt, err := repo.Worktree()
	require.NoError(t, err)

	base := time.Now().Add(-time.Hour).UTC()
	for i, name := range tagNames {
		fname := fmt.Sprintf("f%d", i)
		require.NoError(t, os.WriteFile(filepath.Join(dir, fname), []byte(name), 0o644))
		_, err := wt.Add(fname)
		require.NoError(t, err)
		commitTime := base.Add(time.Duration(i) * time.Minute)
		h, err := wt.Commit(name, &gogit.CommitOptions{
			Author: &object.Signature{Name: "t", Email: "t@t", When: commitTime},
		})
		require.NoError(t, err)
		tagRef := plumbing.NewHashReference(plumbing.NewTagReferenceName(name), h)
		require.NoError(t, repo.Storer.SetReference(tagRef))
	}
	return repo, dir
}

func initEmptyRepoWithCommit(t *testing.T) (*gogit.Repository, string) {
	t.Helper()
	dir := t.TempDir()
	repo, err := gogit.PlainInit(dir, false)
	require.NoError(t, err)
	wt, err := repo.Worktree()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README"), []byte("x"), 0o644))
	_, err = wt.Add("README")
	require.NoError(t, err)
	_, err = wt.Commit("init", &gogit.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@t", When: time.Now().UTC()},
	})
	require.NoError(t, err)
	return repo, dir
}

// ── BASIC layer tests (T010) ───────────────────────────────────────────────────

func TestGatherRepoMeta_LatestTag(t *testing.T) {
	repo, _ := initRepoWithTags(t, []string{"v0.1.0", "v0.2.0", "v0.1.5"})
	m, err := gatherRepoMeta(repo, MetaBasic)
	require.NoError(t, err)
	assert.Equal(t, "v0.1.5", m.LatestTag)
	assert.Len(t, m.HeadSHA, 40)
}

func TestGatherRepoMeta_NoTags(t *testing.T) {
	repo, _ := initEmptyRepoWithCommit(t)
	m, err := gatherRepoMeta(repo, MetaBasic)
	require.NoError(t, err)
	assert.Empty(t, m.LatestTag)
	assert.Len(t, m.HeadSHA, 40)
}

func TestGatherRepoMeta_ShortSHA(t *testing.T) {
	repo, _ := initRepoWithTags(t, []string{"v0.1.0"})
	m, err := gatherRepoMeta(repo, MetaBasic)
	require.NoError(t, err)
	assert.Equal(t, m.HeadSHA[:7], m.ShortSHA)
}

func TestFormatBasic_AllFields(t *testing.T) {
	m := &RepoMeta{LatestTag: "v0.2.0", HeadSHA: "0cbf1757b71232ece3627557d67f9b4209c88c7c", ShortSHA: "0cbf175"}
	out := m.Format(MetaBasic)
	assert.Contains(t, out, "=== Repository metadata ===")
	assert.Contains(t, out, "Latest tag: v0.2.0")
	assert.Contains(t, out, "HEAD: 0cbf1757b71232ece3627557d67f9b4209c88c7c")
	assert.Contains(t, out, "Short SHA: 0cbf175")
	assert.Contains(t, out, "=== End metadata ===")
}

func TestFormatBasic_NoTag(t *testing.T) {
	m := &RepoMeta{HeadSHA: "abc", ShortSHA: "abc"}
	out := m.Format(MetaBasic)
	assert.Contains(t, out, "Latest tag: (none)")
}

func TestFormatBasic_Deterministic(t *testing.T) {
	m := &RepoMeta{LatestTag: "v1", HeadSHA: "deadbeef0000000000000000000000000000abcd", ShortSHA: "deadbee"}
	a := m.Format(MetaBasic)
	b := m.Format(MetaBasic)
	assert.Equal(t, a, b)
}

func TestFormatNil(t *testing.T) {
	var m *RepoMeta
	assert.Equal(t, "", m.Format(MetaBasic))
}

// ── EXTENDED layer tests (T011) ────────────────────────────────────────────────

func TestRedactURL_HTTPSWithCreds(t *testing.T) {
	out := redactURL("https://oauth2:ghp_secretABC@github.com/o/r.git")
	assert.NotContains(t, out, "ghp_secretABC")
	// "<redacted>" is percent-encoded by net/url; check the encoded marker.
	assert.Contains(t, out, "redacted")
}

func TestRedactURL_HTTPSNoCreds(t *testing.T) {
	in := "https://github.com/o/r.git"
	assert.Equal(t, in, redactURL(in))
}

func TestRedactURL_SSH(t *testing.T) {
	in := "git@github.com:o/r.git"
	assert.Equal(t, in, redactURL(in))
}

func TestRedactURL_Malformed(t *testing.T) {
	in := "::not::a::url::"
	assert.Equal(t, in, redactURL(in))
}

func TestGatherRepoMeta_Extended_FullRepo(t *testing.T) {
	repo, _ := initRepoWithTags(t, []string{"v0.1.0", "v0.1.1", "v0.1.2", "v0.1.3", "v0.1.4", "v0.1.5", "v0.1.6"})
	_, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://github.com/o/r.git"},
	})
	require.NoError(t, err)

	m, err := gatherRepoMeta(repo, MetaExtended)
	require.NoError(t, err)
	assert.Len(t, m.RecentTags, 5)
	// newest first
	assert.Equal(t, "v0.1.6", m.RecentTags[0].Name)
	assert.True(t, strings.HasSuffix(m.RemoteURL, ".git"))
	// go-git default branch is "master"
	assert.NotEmpty(t, m.Branch)
}

func TestGatherRepoMeta_Extended_NoRemote(t *testing.T) {
	repo, _ := initRepoWithTags(t, []string{"v1"})
	m, err := gatherRepoMeta(repo, MetaExtended)
	require.NoError(t, err)
	assert.Empty(t, m.RemoteURL)
	assert.NotEmpty(t, m.HeadSHA)
}

func TestGatherRepoMeta_Extended_FewerThan5Tags(t *testing.T) {
	repo, _ := initRepoWithTags(t, []string{"v0.1.0", "v0.2.0"})
	m, err := gatherRepoMeta(repo, MetaExtended)
	require.NoError(t, err)
	assert.Len(t, m.RecentTags, 2)
}

func TestFormatExtended_AllFields(t *testing.T) {
	now := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	m := &RepoMeta{
		LatestTag: "v0.2.0",
		HeadSHA:   "deadbeef0000000000000000000000000000abcd",
		ShortSHA:  "deadbee",
		RemoteURL: "https://github.com/o/r.git",
		Branch:    "main",
		RecentTags: []TagInfo{
			{Name: "v0.2.0", Date: now},
			{Name: "v0.1.9", Date: now.Add(-24 * time.Hour)},
			{Name: "v0.1.8", Date: now.Add(-48 * time.Hour)},
			{Name: "v0.1.7", Date: now.Add(-72 * time.Hour)},
			{Name: "v0.1.6", Date: now.Add(-96 * time.Hour)},
		},
	}
	out := m.Format(MetaExtended)
	assert.Contains(t, out, "Remote: https://github.com/o/r.git")
	assert.Contains(t, out, "Branch: main")
	assert.Contains(t, out, "Recent tags:")
	assert.Contains(t, out, "- v0.2.0 (2026-05-14)")
	assert.Contains(t, out, "- v0.1.6 (2026-05-10)")
}

func TestFormatExtended_NoRecentTags(t *testing.T) {
	m := &RepoMeta{HeadSHA: "x", ShortSHA: "x", Branch: "main"}
	out := m.Format(MetaExtended)
	assert.NotContains(t, out, "Recent tags:")
}

func TestFormatExtended_TokenLeakage(t *testing.T) {
	m := &RepoMeta{
		HeadSHA:   "x",
		ShortSHA:  "x",
		Branch:    "main",
		RemoteURL: "https://oauth2:fake-secret@host/r.git",
	}
	out := m.Format(MetaExtended)
	assert.NotContains(t, out, "fake-secret")
}
