// Package context assembles the diff context for a single documentation entry.
//
// It reads the generated_at watermark from the doc file's YAML frontmatter,
// computes a filtered git diff scoped to the entry's covered paths, enforces
// a token budget, and optionally summarises large diffs via an injected
// Summariser.
package context

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/go-git/go-git/v6"
	"github.com/inhuman/memorialiste/manifest"
)

const defaultTokenBudget = 12000

// Summariser compresses a diff that exceeds the token budget.
// Implemented by LLM provider adapters; a fake is available in internal/fake.
type Summariser interface {
	SummariseDiff(ctx context.Context, diff string) (string, error)
}

// Options configures the context assembly for one run.
type Options struct {
	// RepoPath is the path to the local git repository root.
	RepoPath string
	// TokenBudget is the maximum approximate token count for the raw diff.
	// When zero, defaultTokenBudget (12000) is used.
	// Approximation: len(diff)/4.
	TokenBudget int
	// Summariser is called when the diff exceeds TokenBudget.
	// May be nil when TokenBudget is never expected to be exceeded.
	Summariser Summariser
	// ASTContext enables AST-enriched diff context via grep-ast.
	// Default false — opt-in only.
	ASTContext bool
	// Annotator is the ASTAnnotator implementation used when ASTContext is
	// true. When nil, grepASTAnnotator is used automatically.
	Annotator ASTAnnotator
	// RepoMetaLevel controls how much repository metadata is collected and
	// emitted in the LLM user message. Defaults to MetaBasic when empty.
	RepoMetaLevel MetaLevel
}

// DiffContext holds the assembled context for one doc entry.
type DiffContext struct {
	// Entry is the source doc entry from the manifest.
	Entry manifest.DocEntry
	// DocBody is the doc file content with frontmatter stripped.
	DocBody string
	// Diff is the filtered git diff (raw, summarised, or AST-enriched).
	Diff string
	// HeadSHA is the current HEAD commit SHA for the watermark bump.
	HeadSHA string
	// Summarised is true when Diff was compressed via Summariser.
	Summarised bool
	// ASTEnriched is true when at least one file's diff was annotated with
	// AST scope information.
	ASTEnriched bool
	// RepoMeta holds the collected repository metadata; nil when collection
	// failed gracefully (which itself does not fail the run).
	RepoMeta *RepoMeta
}

// Assemble builds a DiffContext for the given DocEntry.
//
// It opens the git repository at opts.RepoPath, reads the watermark from
// entry.Path, computes the filtered diff, and enforces the token budget.
func Assemble(ctx context.Context, entry manifest.DocEntry, opts Options) (*DiffContext, error) {
	if opts.TokenBudget == 0 {
		opts.TokenBudget = defaultTokenBudget
	}
	opts.RepoMetaLevel = cmp.Or(opts.RepoMetaLevel, MetaBasic)

	watermark, err := ReadWatermark(entry.Path)
	if err != nil {
		return nil, err
	}

	docBody, err := readDocBody(entry.Path)
	if err != nil {
		return nil, err
	}

	diff, err := computeDiff(ctx, opts.RepoPath, watermark, entry.Covers)
	if err != nil {
		return nil, err
	}

	headSHA, err := resolveHEAD(opts.RepoPath)
	if err != nil {
		return nil, err
	}

	var repoMeta *RepoMeta
	if repo, openErr := git.PlainOpen(opts.RepoPath); openErr == nil {
		meta, metaErr := gatherRepoMeta(repo, opts.RepoMetaLevel)
		if metaErr != nil {
			log.Printf("context: gather repo meta: %v", metaErr)
		} else {
			repoMeta = meta
		}
	} else {
		log.Printf("context: gather repo meta: %v", openErr)
	}

	// AST enrichment (opt-in).
	astEnriched := false
	if opts.ASTContext && diff != "" {
		annotator := opts.Annotator
		if annotator == nil {
			annotator = &grepASTAnnotator{repoPath: opts.RepoPath}
		}
		enriched, ok, enrichErr := enrichDiff(ctx, opts.RepoPath, diff, annotator)
		if enrichErr != nil {
			return nil, fmt.Errorf("context: AST enrichment: %w", enrichErr)
		}
		diff = enriched
		astEnriched = ok
	}

	summarised := false
	if ApproxTokens(diff) > opts.TokenBudget {
		if opts.Summariser == nil {
			return nil, fmt.Errorf("context: diff exceeds token budget (%d tokens) but no Summariser provided",
				ApproxTokens(diff))
		}
		diff, err = opts.Summariser.SummariseDiff(ctx, diff)
		if err != nil {
			return nil, fmt.Errorf("context: summarise diff: %w", err)
		}
		summarised = true
	}

	return &DiffContext{
		Entry:       entry,
		DocBody:     docBody,
		Diff:        diff,
		HeadSHA:     headSHA,
		Summarised:  summarised,
		ASTEnriched: astEnriched,
		RepoMeta:    repoMeta,
	}, nil
}

func readDocBody(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("context: read doc body %q: %w", path, err)
	}
	return StripFrontmatter(string(data)), nil
}

func resolveHEAD(repoPath string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("context: open repo for HEAD: %w", err)
	}
	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("context: resolve HEAD: %w", err)
	}
	return ref.Hash().String(), nil
}
