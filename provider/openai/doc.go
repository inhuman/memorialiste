// Package openai is a generic OpenAI-compatible chat completions adapter.
//
// It speaks the OpenAI /v1/chat/completions HTTP shape and works against
// any endpoint that implements that contract (OpenAI proper, Ollama via
// its OpenAI-compat surface, LiteLLM, OpenRouter, one-api, vLLM).
package openai
