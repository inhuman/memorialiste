#!/usr/bin/env bash
# Example 06 — CHANGELOG from recent tags
#
# Uses --repo-meta=extended so the model sees the last 5 tags with dates.

set -euo pipefail

REPO="${REPO:-$(git rev-parse --show-toplevel)}"
IMAGE="${IMAGE:-idconstruct/memorialiste:latest}"
MODEL="${MODEL:-qwen3-coder:30b}"
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
  --ast-context \
  --repo-meta extended \
  --token-budget 200000 \
  --system-prompt "@/prompt.md" \
  --language english
