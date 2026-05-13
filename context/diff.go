package context

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

// ErrDetachedHEAD is returned when the repository is in detached HEAD state.
var ErrDetachedHEAD = errors.New("context: repository is in detached HEAD state")

// ErrShallowClone is returned when history is insufficient to resolve the watermark commit.
var ErrShallowClone = errors.New("context: shallow clone — run git fetch --unshallow")

// ErrUnknownSHA is returned when the generated_at SHA is not found in the repo.
var ErrUnknownSHA = errors.New("context: generated_at SHA not found in repository")

var binaryExtensions = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".svg": {},
	".pdf": {}, ".zip": {}, ".bin": {}, ".exe": {}, ".so": {}, ".dylib": {},
	".woff": {}, ".woff2": {}, ".ttf": {}, ".eot": {}, ".ico": {},
}

// computeDiff returns the filtered unified diff between watermarkSHA and HEAD,
// scoped to covers paths and with excluded files removed.
func computeDiff(ctx context.Context, repoPath, watermarkSHA string, covers []string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("context: open repo %q: %w", repoPath, err)
	}

	headRef, err := repo.Head()
	if err != nil {
		if strings.Contains(err.Error(), "reference not found") {
			return "", ErrDetachedHEAD
		}
		return "", fmt.Errorf("context: resolve HEAD: %w", err)
	}
	if headRef.Type() != plumbing.HashReference && !headRef.Name().IsBranch() {
		return "", ErrDetachedHEAD
	}

	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", fmt.Errorf("context: resolve HEAD commit: %w", err)
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("context: HEAD tree: %w", err)
	}

	var fromTree *object.Tree
	if watermarkSHA == "" {
		// First generation: diff from empty tree (show all).
		fromTree = new(object.Tree)
	} else {
		fromCommit, err := repo.CommitObject(plumbing.NewHash(watermarkSHA))
		if err != nil {
			if errors.Is(err, plumbing.ErrObjectNotFound) {
				return "", fmt.Errorf("%w: %s", ErrUnknownSHA, watermarkSHA)
			}
			if errors.Is(err, plumbing.ErrReferenceNotFound) {
				return "", fmt.Errorf("%w: %s", ErrShallowClone, watermarkSHA)
			}
			return "", fmt.Errorf("context: resolve watermark commit: %w", err)
		}
		fromTree, err = fromCommit.Tree()
		if err != nil {
			return "", fmt.Errorf("context: watermark tree: %w", err)
		}
	}

	changes, err := object.DiffTreeWithOptions(ctx, fromTree, headTree, &object.DiffTreeOptions{})
	if err != nil {
		return "", fmt.Errorf("context: diff tree: %w", err)
	}

	var sb strings.Builder
	for _, change := range changes {
		path := change.To.Name
		if path == "" {
			path = change.From.Name
		}
		if isExcluded(path) {
			continue
		}
		if !matchesCovers(path, covers) {
			continue
		}
		patch, err := change.Patch()
		if err != nil {
			return "", fmt.Errorf("context: generate patch for %q: %w", path, err)
		}
		sb.WriteString(patch.String())
	}

	return sb.String(), nil
}

// isExcluded reports whether path should be stripped from the diff.
func isExcluded(path string) bool {
	if strings.HasPrefix(path, "vendor/") {
		return true
	}
	if strings.HasSuffix(path, "_test.go") {
		return true
	}
	if strings.HasSuffix(path, ".gen.go") {
		return true
	}
	if strings.HasPrefix(path, "migrations/") {
		return true
	}
	if strings.HasPrefix(path, "docs/") {
		return true
	}
	ext := fileExt(path)
	_, isBinary := binaryExtensions[ext]
	return isBinary
}

// matchesCovers reports whether path falls under any of the covers prefixes.
func matchesCovers(path string, covers []string) bool {
	for _, c := range covers {
		if strings.HasPrefix(path, c) {
			return true
		}
	}
	return false
}

func fileExt(path string) string {
	idx := strings.LastIndexByte(path, '.')
	if idx < 0 {
		return ""
	}
	return path[idx:]
}
