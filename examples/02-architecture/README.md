# Example 02 — Architecture Overview

Generate developer-facing documentation describing the **internal
architecture** — package layout, key abstractions, data flow.

## What this example demonstrates

- **Wide covers**: every internal package is in scope.
- **`--ast-context`** — model sees full function signatures via grep-ast's
  TreeContext renderer, not just `+`/`-` diff lines.
- **Larger token budget** — internal architecture docs are dense.
- **Built-in prompt encourages Mermaid** — expect at least one diagram in
  the output (flowchart for data flow, classDiagram for type relationships).

## Files

- `docstructure.yaml` — manifest covering all internal packages
- `run.sh` — local dry-run with `--ast-context`

## Output

A file at `docs/architecture.md` — ~300–500 lines with code samples,
Mermaid diagrams, package-by-package walkthrough.
