package openai_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/inhuman/memorialiste/provider"
	"github.com/inhuman/memorialiste/provider/openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const toolCallResponse = `{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": null,
      "tool_calls": [
        {"id":"call_1","type":"function","function":{"name":"search_code","arguments":"{\"pattern\":\"Foo\"}"}}
      ]
    },
    "finish_reason": "tool_calls"
  }],
  "usage": {"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}
}`

const finalContentResponse = `{
  "choices": [{"message":{"role":"assistant","content":"final answer"}, "finish_reason":"stop"}],
  "usage": {"prompt_tokens":20,"completion_tokens":10,"total_tokens":30}
}`

func toolingClient(t *testing.T, srv *httptest.Server) provider.ToolingProvider {
	t.Helper()
	p := openai.New(openai.Config{BaseURL: srv.URL, Model: "test-model"})
	tp, ok := p.(provider.ToolingProvider)
	require.True(t, ok, "openai client must implement ToolingProvider")
	return tp
}

func TestTools_HappyPath_ParsesToolCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(toolCallResponse))
	}))
	defer srv.Close()
	tp := toolingClient(t, srv)

	step, usage, err := tp.CompleteWithTools(t.Context(),
		[]provider.Message{{Role: "user", Content: "doc this"}},
		[]provider.ToolSchema{{Name: "search_code", Description: "search", Parameters: map[string]any{"type": "object"}}})
	require.NoError(t, err)
	require.Len(t, step.ToolCalls, 1)
	assert.Equal(t, "call_1", step.ToolCalls[0].ID)
	assert.Equal(t, "search_code", step.ToolCalls[0].Name)
	assert.Contains(t, step.ToolCalls[0].Arguments, "Foo")
	assert.Equal(t, 15, usage.TotalTokens)
}

func TestTools_RequestShape(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(finalContentResponse))
	}))
	defer srv.Close()
	tp := toolingClient(t, srv)

	_, _, err := tp.CompleteWithTools(t.Context(),
		[]provider.Message{{Role: "user", Content: "x"}},
		[]provider.ToolSchema{{Name: "search_code", Description: "d", Parameters: map[string]any{"type": "object"}}})
	require.NoError(t, err)

	assert.Equal(t, "auto", captured["tool_choice"])
	assert.Equal(t, false, captured["stream"])
	tools, ok := captured["tools"].([]any)
	require.True(t, ok)
	require.Len(t, tools, 1)
	tool := tools[0].(map[string]any)
	assert.Equal(t, "function", tool["type"])
	fn := tool["function"].(map[string]any)
	assert.Equal(t, "search_code", fn["name"])
}

func TestTools_SecondTurn_PassesAssistantAndToolMessages(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(finalContentResponse))
	}))
	defer srv.Close()
	tp := toolingClient(t, srv)

	messages := []provider.Message{
		{Role: "user", Content: "doc this"},
		{Role: "assistant", ToolCalls: []provider.ToolCall{{ID: "call_1", Name: "search_code", Arguments: `{"pattern":"Foo"}`}}},
		{Role: "tool", Content: `{"hits":[]}`, ToolCallID: "call_1"},
	}
	step, _, err := tp.CompleteWithTools(t.Context(), messages,
		[]provider.ToolSchema{{Name: "search_code", Description: "d", Parameters: map[string]any{}}})
	require.NoError(t, err)
	assert.Equal(t, "final answer", step.Content)

	msgs, ok := captured["messages"].([]any)
	require.True(t, ok)
	require.Len(t, msgs, 3)
	assistant := msgs[1].(map[string]any)
	assert.Equal(t, "assistant", assistant["role"])
	tcs, ok := assistant["tool_calls"].([]any)
	require.True(t, ok)
	require.Len(t, tcs, 1)
	assert.Equal(t, "call_1", tcs[0].(map[string]any)["id"])

	toolMsg := msgs[2].(map[string]any)
	assert.Equal(t, "tool", toolMsg["role"])
	assert.Equal(t, "call_1", toolMsg["tool_call_id"])
}

func TestTools_Unsupported_ToolsKeyword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"tools not supported"}`))
	}))
	defer srv.Close()
	tp := toolingClient(t, srv)

	_, _, err := tp.CompleteWithTools(t.Context(),
		[]provider.Message{{Role: "user", Content: "x"}},
		[]provider.ToolSchema{{Name: "search_code"}})
	require.Error(t, err)
	assert.True(t, errors.Is(err, provider.ErrToolsNotSupported), "expected ErrToolsNotSupported, got %v", err)
}

func TestTools_Unsupported_FunctionKeyword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"function calling unsupported"}`))
	}))
	defer srv.Close()
	tp := toolingClient(t, srv)

	_, _, err := tp.CompleteWithTools(t.Context(),
		[]provider.Message{{Role: "user", Content: "x"}},
		[]provider.ToolSchema{{Name: "search_code"}})
	require.Error(t, err)
	assert.True(t, errors.Is(err, provider.ErrToolsNotSupported), "expected ErrToolsNotSupported")
}

func TestTools_GenericNon2xx_NotMarkedUnsupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"server boom"}`))
	}))
	defer srv.Close()
	tp := toolingClient(t, srv)

	_, _, err := tp.CompleteWithTools(t.Context(),
		[]provider.Message{{Role: "user", Content: "x"}},
		[]provider.ToolSchema{{Name: "search_code"}})
	require.Error(t, err)
	assert.False(t, errors.Is(err, provider.ErrToolsNotSupported))
	var httpErr *openai.HTTPError
	assert.True(t, errors.As(err, &httpErr), "expected *HTTPError, got %v", err)
}
