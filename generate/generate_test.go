package generate_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/inhuman/memorialiste/generate"
	"github.com/inhuman/memorialiste/internal/fake"
	"github.com/inhuman/memorialiste/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_MessagesAssembledInCorrectOrder(t *testing.T) {
	var captured []provider.Message
	fp := &fake.Provider{
		CompleteFunc: func(_ context.Context, msgs []provider.Message) (string, provider.TokenUsage, error) {
			captured = msgs
			return "OK", provider.TokenUsage{}, nil
		},
	}

	_, err := generate.Generate(context.Background(), generate.Input{
		DocBody:  "# Doc",
		Diff:     "=== file.go ===\n+ change",
		Language: "english",
		Prompt:   "Be concise.",
	}, fp)
	require.NoError(t, err)

	require.Len(t, captured, 3)
	assert.Equal(t, "system", captured[0].Role)
	assert.Equal(t, "user", captured[1].Role)
	assert.Equal(t, "user", captured[2].Role)
	assert.Contains(t, captured[1].Content, "# Doc")
	assert.Contains(t, captured[1].Content, "=== file.go ===")
	assert.Equal(t, "Be concise.", captured[2].Content)
}

func TestGenerate_LanguageSubstituted(t *testing.T) {
	var systemContent string
	fp := &fake.Provider{
		CompleteFunc: func(_ context.Context, msgs []provider.Message) (string, provider.TokenUsage, error) {
			systemContent = msgs[0].Content
			return "OK", provider.TokenUsage{}, nil
		},
	}

	_, err := generate.Generate(context.Background(), generate.Input{
		Language: "russian",
	}, fp)
	require.NoError(t, err)

	assert.Contains(t, systemContent, "Write in russian.")
	assert.NotContains(t, systemContent, "{language}")
}

func TestGenerate_NoExtraPrompt_TwoMessages(t *testing.T) {
	var msgCount int
	fp := &fake.Provider{
		CompleteFunc: func(_ context.Context, msgs []provider.Message) (string, provider.TokenUsage, error) {
			msgCount = len(msgs)
			return "OK", provider.TokenUsage{}, nil
		},
	}

	_, err := generate.Generate(context.Background(), generate.Input{
		DocBody: "doc",
		Diff:    "diff",
		// Prompt empty → no extra user message
	}, fp)
	require.NoError(t, err)
	assert.Equal(t, 2, msgCount)
}

func TestGenerate_StripsArtifacts(t *testing.T) {
	fp := &fake.Provider{
		CompleteFunc: func(_ context.Context, _ []provider.Message) (string, provider.TokenUsage, error) {
			return "Here's the updated documentation:\n```markdown\n# Title\n\nBody\n```", provider.TokenUsage{TotalTokens: 100}, nil
		},
	}

	result, err := generate.Generate(context.Background(), generate.Input{}, fp)
	require.NoError(t, err)
	assert.Equal(t, "# Title\n\nBody", result.Content)
	assert.Equal(t, 100, result.TokenUsage.TotalTokens)
}

func TestGenerate_EmptyDiff_StillProducesCall(t *testing.T) {
	called := false
	fp := &fake.Provider{
		CompleteFunc: func(_ context.Context, _ []provider.Message) (string, provider.TokenUsage, error) {
			called = true
			return "", provider.TokenUsage{}, nil
		},
	}

	result, err := generate.Generate(context.Background(), generate.Input{
		DocBody: "# Doc",
		Diff:    "",
	}, fp)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Empty(t, result.Content)
}

func TestGenerate_ProviderErrorPropagated(t *testing.T) {
	wantErr := errors.New("boom")
	fp := &fake.Provider{
		CompleteFunc: func(_ context.Context, _ []provider.Message) (string, provider.TokenUsage, error) {
			return "", provider.TokenUsage{}, wantErr
		},
	}

	_, err := generate.Generate(context.Background(), generate.Input{}, fp)
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

func TestGenerate_NilProvider_Error(t *testing.T) {
	_, err := generate.Generate(context.Background(), generate.Input{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider is nil")
}

func TestGenerate_RepoMetaPrepended(t *testing.T) {
	var userContent string
	fp := &fake.Provider{
		CompleteFunc: func(_ context.Context, msgs []provider.Message) (string, provider.TokenUsage, error) {
			userContent = msgs[1].Content
			return "OK", provider.TokenUsage{}, nil
		},
	}

	const meta = "=== Repository metadata ===\nLatest tag: vX\n=== End metadata ==="
	_, err := generate.Generate(context.Background(), generate.Input{
		RepoMeta: meta,
		DocBody:  "DOC-BODY",
		Diff:     "DIFF-CONTENT",
	}, fp)
	require.NoError(t, err)
	metaIdx := strings.Index(userContent, "=== Repository metadata ===")
	docIdx := strings.Index(userContent, "DOC-BODY")
	require.GreaterOrEqual(t, metaIdx, 0)
	require.Greater(t, docIdx, metaIdx, "expected meta block before doc body")
}

func TestGenerate_DocBodyAndDiffSeparated(t *testing.T) {
	var userContent string
	fp := &fake.Provider{
		CompleteFunc: func(_ context.Context, msgs []provider.Message) (string, provider.TokenUsage, error) {
			userContent = msgs[1].Content
			return "OK", provider.TokenUsage{}, nil
		},
	}

	_, err := generate.Generate(context.Background(), generate.Input{
		DocBody: "DOC-BODY",
		Diff:    "DIFF-CONTENT",
	}, fp)
	require.NoError(t, err)
	idx := strings.Index(userContent, "---")
	assert.Positive(t, idx, "expected --- separator between doc body and diff")
	assert.Contains(t, userContent[:idx], "DOC-BODY")
	assert.Contains(t, userContent[idx:], "DIFF-CONTENT")
}
