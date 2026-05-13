---
generated_at: f75a840644baa7f6379ec00a9d2be57e235a8a1f
---

# Context Assembly

The context package assembles the diff context for a single documentation entry. It reads the generated_at watermark from the doc file's YAML frontmatter, computes a filtered git diff scoped to the entry's covered paths, enforces a token budget, and optionally summarises large diffs via an injected Summariser.

## Options

The `Options` struct configures the context assembly for one run.

```go
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
}
```

## DiffContext

The `DiffContext` struct holds the assembled context for one doc entry.

```go
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
}
```

## Assemble

`Assemble` builds a DiffContext for the given DocEntry.

```go
func Assemble(ctx context.Context, entry manifest.DocEntry, opts Options) (*DiffContext, error)
```

It opens the git repository at opts.RepoPath, reads the watermark from entry.Path, computes the filtered diff, and enforces the token budget.

## AST Context

When `opts.ASTContext` is true, the diff is enriched with AST scope information. This feature is opt-in and requires the `grep-ast` tool to be available. The `ASTAnnotator` interface is used to perform the annotation.

```go
type ASTAnnotator interface {
    // Annotate returns the AST annotation for filePath.
    // changedLines contains 1-based line numbers that appear in the diff.
    // On timeout or analysis failure it returns an empty annotation and nil
    // error — the caller falls back to the unenriched diff.
    Annotate(ctx context.Context, filePath string, changedLines []int) (ASTAnnotation, error)
}

type ASTAnnotation struct {
    // FilePath is the repo-relative path of the file.
    FilePath string
    // Rendered is the full grep-ast TreeContext rendering of the file with
    // changed lines marked. Empty when the file is unsupported or the
    // renderer failed — callers should fall back to the raw diff in that case.
    Rendered string
}
```

When AST context is enabled, `enrichDiff` is used to process the diff and replace the raw diff hunks with AST-enriched renders where available. When a file's annotation has non-empty Rendered, the rendering replaces the raw diff hunks for that file (TreeContext already shows changed lines in context). Otherwise the raw diff is emitted as-is.

## Token Budget

The `ApproxTokens` function estimates the token count of a string using the formula `len(s)/4`.

```go
func ApproxTokens(s string) int
```