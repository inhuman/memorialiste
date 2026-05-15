# Example 07 — Code Search Tool

Generate documentation where the LLM **actively pulls Go declarations**
during generation via the `search_code` function-calling tool. Useful when
the doc covers symbols defined far from the diff itself.

## What this example demonstrates

- **`--code-search`** — exposes the `search_code` tool to the LLM.
- **No `--ast-context`** — by design. With AST on, the model already sees
  the enclosing code of every changed line and rarely bothers with the tool.
  This example deliberately omits AST so the model must use the tool to
  get full bodies.
- **Custom system prompt (`prompt.md`)** that explicitly asks the model to
  call `search_code` before writing the final markdown.
- **`--code-search-max-turns 6`** — generous ceiling to allow 3-5 lookups.

## Verified models

Tool calling must follow the OpenAI Tools API correctly (proper
`tool_calls` in the response, not stringified JSON in `content`).
Verified working with local Ollama:

- `qwen3:14b` ✅
- `qwen3.6:35b` ✅
- `gpt-oss:120b` ✅

Models that fail to use the tool correctly (return JSON inside `content`
field instead of structured `tool_calls`):

- `qwen2.5-coder:7b` ❌
- `qwen3-coder:30b` ⚠ (works on small contexts, falls back to text on
  large contexts)

If the `code-search: turn=` log line never appears in your run, the model
isn't using the tool — switch to one of the verified ones above.

## Files

- `docstructure.yaml` — manifest scoped to the core pipeline packages
- `prompt.md` — system prompt that mandates tool calls
- `run.sh` — local dry-run with `--code-search` against Ollama

## Output

`docs/architecture.md` — architecture overview where each described
declaration's body was fetched by the model via `search_code`. You can
verify by inspecting the run logs for `code-search: turn=N name=...
args={"pattern":"...","path":"..."}` entries.

## Sample log lines

```
[1/1] calling LLM (qwen3:14b)
code-search: turn=1 name=search_code args={"pattern":"^Assemble$","path":"context/"}
code-search: turn=1 name=search_code args={"pattern":"^Generate$","path":"generate/"}
[1/1] LLM returned 3332 chars; tokens prompt=47445 completion=1450 total=48895
```

Two tool calls in one turn is normal — the model issued parallel calls.
The turn counter increments by 1 per turn, not per call, so this counts
as 1 of the configured 6 max turns.

## Tips

- **Want comprehensive context?** Add `--ast-context` — the model gets
  both AST around changes AND the tool for everything else. The model
  picks what it needs.
- **Tool not being called?** Make the system prompt more demanding
  ("MUST call search_code at least N times before writing markdown")
  and/or use a more tool-following model.
- **Hitting max-turns?** Either bump `--code-search-max-turns`, or
  tighten the prompt to ask for specific declarations rather than
  open-ended exploration.
