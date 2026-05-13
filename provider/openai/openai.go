package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/inhuman/memorialiste/provider"
)

const (
	defaultTimeout    = 5 * time.Minute
	completionsPath   = "/v1/chat/completions"
	bodyExcerptMaxLen = 512
)

// Config configures a generic OpenAI-compatible adapter.
type Config struct {
	// BaseURL is the provider root URL, e.g. "http://localhost:11434".
	BaseURL string
	// Model is the model tag passed in the request body.
	Model string
	// APIKey is sent as `Authorization: Bearer <key>` when non-empty.
	APIKey string
	// ModelParams is merged into the top-level request body before reserved
	// keys (model, messages, stream) are stamped by the adapter.
	ModelParams json.RawMessage
	// Timeout caps each request via context.WithTimeout.
	// Defaults to 5 minutes when zero.
	Timeout time.Duration
	// HTTPClient is optional; defaults to &http.Client{} when nil.
	HTTPClient *http.Client
}

// New constructs an OpenAI-compatible Provider.
func New(cfg Config) provider.Provider {
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{}
	}
	return &client{cfg: cfg}
}

type client struct {
	cfg Config
}

// chatRequest matches the OpenAI /v1/chat/completions request body shape.
type chatRequest struct {
	Model    string             `json:"model"`
	Messages []provider.Message `json:"messages"`
	Stream   bool               `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Complete implements provider.Provider.
func (c *client) Complete(ctx context.Context, messages []provider.Message) (string, provider.TokenUsage, error) {
	body, err := c.buildBody(messages)
	if err != nil {
		return "", provider.TokenUsage{}, fmt.Errorf("openai: build request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	url := strings.TrimRight(c.cfg.BaseURL, "/") + completionsPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", provider.TokenUsage{}, fmt.Errorf("openai: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return "", provider.TokenUsage{}, fmt.Errorf("openai: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", provider.TokenUsage{}, fmt.Errorf("openai: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", provider.TokenUsage{}, &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       truncate(string(respBody), bodyExcerptMaxLen),
		}
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", provider.TokenUsage{}, fmt.Errorf("openai: parse response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", provider.TokenUsage{}, fmt.Errorf("openai: response has no choices")
	}

	usage := provider.TokenUsage{
		PromptTokens:     parsed.Usage.PromptTokens,
		CompletionTokens: parsed.Usage.CompletionTokens,
		TotalTokens:      parsed.Usage.TotalTokens,
	}
	return parsed.Choices[0].Message.Content, usage, nil
}

// buildBody merges ModelParams (user-supplied) into the request body, then
// stamps reserved keys (model, messages, stream). On conflict the reserved
// key wins and a warning is logged.
func (c *client) buildBody(messages []provider.Message) ([]byte, error) {
	body := map[string]any{}
	if len(c.cfg.ModelParams) > 0 {
		if err := json.Unmarshal(c.cfg.ModelParams, &body); err != nil {
			return nil, fmt.Errorf("model-params JSON: %w", err)
		}
	}
	for _, reserved := range []string{"model", "messages", "stream"} {
		if _, taken := body[reserved]; taken {
			log.Printf("openai: model-params override of reserved key %q ignored", reserved)
		}
	}
	body["model"] = c.cfg.Model
	body["messages"] = messages
	body["stream"] = false

	return json.Marshal(body)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
