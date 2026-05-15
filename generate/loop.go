package generate

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/inhuman/memorialiste/provider"
)

// DefaultMaxTurns is the default tool-call loop ceiling.
const DefaultMaxTurns = 10

// runToolLoop drives the multi-turn tool-call loop. It returns the final
// model text (with Strip applied) and aggregated token usage.
func runToolLoop(ctx context.Context, in Input, p provider.ToolingProvider, messages []provider.Message) (string, provider.TokenUsage, error) {
	maxTurns := cmp.Or(in.MaxTurns, DefaultMaxTurns)
	tools := []provider.ToolSchema{SearchCodeSchema}
	var aggregate provider.TokenUsage

	for turn := 1; turn <= maxTurns; turn++ {
		step, usage, err := p.CompleteWithTools(ctx, messages, tools)
		aggregate.PromptTokens += usage.PromptTokens
		aggregate.CompletionTokens += usage.CompletionTokens
		aggregate.TotalTokens += usage.TotalTokens
		if err != nil {
			if errors.Is(err, provider.ErrToolsNotSupported) {
				return "", aggregate, fmt.Errorf("generate: provider does not support function calling; set --code-search=false: %w", err)
			}
			return "", aggregate, fmt.Errorf("generate: tool-call: %w", err)
		}
		if len(step.ToolCalls) == 0 {
			return Strip(step.Content), aggregate, nil
		}

		messages = append(messages, provider.Message{
			Role:      "assistant",
			Content:   step.Content,
			ToolCalls: step.ToolCalls,
		})
		for _, call := range step.ToolCalls {
			log.Printf("code-search: turn=%d name=%s args=%s", turn, call.Name, call.Arguments)
			result := dispatchSearchCode(ctx, call, in.RepoRoot)
			messages = append(messages, provider.Message{
				Role:       "tool",
				Content:    result.Content,
				ToolCallID: result.CallID,
			})
		}
	}
	return "", aggregate, fmt.Errorf("generate: tool-call loop exceeded MaxTurns (%d)", maxTurns)
}
