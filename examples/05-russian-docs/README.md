# Example 05 — Russian-language Documentation

Pure language demonstration — every example above uses English by default;
this one shows the `--language russian` flag end-to-end.

## What this example demonstrates

- **`--language russian`** — the built-in prompt template substitutes
  `russian` into `Write in {language}.`
- Otherwise identical to Example 01 (user guide) — only the output
  language differs.

You can swap `russian` for `spanish`, `german`, `french`, `chinese`,
`japanese`, etc. Quality depends on the chosen model's training mix —
Qwen3 family handles Chinese natively, multilingual models like Mistral
or Llama-3 do most European languages well.

## Files

- `docstructure.yaml`
- `run.sh` — sets `--language russian`

## Output

`docs/ru/руководство.md` (Cyrillic path supported) — same structure as
Example 01 but in Russian.
