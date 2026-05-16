package codesearch

import (
	"cmp"
	"context"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Default tuning knobs exposed for callers.
const (
	// DefaultLimit caps the number of returned hits when SearchRequest.Limit ≤ 0.
	DefaultLimit = 20
	// DefaultLineCap caps the per-hit source body length in lines.
	DefaultLineCap = 200
	// DefaultParseTimeout caps the duration of a single parser.ParseFile call.
	DefaultParseTimeout = 5 * time.Second
)

// SearchRequest is one tool invocation.
type SearchRequest struct {
	// RepoRoot is the anchor for path validation. May be absolute or relative
	// to the current working directory.
	RepoRoot string `json:"repo_root,omitempty"`
	// Path is a repo-relative scope. Empty means whole repo.
	Path string `json:"path,omitempty"`
	// Pattern is a Go-flavoured regex matched against identifier names.
	Pattern string `json:"pattern"`
	// Limit caps the number of returned hits. 0 → DefaultLimit.
	Limit int `json:"limit,omitempty"`
	// ParseTimeout caps a single parser.ParseFile call. 0 → DefaultParseTimeout.
	// Set by the dispatcher from cliconfig.Config.ASTParseTimeout; not exposed
	// to the LLM tool schema.
	ParseTimeout time.Duration `json:"-"`
}

// SearchHit is one matched declaration.
type SearchHit struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	FilePath  string `json:"file"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Source    string `json:"source"`
	Truncated bool   `json:"truncated,omitempty"`
}

// SearchResult is what Search returns.
type SearchResult struct {
	Hits         []SearchHit `json:"hits"`
	Truncated    bool        `json:"truncated,omitempty"`
	FilesScanned int         `json:"files_scanned"`
	FilesSkipped int         `json:"files_skipped"`
}

// Search walks RepoRoot/Path, parses .go files via go/ast, and returns
// declarations whose identifier name matches the regex Pattern.
//
// Returns an error on invalid pattern, path traversal outside RepoRoot, or
// when RepoRoot cannot be resolved. Per-file parse errors and timeouts are
// silently skipped and counted in SearchResult.FilesSkipped.
func Search(ctx context.Context, req SearchRequest) (*SearchResult, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	parseTimeout := cmp.Or(req.ParseTimeout, DefaultParseTimeout)
	repoRoot := cmp.Or(req.RepoRoot, ".")

	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("codesearch: resolve repo root: %w", err)
	}
	absScope, err := filepath.Abs(filepath.Join(absRoot, req.Path))
	if err != nil {
		return nil, fmt.Errorf("codesearch: resolve scope: %w", err)
	}
	rel, err := filepath.Rel(absRoot, absScope)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("codesearch: scope %q escapes repo root", req.Path)
	}

	pattern, err := regexp.Compile(req.Pattern)
	if err != nil {
		return nil, fmt.Errorf("codesearch: invalid pattern: %w", err)
	}

	paths, err := walkGoFiles(absScope)
	if err != nil {
		return nil, fmt.Errorf("codesearch: walk: %w", err)
	}

	fset := token.NewFileSet()
	res := &SearchResult{}
	var allHits []SearchHit
	for _, path := range paths {
		src, err := os.ReadFile(path)
		if err != nil {
			res.FilesSkipped++
			continue
		}
		file, err := parseGoFile(ctx, fset, path, src, parseTimeout)
		if err != nil {
			res.FilesSkipped++
			continue
		}
		res.FilesScanned++

		relPath, relErr := filepath.Rel(absRoot, path)
		if relErr != nil {
			relPath = path
		}
		relPath = filepath.ToSlash(relPath)

		hits := extractHits(fset, file, src, relPath, pattern, token.NoPos)
		allHits = append(allHits, hits...)
	}

	slices.SortFunc(allHits, func(a, b SearchHit) int {
		if c := cmp.Compare(a.FilePath, b.FilePath); c != 0 {
			return c
		}
		return cmp.Compare(a.StartLine, b.StartLine)
	})

	if len(allHits) > limit {
		res.Hits = allHits[:limit]
		res.Truncated = true
	} else {
		res.Hits = allHits
	}
	return res, nil
}
