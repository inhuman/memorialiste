package platform

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPSPush_SSHRemoteRejected(t *testing.T) {
	err := HTTPSPush(t.Context(), t.TempDir(), "git@github.com:o/r.git", "main", "tok", 0)
	require.ErrorIs(t, err, ErrSSHRemoteNotSupported)
}

func TestHTTPSPush_HTTPSchemeRejected(t *testing.T) {
	err := HTTPSPush(t.Context(), t.TempDir(), "http://example.com/o/r.git", "main", "tok", 0)
	require.ErrorIs(t, err, ErrSSHRemoteNotSupported)
}

func TestDoPush_PushesBranchToBareRepo(t *testing.T) {
	srcDir := t.TempDir()
	repo, err := gogit.PlainInit(srcDir, false)
	require.NoError(t, err)
	wt, err := repo.Worktree()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "README.md"), []byte("# hi\n"), 0o644))
	_, err = wt.Add("README.md")
	require.NoError(t, err)
	_, err = wt.Commit("init", &gogit.CommitOptions{
		Author: &object.Signature{Name: "x", Email: "x@x", When: time.Now()},
	})
	require.NoError(t, err)

	// Create a feature branch at HEAD
	headRef, err := repo.Head()
	require.NoError(t, err)
	branchRef := plumbing.NewBranchReferenceName("feature")
	require.NoError(t, repo.Storer.SetReference(plumbing.NewHashReference(branchRef, headRef.Hash())))

	bareDir := t.TempDir()
	_, err = gogit.PlainInit(bareDir, true)
	require.NoError(t, err)

	err = doPush(t.Context(), srcDir, "file://"+bareDir, "feature", "", 30*time.Second)
	require.NoError(t, err)

	bare, err := gogit.PlainOpen(bareDir)
	require.NoError(t, err)
	ref, err := bare.Reference(branchRef, false)
	require.NoError(t, err)
	assert.Equal(t, headRef.Hash(), ref.Hash())
}

func TestRedactToken(t *testing.T) {
	err := redactToken(assertErr("failed with token abc123"), "abc123")
	require.NotNil(t, err)
	assert.NotContains(t, err.Error(), "abc123")
	assert.Contains(t, err.Error(), "[REDACTED]")

	// nil passthrough
	assert.Nil(t, redactToken(nil, "abc"))
	// empty token passthrough
	in := assertErr("boom")
	assert.Equal(t, in, redactToken(in, ""))
}

type stringErr string

func (e stringErr) Error() string { return string(e) }

func assertErr(s string) error { return stringErr(s) }
