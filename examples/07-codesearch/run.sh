#!/usr/bin/env bash
# Example 07 — Code Search Tool
#
# Generates docs/architecture.md where the LLM uses the `search_code` tool
# to fetch declaration bodies. Picks a model known to follow the OpenAI
# Tools API correctly (set MODEL=... to override).
#
# Watch for `code-search: turn=N name=...` lines in the output — that's
# the model actually calling the tool.

set -euo pipefail

REPO="${REPO:-$(git rev-parse --show-toplevel)}"
IMAGE="${IMAGE:-idconstruct/memorialiste:latest}"
MODEL="${MODEL:-gpt-oss:120b}"
PROVIDER_URL="${PROVIDER_URL:-http://localhost:11434}"

docker run --rm --network=host --user "$(id -u):$(id -g)" \
  -v "$REPO":/repo \
  -v "$(pwd)/docstructure.yaml":/manifest.yaml:ro \
  -v "$(pwd)/prompt.md":/prompt.md:ro \
  "$IMAGE" \
  --repo /repo \
  --doc-structure /manifest.yaml \
  --provider-url "$PROVIDER_URL" \
  --model "$MODEL" \
  --code-search \
  --code-search-max-turns 6 \
  --token-budget 200000 \
  --system-prompt "@/prompt.md" \
  --language english
