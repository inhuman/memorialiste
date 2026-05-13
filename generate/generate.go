// Package generate runs one LLM call to produce updated documentation
// Markdown from the existing doc body plus an assembled diff context.
package generate

import (
	"context"
	"fmt"

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
		{Role: "user", Content: buildUserContent(in.DocBody, in.Diff)},
	}
	if in.Prompt != "" {
		messages = append(messages, provider.Message{Role: "user", Content: in.Prompt})
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

// buildUserContent joins the existing doc body and the diff context into one
// user message. A `---` separator delimits the two sections so the model can
// tell them apart.
func buildUserContent(docBody, diff string) string {
	if docBody == "" {
		return diff
	}
	if diff == "" {
		return docBody
	}
	return docBody + "\n\n---\n\n" + diff
}
