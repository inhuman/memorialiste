# Example 04 — AI-Readable Context

Generate a `docs/ai/context.md` file **optimized for consumption by LLMs**,
not humans. Useful when you want to drop a single file into an AI chat as
"here's what this project is" or for tools like Cursor / Claude that read a
project context file.

## What this example demonstrates

- **Custom system prompt** that explicitly targets LLM consumers — terse,
  structured, no decorative markdown.
- **Mermaid disabled** in this prompt (it's noise for LLMs).
- **Extended repo meta** — model sees recent tags too.
- **High `token-budget`** — single self-contained context dump.

## Files

- `docstructure.yaml`
- `prompt.md` — LLM-targeted system prompt
- `run.sh`

## Output

`docs/ai/context.md` — a dense, structured project context file. No
narrative prose, no decorative headers. Pure facts: purpose, packages,
public types, key invariants, CLI flags.

Drop it into `CLAUDE.md` / `.cursorrules` / system prompt of any assistant.
