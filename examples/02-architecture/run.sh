#!/usr/bin/env bash
# Example 02 — Architecture Overview
#
# Generates docs/architecture.md covering every internal package, with
# AST-enriched context and Mermaid diagrams.

set -euo pipefail

REPO="${REPO:-$(git rev-parse --show-toplevel)}"
IMAGE="${IMAGE:-idconstruct/memorialiste:latest}"
MODEL="${MODEL:-qwen3-coder:30b}"
PROVIDER_URL="${PROVIDER_URL:-http://localhost:11434}"

docker run --rm --network=host --user "$(id -u):$(id -g)" \
  -v "$REPO":/repo \
  -v "$(pwd)/docstructure.yaml":/manifest.yaml:ro \
  "$IMAGE" \
  --repo /repo \
  --doc-structure /manifest.yaml \
  --provider-url "$PROVIDER_URL" \
  --model "$MODEL" \
  --ast-context \
  --token-budget 200000 \
  --language english
