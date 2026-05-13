package generate_test

import (
	"context"
	"testing"

	"github.com/inhuman/memorialiste/generate"
	"github.com/inhuman/memorialiste/internal/fake"
	"github.com/inhuman/memorialiste/provider"
	"github.com/stretchr/testify/require"
)

// fakeNoOpProvider implements provider.Provider with a no-op completion.
// Used when the test only cares about pre-call behaviour (e.g. system prompt
// loading errors that surface before the provider is even invoked).
type fakeNoOpProvider struct{}

func (fakeNoOpProvider) Complete(_ context.Context, _ []provider.Message) (string, provider.TokenUsage, error) {
	return "", provider.TokenUsage{}, nil
}

// callGenerateAndCaptureSystem runs Generate and returns the system message
// the provider received. Used to verify system-prompt loading + substitution
// without coupling tests to internal package details.
func callGenerateAndCaptureSystem(t *testing.T, systemPrompt, language string) string {
	t.Helper()
	var captured string
	fp := &fake.Provider{
		CompleteFunc: func(_ context.Context, msgs []provider.Message) (string, provider.TokenUsage, error) {
			captured = msgs[0].Content
			return "OK", provider.TokenUsage{}, nil
		},
	}
	_, err := generate.Generate(t.Context(), generate.Input{
		SystemPrompt: systemPrompt,
		Language:     language,
	}, fp)
	require.NoError(t, err)
	return captured
}
