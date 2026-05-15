package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/inhuman/memorialiste/provider"
)

var toolsRejectRE = regexp.MustCompile(`(?i)tool|function`)

// CompleteWithTools implements provider.ToolingProvider. It performs one
// round-trip with the model exposing the supplied tools and parses any
// tool_calls or final content from the response.
func (c *client) CompleteWithTools(ctx context.Context, messages []provider.Message, tools []provider.ToolSchema) (provider.Step, provider.TokenUsage, error) {
	body, err := c.buildToolsBody(messages, tools)
	if err != nil {
		return provider.Step{}, provider.TokenUsage{}, fmt.Errorf("openai: build tools request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	url := strings.TrimRight(c.cfg.BaseURL, "/") + completionsPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return provider.Step{}, provider.TokenUsage{}, fmt.Errorf("openai: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return provider.Step{}, provider.TokenUsage{}, fmt.Errorf("openai: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return provider.Step{}, provider.TokenUsage{}, fmt.Errorf("openai: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		httpErr := &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       truncate(string(respBody), bodyExcerptMaxLen),
		}
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && toolsRejectRE.MatchString(httpErr.Body) {
			return provider.Step{}, provider.TokenUsage{}, fmt.Errorf("openai: tools rejected: %w (HTTP %d: %s)", provider.ErrToolsNotSupported, httpErr.StatusCode, httpErr.Body)
		}
		return provider.Step{}, provider.TokenUsage{}, httpErr
	}

	var parsed toolsResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return provider.Step{}, provider.TokenUsage{}, fmt.Errorf("openai: parse tools response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return provider.Step{}, provider.TokenUsage{}, errors.New("openai: response has no choices")
	}

	choice := parsed.Choices[0]
	step := provider.Step{Content: choice.Message.Content}
	for _, tc := range choice.Message.ToolCalls {
		step.ToolCalls = append(step.ToolCalls, provider.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	usage := provider.TokenUsage{
		PromptTokens:     parsed.Usage.PromptTokens,
		CompletionTokens: parsed.Usage.CompletionTokens,
		TotalTokens:      parsed.Usage.TotalTokens,
	}
	return step, usage, nil
}

func (c *client) buildToolsBody(messages []provider.Message, tools []provider.ToolSchema) ([]byte, error) {
	body := map[string]any{}
	if len(c.cfg.ModelParams) > 0 {
		if err := json.Unmarshal(c.cfg.ModelParams, &body); err != nil {
			return nil, fmt.Errorf("model-params JSON: %w", err)
		}
	}
	for _, reserved := range []string{"model", "messages", "stream", "tools", "tool_choice"} {
		if _, taken := body[reserved]; taken {
			log.Printf("openai: model-params override of reserved key %q ignored", reserved)
		}
	}
	body["model"] = c.cfg.Model
	body["messages"] = marshalToolMessages(messages)
	body["stream"] = false

	toolDefs := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		toolDefs = append(toolDefs, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		})
	}
	body["tools"] = toolDefs
	body["tool_choice"] = "auto"
	return json.Marshal(body)
}

// marshalToolMessages converts []provider.Message into the OpenAI Chat
// Completions wire shape, including tool_calls on assistant messages and
// tool_call_id on tool messages when present.
func marshalToolMessages(messages []provider.Message) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		entry := map[string]any{"role": m.Role}
		if m.Role == "tool" {
			entry["content"] = m.Content
			entry["tool_call_id"] = m.ToolCallID
			out = append(out, entry)
			continue
		}
		if len(m.ToolCalls) > 0 {
			calls := make([]map[string]any, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				calls = append(calls, map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Name,
						"arguments": tc.Arguments,
					},
				})
			}
			entry["tool_calls"] = calls
			if m.Content != "" {
				entry["content"] = m.Content
			} else {
				entry["content"] = nil
			}
			out = append(out, entry)
			continue
		}
		entry["content"] = m.Content
		out = append(out, entry)
	}
	return out
}

// toolsResponse mirrors the OpenAI Chat Completions response when tools
// are involved.
type toolsResponse struct {
	Choices []struct {
		Message struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}
