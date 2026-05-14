package output

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// DefaultBranchPrefix is the default prefix for the auto-generated branch name.
const DefaultBranchPrefix = "docs/memorialiste-"

// Entry is one generated doc ready to be persisted.
type Entry struct {
	// Path is the repository-relative target path (e.g. "docs/architecture.md").
	Path string
	// Body is the clean Markdown body — no frontmatter, no preamble.
	Body string
	// HeadSHA is stamped as the generated_at watermark.
	HeadSHA string
	// Audience comes from the manifest entry's audience field. When set,
	// the audience slug is embedded in the auto-generated branch name so
	// reviewers can tell at a glance which audience the MR/PR targets.
	// Optional — empty audience falls back to the timestamp-only format.
	Audience string
}

// Author is the commit identity used when the system creates a commit.
type Author struct {
	Name  string
	Email string
}

// Options configures Apply.
type Options struct {
	// RepoPath is the path to the local git repository root.
	RepoPath string
	// DryRun, when true, writes files to disk but skips all git operations.
	DryRun bool
	// BranchPrefix overrides DefaultBranchPrefix when non-empty.
	BranchPrefix string
	// Author overrides repo config and the hard-coded fallback when non-empty.
	Author Author
	// Now is an optional clock for tests; defaults to time.Now when nil.
	Now func() time.Time
}

// SkipReason records why an entry was not written.
type SkipReason struct {
	Path   string
	Reason string
}

// Result summarises what Apply did.
type Result struct {
	// WrittenFiles is the list of repo-relative paths actually written.
	WrittenFiles []string
	// SkippedEntries is the list of entries that were not written, with reason.
	SkippedEntries []SkipReason
	// BranchName is empty in dry-run mode or when no files were written.
	BranchName string
	// CommitSHA is empty in dry-run mode or when no files were written.
	CommitSHA string
	// CommitSubject is the first line of the commit message; empty when no
	// commit was created.
	CommitSubject string
	// CommitBody is the body of the commit message (everything after the
	// blank line that follows the subject); empty when no commit was created.
	CommitBody string
}

// ErrBranchExists is returned when the proposed branch name collides
// with an existing local ref.
type ErrBranchExists struct {
	Name string
}

// Error implements error.
func (e *ErrBranchExists) Error() string {
	return fmt.Sprintf("output: branch %q already exists", e.Name)
}

// Apply writes each Entry to disk (creating directories as needed), then —
// when not DryRun and at least one file was written — creates a fresh local
// branch and commits the written files.
//
// Returns an empty Result with no error when entries is empty.
func Apply(_ context.Context, opts Options, entries []Entry) (*Result, error) {
	if opts.Now == nil {
		opts.Now = time.Now
	}

	result := &Result{}

	// --- Phase 1: classify entries (skip empty bodies, build write plan)
	type plannedWrite struct {
		absPath, relPath, body, sha, audience string
	}
	var plan []plannedWrite
	for _, e := range entries {
		if e.Body == "" {
			result.SkippedEntries = append(result.SkippedEntries, SkipReason{
				Path:   e.Path,
				Reason: "empty body",
			})
			continue
		}
		absPath := e.Path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(opts.RepoPath, e.Path)
		}
		relPath, relErr := filepath.Rel(opts.RepoPath, absPath)
		if relErr != nil {
			relPath = e.Path
		}
		plan = append(plan, plannedWrite{
			absPath:  absPath,
			relPath:  filepath.ToSlash(relPath),
			body:     e.Body,
			sha:      e.HeadSHA,
			audience: e.Audience,
		})
	}

	// --- Dry-run: write files now, skip git ops
	if opts.DryRun {
		for _, p := range plan {
			if err := writeFile(p.absPath, p.body, p.sha); err != nil {
				return nil, err
			}
			result.WrittenFiles = append(result.WrittenFiles, p.relPath)
		}
		return result, nil
	}

	// --- No-op exit (FR-011, SC-001): nothing to write → nothing to commit
	if len(plan) == 0 {
		return result, nil
	}

	// --- Phase 2: open repo, resolve HEAD, prepare branch
	repo, err := git.PlainOpen(opts.RepoPath)
	if err != nil {
		return nil, fmt.Errorf("output: open repo %q: %w", opts.RepoPath, err)
	}

	headRef, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("output: resolve HEAD (detached HEAD or no commits?): %w", err)
	}
	headHash := headRef.Hash()
	shortSHA := headHash.String()[:7]

	prefix := opts.BranchPrefix
	if prefix == "" {
		prefix = DefaultBranchPrefix
	}
	audiences := make([]string, 0, len(plan))
	for _, p := range plan {
		audiences = append(audiences, p.audience)
	}
	branch := branchName(prefix, audiences)
	refName := plumbing.NewBranchReferenceName(branch)

	// Collision check
	if _, refErr := repo.Reference(refName, false); refErr == nil {
		return nil, &ErrBranchExists{Name: branch}
	} else if !errors.Is(refErr, plumbing.ErrReferenceNotFound) {
		return nil, fmt.Errorf("output: check branch %q: %w", branch, refErr)
	}

	// Create local branch ref pointing at HEAD
	if err := repo.Storer.SetReference(plumbing.NewHashReference(refName, headHash)); err != nil {
		return nil, fmt.Errorf("output: create branch %q: %w", branch, err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("output: worktree: %w", err)
	}
	// Checkout the new branch BEFORE writing files. Both refs point at the
	// same commit, so this is a no-op for the worktree contents and avoids
	// the "worktree contains unstaged changes" error.
	if err := wt.Checkout(&git.CheckoutOptions{Branch: refName}); err != nil {
		return nil, fmt.Errorf("output: checkout %q: %w", branch, err)
	}

	// Phase 3: write files now that we're on the new branch
	for _, p := range plan {
		if err := writeFile(p.absPath, p.body, p.sha); err != nil {
			return nil, err
		}
		result.WrittenFiles = append(result.WrittenFiles, p.relPath)
	}

	// Phase 4: stage ONLY the files we wrote (unrelated changes stay untouched)
	for _, relPath := range result.WrittenFiles {
		if _, err := wt.Add(relPath); err != nil {
			return nil, fmt.Errorf("output: stage %q: %w", relPath, err)
		}
	}

	author := resolveAuthor(repo, opts.Author)
	subject, body := buildCommitParts(shortSHA, result.WrittenFiles)
	msg := subject + "\n\n" + body

	commitHash, err := wt.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  author.Name,
			Email: author.Email,
			When:  opts.Now(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("output: commit: %w", err)
	}

	result.BranchName = branch
	result.CommitSHA = commitHash.String()
	result.CommitSubject = subject
	result.CommitBody = body
	return result, nil
}

// buildCommitParts produces the subject and body of the commit message.
// The full commit message is subject + "\n\n" + body.
func buildCommitParts(shortSHA string, files []string) (subject, body string) {
	subject = "docs: update documentation to " + shortSHA
	var sb strings.Builder
	sb.WriteString("Updated files:\n")
	for _, f := range files {
		sb.WriteString("- ")
		sb.WriteString(f)
		sb.WriteString("\n")
	}
	return subject, sb.String()
}

