package provider

import (
	"context"
	"errors"
)

// ToolSchema describes a callable tool exposed to the LLM via the
// OpenAI-style Tools API.
type ToolSchema struct {
	// Name is the tool identifier (e.g. "search_code").
	Name string
	// Description is the human-readable purpose, surfaced to the model.
	Description string
	// Parameters is a JSON-schema object describing the tool's arguments.
	Parameters map[string]any
}

// ToolCall is a single model request to invoke a tool.
type ToolCall struct {
	// ID is provider-assigned and must be echoed in the corresponding ToolResult.
	ID string
	// Name is the tool identifier the model wants to invoke.
	Name string
	// Arguments is the raw JSON string supplied by the model.
	Arguments string
}

// ToolResult is what we send back to the model after executing a ToolCall.
type ToolResult struct {
	// CallID matches the originating ToolCall.ID.
	CallID string
	// Content is typically JSON-encoded tool output.
	Content string
}

// Step is one round-trip with the model when tools are involved. Either
// Content (final text) or ToolCalls is populated.
type Step struct {
	Content   string
	ToolCalls []ToolCall
}

// ToolingProvider extends Provider with function-calling support. Adapters
// that don't support tools simply don't implement this interface.
type ToolingProvider interface {
	Provider
	// CompleteWithTools sends messages + tools; returns one model step.
	// To continue the loop, the caller appends the assistant message and
	// tool results back into messages and invokes this method again.
	CompleteWithTools(ctx context.Context, messages []Message, tools []ToolSchema) (Step, TokenUsage, error)
}

// ErrToolsNotSupported signals the provider rejected a tools-shaped
// request (typically 400 with body mentioning tools/function calling).
// Wrapped by adapters; callers check via errors.Is.
var ErrToolsNotSupported = errors.New("provider: tools not supported by this endpoint")
