// Command memorialiste updates documentation from git diffs using an LLM.
//
// This is a minimal CLI scaffold — it wires manifest → context → generate
// and writes output to disk. Full flag coverage, platform integration
// (GitLab/GitHub MR/PR), and commit creation are still pending (US3 + US4).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	mctx "github.com/inhuman/memorialiste/context"
	"github.com/inhuman/memorialiste/generate"
	"github.com/inhuman/memorialiste/manifest"
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
		dryRun       = flag.Bool("dry-run", true, "Write files locally; skip commit/MR (always true in this MVP build)")
	)
	flag.Parse()

	if !*dryRun {
		log.Println("warning: only --dry-run is supported in this build; behaving as if --dry-run=true")
	}

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

	// Process each doc entry
	for i, entry := range m.Docs {
		entryPath := entry.Path
		if !filepath.IsAbs(entryPath) {
			entryPath = filepath.Join(*repoPath, entryPath)
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

		// Write output with new frontmatter
		final := mctx.WriteFrontmatter(result.Content, dc.HeadSHA)
		if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
			log.Fatalf("mkdir for %q: %v", entryPath, err)
		}
		if err := os.WriteFile(entryPath, []byte(final), 0o644); err != nil {
			log.Fatalf("write %q: %v", entryPath, err)
		}
		log.Printf("[%d/%d] wrote %s", i+1, len(m.Docs), entry.Path)
	}

	fmt.Println("done")
}
