package ollama

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/inhuman/memorialiste/provider"
	"github.com/inhuman/memorialiste/provider/openai"
)

// DefaultBaseURL is the OpenAI-compat endpoint Ollama exposes by default.
const DefaultBaseURL = "http://localhost:11434"

// Config configures the Ollama adapter. No API key — Ollama is auth-less.
type Config struct {
	// BaseURL defaults to DefaultBaseURL when empty.
	BaseURL string
	// Model is the Ollama model tag, e.g. "qwen3:8b".
	Model string
	// ModelParams is merged into the request body.
	ModelParams json.RawMessage
	// Timeout caps each request; defaults to 5 minutes when zero.
	Timeout time.Duration
	// HTTPClient is optional; defaults to &http.Client{} when nil.
	HTTPClient *http.Client
}

// New constructs an Ollama Provider.
func New(cfg Config) provider.Provider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	return openai.New(openai.Config{
		BaseURL:     cfg.BaseURL,
		Model:       cfg.Model,
		ModelParams: cfg.ModelParams,
		Timeout:     cfg.Timeout,
		HTTPClient:  cfg.HTTPClient,
		// APIKey intentionally omitted — Ollama is auth-less.
	})
}
