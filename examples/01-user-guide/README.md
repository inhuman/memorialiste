# Example 01 — User Guide

Generate an end-user guide explaining **how to USE memorialiste** (not how it
is built). The model focuses on CLI flags, env vars, GitLab/GitHub recipes,
local dry-run.

## What this example demonstrates

- **Minimal config**: only the CLI entry point (`cmd/`) and config layer
  (`cliconfig/`) are in `covers` — model doesn't see internal packages.
- **Built-in prompt** — no custom system prompt needed.
- **No AST context** — the audience doesn't care about Go internals.
- **English language** — default.

## Files

- `docstructure.yaml` — manifest scoped to user-facing surface
- `run.sh` — local dry-run against Ollama

## Output

A file at `docs/user/guide.md` covering installation, basic usage, CLI
reference, troubleshooting. Around 150–250 lines.
