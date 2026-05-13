// Package provider declares the LLM provider abstraction.
//
// The Provider interface is the only contract the core depends on; concrete
// HTTP adapters live in provider/openai and provider/ollama and are wired
// in by cmd/memorialiste at construction time.
package provider
