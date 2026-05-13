// Package ollama is a thin Ollama-flavoured wrapper around provider/openai.
//
// Ollama exposes an OpenAI-compatible /v1/chat/completions endpoint, so the
// only differences from the generic adapter are the default base URL and the
// absence of bearer authentication.
package ollama
