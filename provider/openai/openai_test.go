package openai_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/inhuman/memorialiste/provider"
	"github.com/inhuman/memorialiste/provider/openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const goodResponse = `{
  "choices": [{"message": {"role": "assistant", "content": "hello world"}}],
  "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
}`

func TestComplete_RequestBodyShape(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(goodResponse))
	}))
	defer srv.Close()

	p := openai.New(openai.Config{BaseURL: srv.URL, Model: "qwen3:8b"})
	_, _, err := p.Complete(t.Context(), []provider.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "usr"},
	})
	require.NoError(t, err)

	assert.Equal(t, "qwen3:8b", captured["model"])
	assert.Equal(t, false, captured["stream"])
	msgs, ok := captured["messages"].([]any)
	require.True(t, ok)
	assert.Len(t, msgs, 2)
}

func TestComplete_BearerAuth_WhenAPIKeySet(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(goodResponse))
	}))
	defer srv.Close()

	p := openai.New(openai.Config{BaseURL: srv.URL, Model: "m", APIKey: "sk-test"})
	_, _, err := p.Complete(t.Context(), []provider.Message{{Role: "user", Content: "x"}})
	require.NoError(t, err)
	assert.Equal(t, "Bearer sk-test", authHeader)
}

func TestComplete_NoAuth_WhenAPIKeyEmpty(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(goodResponse))
	}))
	defer srv.Close()

	p := openai.New(openai.Config{BaseURL: srv.URL, Model: "m"})
	_, _, err := p.Complete(t.Context(), []provider.Message{{Role: "user", Content: "x"}})
	require.NoError(t, err)
	assert.Empty(t, authHeader)
}

func TestComplete_ModelParamsMergedAtTopLevel(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(goodResponse))
	}))
	defer srv.Close()

	p := openai.New(openai.Config{
		BaseURL:     srv.URL,
		Model:       "m",
		ModelParams: json.RawMessage(`{"temperature": 0.2, "top_p": 0.9}`),
	})
	_, _, err := p.Complete(t.Context(), []provider.Message{{Role: "user", Content: "x"}})
	require.NoError(t, err)

	assert.InDelta(t, 0.2, captured["temperature"], 1e-9)
	assert.InDelta(t, 0.9, captured["top_p"], 1e-9)
	// reserved keys overridden by adapter
	assert.Equal(t, "m", captured["model"])
}

func TestComplete_ReservedKeysOverridden(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(goodResponse))
	}))
	defer srv.Close()

	p := openai.New(openai.Config{
		BaseURL:     srv.URL,
		Model:       "real-model",
		ModelParams: json.RawMessage(`{"model": "hijacked", "stream": true}`),
	})
	_, _, err := p.Complete(t.Context(), []provider.Message{{Role: "user", Content: "x"}})
	require.NoError(t, err)

	assert.Equal(t, "real-model", captured["model"], "adapter must override hijacked model")
	assert.Equal(t, false, captured["stream"], "adapter must force stream=false")
}

func TestComplete_ParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(goodResponse))
	}))
	defer srv.Close()

	p := openai.New(openai.Config{BaseURL: srv.URL, Model: "m"})
	content, usage, err := p.Complete(t.Context(), []provider.Message{{Role: "user", Content: "x"}})
	require.NoError(t, err)
	assert.Equal(t, "hello world", content)
	assert.Equal(t, 10, usage.PromptTokens)
	assert.Equal(t, 5, usage.CompletionTokens)
	assert.Equal(t, 15, usage.TotalTokens)
}

func TestComplete_Non2xx_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("model overloaded"))
	}))
	defer srv.Close()

	p := openai.New(openai.Config{BaseURL: srv.URL, Model: "m"})
	_, _, err := p.Complete(t.Context(), []provider.Message{{Role: "user", Content: "x"}})
	require.Error(t, err)

	var httpErr *openai.HTTPError
	require.True(t, errors.As(err, &httpErr))
	assert.Equal(t, 500, httpErr.StatusCode)
	assert.Contains(t, httpErr.Body, "model overloaded")
}

func TestComplete_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(goodResponse))
	}))
	defer srv.Close()

	p := openai.New(openai.Config{
		BaseURL: srv.URL,
		Model:   "m",
		Timeout: 50 * time.Millisecond,
	})
	_, _, err := p.Complete(context.Background(), []provider.Message{{Role: "user", Content: "x"}})
	require.Error(t, err)
	// either DeadlineExceeded surfaces through net/http or it's wrapped
	assert.True(t,
		errors.Is(err, context.DeadlineExceeded) || assert.Contains(t, err.Error(), "deadline"),
		"expected deadline error, got: %v", err,
	)
}

func TestComplete_EmptyUsage_ZeroValuedTokenUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	p := openai.New(openai.Config{BaseURL: srv.URL, Model: "m"})
	_, usage, err := p.Complete(t.Context(), []provider.Message{{Role: "user", Content: "x"}})
	require.NoError(t, err)
	assert.Equal(t, provider.TokenUsage{}, usage)
}

func TestComplete_NoChoices_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	p := openai.New(openai.Config{BaseURL: srv.URL, Model: "m"})
	_, _, err := p.Complete(t.Context(), []provider.Message{{Role: "user", Content: "x"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}
