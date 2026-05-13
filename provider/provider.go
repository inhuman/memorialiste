package provider

import "context"

// Message is one chat turn sent to or received from the LLM.
type Message struct {
	// Role is "system", "user", or "assistant".
	Role string
	// Content is the message body.
	Content string
}

// TokenUsage reports token counts for one completion call.
// Zero values are returned when the provider omits the usage block.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Provider sends chat messages to an LLM and returns the assistant reply.
//
// Implementations live under provider/* (one per adapter) and a fake is
// available in internal/fake for unit tests.
type Provider interface {
	Complete(ctx context.Context, messages []Message) (string, TokenUsage, error)
}
