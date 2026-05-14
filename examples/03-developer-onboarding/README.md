# Example 03 — Developer Onboarding Guide

Generate a guide for **new contributors** — how to set up dev env, where
the code lives, how to run tests, conventions (Constitution principles).

## What this example demonstrates

- **Custom system prompt** via `--system-prompt @prompt.md` — overrides the
  built-in writer-role prompt with a contributor-onboarding prompt.
- **Wide covers** plus the constitution copy in `.specify/` (well, no — that
  one's gitignored; we instead point at top-level `cmd/` + core packages).
- **`--ast-context`** for code-structure awareness.

## Files

- `docstructure.yaml`
- `prompt.md` — custom system prompt focused on onboarding
- `run.sh` — wires `--system-prompt @./prompt.md`

## Output

`docs/contributing.md` — a contributor-focused guide: clone, build, test,
make a PR, project conventions.
