// Package generate runs one LLM call to produce updated documentation
// Markdown from the existing doc body plus an assembled diff context.
package generate

import (
	"context"
	"fmt"
	"time"

	"github.com/inhuman/memorialiste/provider"
)

// Input is everything needed for one generation call.
type Input struct {
	// DocBody is the existing doc content with frontmatter stripped.
	DocBody string
	// Diff is the assembled diff context (raw, summarised, or AST-enriched).
	Diff string
	// Language is the target output language; substituted into the
	// {language} placeholder in the system prompt.
	Language string
	// Prompt is an optional extra user prompt appended after the diff
	// context as a separate user message.
	Prompt string
	// SystemPrompt is the system prompt source:
	//   - empty   → use built-in default
	//   - "@path" → load contents from path
	//   - other   → literal string
	SystemPrompt string
	// RepoMeta is a pre-formatted repository metadata block prepended to
	// the user message ahead of DocBody and Diff. May be empty.
	RepoMeta string
	// CodeSearch enables the AST code-search tool-call loop when the
	// provider implements provider.ToolingProvider. Default false preserves
	// the legacy single-shot path with zero overhead.
	CodeSearch bool
	// MaxTurns caps the number of tool-call turns; 0 → DefaultMaxTurns.
	MaxTurns int
	// RepoRoot anchors the search_code tool to the local repository. Required
	// when CodeSearch is true.
	RepoRoot string
	// ASTParseTimeout caps a single parser.ParseFile call inside the
	// search_code tool. 0 → codesearch.DefaultParseTimeout.
	ASTParseTimeout time.Duration
}

// Result is the output of a generation call.
type Result struct {
	// Content is the clean Markdown body with LLM artifacts stripped.
	Content string
	// TokenUsage is the token accounting reported by the provider.
	TokenUsage provider.TokenUsage
}

// Generate runs one LLM completion with the given input and provider.
//
// It loads the system prompt (resolving @file and substituting {language}),
// assembles the user messages (doc body + diff, then optional extra prompt),
// invokes p.Complete, strips artifacts from the response, and returns the
// clean Markdown alongside the provider's token usage.
func Generate(ctx context.Context, in Input, p provider.Provider) (*Result, error) {
	if p == nil {
		return nil, fmt.Errorf("generate: provider is nil")
	}

	system, err := loadSystemPrompt(in.SystemPrompt, in.Language)
	if err != nil {
		return nil, err
	}

	messages := []provider.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: buildUserContent(in.RepoMeta, in.DocBody, in.Diff)},
	}
	if in.Prompt != "" {
		messages = append(messages, provider.Message{Role: "user", Content: in.Prompt})
	}

	if in.CodeSearch {
		if tp, ok := p.(provider.ToolingProvider); ok {
			content, usage, err := runToolLoop(ctx, in, tp, messages)
			if err != nil {
				return nil, err
			}
			return &Result{Content: content, TokenUsage: usage}, nil
		}
	}

	content, usage, err := p.Complete(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("generate: provider call: %w", err)
	}

	return &Result{
		Content:    Strip(content),
		TokenUsage: usage,
	}, nil
}

// buildUserContent joins optional repo metadata, the existing doc body, and
// the diff context into one user message. A `---` separator delimits the doc
// body and diff sections so the model can tell them apart. When non-empty,
// the metadata block is prepended ahead of both, separated by a blank line.
func buildUserContent(repoMeta, docBody, diff string) string {
	var body string
	switch {
	case docBody == "":
		body = diff
	case diff == "":
		body = docBody
	default:
		body = docBody + "\n\n---\n\n" + diff
	}
	if repoMeta == "" {
		return body
	}
	if body == "" {
		return repoMeta
	}
	return repoMeta + "\n\n" + body
}
