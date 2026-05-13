// Command memorialiste updates documentation from git diffs using an LLM.
//
// Pipeline: manifest → context (diff + AST) → generate (LLM) → output (write + commit).
// Platform integration (pushing branches, opening MR/PR) is pending (US4).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"path/filepath"

	mctx "github.com/inhuman/memorialiste/context"
	"github.com/inhuman/memorialiste/generate"
	"github.com/inhuman/memorialiste/manifest"
	"github.com/inhuman/memorialiste/output"
	"github.com/inhuman/memorialiste/provider/openai"
)

func main() {
	var (
		providerURL  = flag.String("provider-url", "http://localhost:11434", "OpenAI-compatible base URL")
		model        = flag.String("model", "qwen3-coder:30b", "Model tag")
		modelParams  = flag.String("model-params", "", "Extra model parameters as JSON object")
		systemPrompt = flag.String("system-prompt", "", "System prompt: literal string, @path/to/file, or empty for built-in")
		extraPrompt  = flag.String("prompt", "", "Additional user prompt appended after diff context")
		language     = flag.String("language", "english", "Output language")
		docStructure = flag.String("doc-structure", "docs/.docstructure.yaml", "Path to the doc structure manifest")
		repoPath     = flag.String("repo", ".", "Path to the local git repository root")
		apiKey       = flag.String("api-key", "", "Bearer token for the LLM provider (optional)")
		tokenBudget  = flag.Int("token-budget", 12000, "Max tokens for diff context before summarisation")
		astContext   = flag.Bool("ast-context", false, "Enable AST-enriched diff context via grep-ast")
		dryRun       = flag.Bool("dry-run", true, "Write files locally; skip branch+commit when true")
		branchPrefix = flag.String("branch-prefix", output.DefaultBranchPrefix, "Prefix for the auto-generated branch name")
	)
	flag.Parse()

	ctx := context.Background()

	// Parse manifest
	m, err := manifest.Parse(*docStructure)
	if err != nil {
		log.Fatalf("manifest: %v", err)
	}
	log.Printf("loaded %d doc entries from %s", len(m.Docs), *docStructure)

	// Construct provider
	prov := openai.New(openai.Config{
		BaseURL:     *providerURL,
		Model:       *model,
		APIKey:      *apiKey,
		ModelParams: json.RawMessage(*modelParams),
	})

	// Collect generated entries for output.Apply
	var entries []output.Entry

	for i, entry := range m.Docs {
		entryPath := entry.Path
		if !filepath.IsAbs(entryPath) {
			entryPath = filepath.Join(*repoPath, entry.Path)
		}

		log.Printf("[%d/%d] %s — assembling context", i+1, len(m.Docs), entry.Path)
		dc, err := mctx.Assemble(ctx, manifest.DocEntry{
			Path:        entryPath,
			Covers:      entry.Covers,
			Audience:    entry.Audience,
			Description: entry.Description,
		}, mctx.Options{
			RepoPath:    *repoPath,
			TokenBudget: *tokenBudget,
			ASTContext:  *astContext,
		})
		if err != nil {
			log.Fatalf("assemble %q: %v", entry.Path, err)
		}
		log.Printf("[%d/%d] diff=%d chars, summarised=%v, ast=%v, head=%s",
			i+1, len(m.Docs), len(dc.Diff), dc.Summarised, dc.ASTEnriched, dc.HeadSHA[:8])

		if dc.Diff == "" {
			log.Printf("[%d/%d] no changes since watermark — skipping", i+1, len(m.Docs))
			continue
		}

		log.Printf("[%d/%d] calling LLM (%s)", i+1, len(m.Docs), *model)
		result, err := generate.Generate(ctx, generate.Input{
			DocBody:      dc.DocBody,
			Diff:         dc.Diff,
			Language:     *language,
			Prompt:       *extraPrompt,
			SystemPrompt: *systemPrompt,
		}, prov)
		if err != nil {
			log.Fatalf("generate %q: %v", entry.Path, err)
		}
		log.Printf("[%d/%d] LLM returned %d chars; tokens prompt=%d completion=%d total=%d",
			i+1, len(m.Docs), len(result.Content),
			result.TokenUsage.PromptTokens, result.TokenUsage.CompletionTokens, result.TokenUsage.TotalTokens)

		entries = append(entries, output.Entry{
			Path:    entry.Path,
			Body:    result.Content,
			HeadSHA: dc.HeadSHA,
		})
	}

	// Output stage: write files + (when not dry-run) branch + commit
	res, err := output.Apply(ctx, output.Options{
		RepoPath:     *repoPath,
		DryRun:       *dryRun,
		BranchPrefix: *branchPrefix,
	}, entries)
	if err != nil {
		log.Fatalf("output: %v", err)
	}

	log.Printf("wrote %d files (skipped %d)", len(res.WrittenFiles), len(res.SkippedEntries))
	if res.BranchName != "" {
		log.Printf("created branch %s with commit %s", res.BranchName, res.CommitSHA[:8])
	} else if !*dryRun && len(res.WrittenFiles) == 0 {
		log.Printf("nothing to update — no branch created")
	}

	fmt.Println("done")
}
