// Command memorialiste updates documentation from git diffs using an LLM.
//
// Pipeline: manifest → context (diff + AST) → generate (LLM) → output (write + commit)
// → platform (push + open MR/PR).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/inhuman/memorialiste/cliconfig"
	mctx "github.com/inhuman/memorialiste/context"
	"github.com/inhuman/memorialiste/generate"
	"github.com/inhuman/memorialiste/manifest"
	"github.com/inhuman/memorialiste/output"
	"github.com/inhuman/memorialiste/platform"
	"github.com/inhuman/memorialiste/platform/github"
	"github.com/inhuman/memorialiste/platform/gitlab"
	"github.com/inhuman/memorialiste/provider/openai"
)

func main() {
	cfg, err := cliconfig.Parse(os.Args[1:], os.Getenv)
	if err != nil {
		// *ValidationError already formats with "error: " prefix per line.
		var vErr *cliconfig.ValidationError
		if errors.As(err, &vErr) {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		os.Exit(1)
	}
	if err := run(context.Background(), cfg); err != nil {
		log.Fatalf("%v", err)
	}
}

func run(ctx context.Context, cfg *cliconfig.Config) error {
	m, err := manifest.Parse(cfg.DocStructure)
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	log.Printf("loaded %d doc entries from %s", len(m.Docs), cfg.DocStructure)

	prov := openai.New(openai.Config{
		BaseURL:     cfg.ProviderURL,
		Model:       cfg.Model,
		APIKey:      cfg.APIKey,
		ModelParams: json.RawMessage(cfg.ModelParams),
	})

	var entries []output.Entry

	for i, entry := range m.Docs {
		entryPath := entry.Path
		if !filepath.IsAbs(entryPath) {
			entryPath = filepath.Join(cfg.RepoPath, entry.Path)
		}

		log.Printf("[%d/%d] %s — assembling context", i+1, len(m.Docs), entry.Path)
		dc, err := mctx.Assemble(ctx, manifest.DocEntry{
			Path:        entryPath,
			Covers:      entry.Covers,
			Audience:    entry.Audience,
			Description: entry.Description,
		}, mctx.Options{
			RepoPath:      cfg.RepoPath,
			TokenBudget:   cfg.TokenBudget,
			ASTContext:    cfg.ASTContext,
			RepoMetaLevel: mctx.MetaLevel(cfg.RepoMeta),
		})
		if err != nil {
			return fmt.Errorf("assemble %q: %w", entry.Path, err)
		}
		log.Printf("[%d/%d] diff=%d chars, summarised=%v, ast=%v, head=%s",
			i+1, len(m.Docs), len(dc.Diff), dc.Summarised, dc.ASTEnriched, dc.HeadSHA[:8])

		var metaBlock string
		if dc.RepoMeta != nil {
			metaBlock = dc.RepoMeta.Format(mctx.MetaLevel(cfg.RepoMeta))
			log.Printf("[%d/%d] meta: tag=%s sha=%s level=%s",
				i+1, len(m.Docs), dc.RepoMeta.LatestTag, dc.RepoMeta.ShortSHA, cfg.RepoMeta)
		}

		if dc.Diff == "" {
			log.Printf("[%d/%d] no changes since watermark — skipping", i+1, len(m.Docs))
			continue
		}

		log.Printf("[%d/%d] calling LLM (%s)", i+1, len(m.Docs), cfg.Model)
		result, err := generate.Generate(ctx, generate.Input{
			DocBody:      dc.DocBody,
			Diff:         dc.Diff,
			Language:     cfg.Language,
			Prompt:       cfg.Prompt,
			SystemPrompt: cfg.SystemPrompt,
			RepoMeta:     metaBlock,
		}, prov)
		if err != nil {
			return fmt.Errorf("generate %q: %w", entry.Path, err)
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

	res, err := output.Apply(ctx, output.Options{
		RepoPath:     cfg.RepoPath,
		DryRun:       cfg.DryRun,
		BranchPrefix: cfg.BranchPrefix,
	}, entries)
	if err != nil {
		return fmt.Errorf("output: %w", err)
	}

	log.Printf("wrote %d files (skipped %d)", len(res.WrittenFiles), len(res.SkippedEntries))
	if res.BranchName != "" {
		log.Printf("created branch %s with commit %s", res.BranchName, res.CommitSHA[:8])
	} else if !cfg.DryRun && len(res.WrittenFiles) == 0 {
		log.Printf("nothing to update — no branch created")
	}

	if !cfg.DryRun && res.BranchName != "" {
		var plat platform.Platform
		switch cfg.Platform {
		case "gitlab":
			plat = gitlab.New(gitlab.Config{
				BaseURL:   cfg.PlatformURL,
				Token:     cfg.PlatformToken,
				ProjectID: cfg.ProjectID,
				RepoPath:  cfg.RepoPath,
			})
		case "github":
			plat = github.New(github.Config{
				BaseURL:    cfg.PlatformURL,
				Token:      cfg.PlatformToken,
				Repository: cfg.ProjectID,
				RepoPath:   cfg.RepoPath,
			})
		}

		log.Printf("pushing branch %s to %s", res.BranchName, cfg.Platform)
		if err := plat.Push(ctx, res.BranchName, res.CommitSHA); err != nil {
			return fmt.Errorf("push: %w", err)
		}

		log.Printf("opening MR/PR against %s", cfg.BaseBranch)
		cr, err := plat.OpenChangeRequest(ctx, platform.ChangeRequest{
			SourceBranch: res.BranchName,
			TargetBranch: cfg.BaseBranch,
			Title:        res.CommitSubject,
			Body:         res.CommitBody,
		})
		if err != nil {
			return fmt.Errorf("open MR/PR: %w", err)
		}
		log.Printf("opened: %s", cr.URL)
	}

	fmt.Println("done")
	return nil
}
