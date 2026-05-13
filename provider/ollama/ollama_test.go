package ollama_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/inhuman/memorialiste/provider"
	"github.com/inhuman/memorialiste/provider/ollama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const goodResponse = `{"choices":[{"message":{"content":"ok"}}],"usage":{"total_tokens":1}}`

func TestNew_DefaultBaseURL(t *testing.T) {
	// We can't actually dial localhost:11434 in the test, but we can
	// verify the public default constant.
	assert.Equal(t, "http://localhost:11434", ollama.DefaultBaseURL)
}

func TestComplete_NoAuthorizationHeaderEverSent(t *testing.T) {
	var hadAuth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			hadAuth = true
		}
		_, _ = w.Write([]byte(goodResponse))
	}))
	defer srv.Close()

	p := ollama.New(ollama.Config{BaseURL: srv.URL, Model: "qwen3:8b"})
	_, _, err := p.Complete(t.Context(), []provider.Message{{Role: "user", Content: "x"}})
	require.NoError(t, err)
	assert.False(t, hadAuth, "Ollama adapter must never set Authorization header")
}

func TestComplete_ExplicitBaseURLPassesThrough(t *testing.T) {
	var requestPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		_, _ = w.Write([]byte(goodResponse))
	}))
	defer srv.Close()

	p := ollama.New(ollama.Config{BaseURL: srv.URL, Model: "m"})
	_, _, err := p.Complete(t.Context(), []provider.Message{{Role: "user", Content: "x"}})
	require.NoError(t, err)
	assert.Equal(t, "/v1/chat/completions", requestPath)
}
