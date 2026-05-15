# memorialiste

> La mémorialiste visits your repository, reads what changed since its last visit, writes the missing chapters of your project's story, and leaves a merge request behind.

A one-shot CLI tool that keeps documentation up-to-date with source code changes.
Each run computes a `git diff` since the last documentation update, calls an
OpenAI-compatible LLM to rewrite the affected docs, and opens a Merge/Pull Request.

## Installation

```sh
docker pull idconstruct/memorialiste:latest
```

Pin a specific version for reproducibility:

```sh
docker pull idconstruct/memorialiste:v0.2.1
```

## Usage

### GitLab CI

```yaml
update-docs:
  image: idconstruct/memorialiste:latest
  variables:
    MEMORIALISTE_AST_CONTEXT: "true"
  script:
    - memorialiste
      --provider-url "$OLLAMA_URL"
      --model "qwen3-coder:30b"
      --platform gitlab
      --platform-token "$GITLAB_TOKEN"
      --project-id "$CI_PROJECT_ID"
      --dry-run=false
  rules:
    - if: $CI_COMMIT_BRANCH == "main"
```

### GitHub Actions

```yaml
- name: Update docs
  run: |
    docker run --rm --network=host \
      -v ${{ github.workspace }}:/repo \
      -e MEMORIALISTE_PLATFORM_TOKEN=${{ secrets.GITHUB_TOKEN }} \
      idconstruct/memorialiste:latest \
      --repo /repo \
      --provider-url "$OLLAMA_URL" \
      --model qwen3-coder:30b \
      --platform github \
      --project-id "${{ github.repository }}" \
      --dry-run=false \
      --ast-context
```

### Local dry-run

```sh
docker run --rm --network=host --user $(id -u):$(id -g) \
  -v $(pwd):/repo \
  idconstruct/memorialiste:latest \
  --repo /repo \
  --provider-url http://localhost:11434 \
  --model qwen3-coder:30b \
  --ast-context
```

## Using Claude, Gemini, GPT-4 and other models

memorialiste talks to LLMs **exclusively via the OpenAI-compatible
`/v1/chat/completions` API**. There is no native Anthropic / Google / OpenAI
SDK. To use any non-Ollama model, run an **OpenAI-compatible proxy** that
translates requests to the target provider's native API. memorialiste itself
needs zero changes — you point `--provider-url` at the proxy and adjust
`--model`.

### Self-hosted: LiteLLM

[LiteLLM](https://github.com/BerriAI/litellm) supports ~100 models (Claude,
Gemini, Bedrock, Vertex AI, etc.) and runs as a Docker sidecar.

```yaml
# docker-compose.yml
services:
  litellm:
    image: ghcr.io/berriai/litellm:main-latest
    ports: ["4000:4000"]
    environment:
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
      OPENAI_API_KEY:    ${OPENAI_API_KEY}
```

```sh
memorialiste --provider-url http://litellm:4000 --model claude-3-5-sonnet-20241022
```

### Self-hosted: one-api

[one-api](https://github.com/songquanpeng/one-api) — aggregator with a web UI,
same OpenAI-compat surface. Point `--provider-url` at its base URL.

### Cloud: OpenRouter

[OpenRouter](https://openrouter.ai) routes to Claude, GPT-4, Gemini and many
others via a single OpenAI-compat endpoint.

```sh
memorialiste \
  --provider-url https://openrouter.ai/api/v1 \
  --api-key "$OPENROUTER_API_KEY" \
  --model anthropic/claude-3.5-sonnet
```

The `--api-key` value is sent as `Authorization: Bearer <key>` to the
provider. Combine with `--model-params` to tune temperature, top_p, etc.

## CLI Flags & Environment Variables

All flags can be set via environment variables (uppercase snake_case with
`MEMORIALISTE_` prefix). Flags take precedence over env vars.

| Flag | Env var | Default | Description |
|------|---------|---------|-------------|
| `--provider-url` | `MEMORIALISTE_PROVIDER_URL` | `http://localhost:11434` | OpenAI-compatible base URL |
| `--model` | `MEMORIALISTE_MODEL` | `qwen3-coder:30b` | Model tag |
| `--model-params` | `MEMORIALISTE_MODEL_PARAMS` | `""` | Extra model params JSON (e.g. `{"temperature":0.2}`) |
| `--system-prompt` | `MEMORIALISTE_SYSTEM_PROMPT` | built-in | System prompt literal OR `@path/to/file` |
| `--prompt` | `MEMORIALISTE_PROMPT` | `""` | Additional user prompt appended after diff |
| `--language` | `MEMORIALISTE_LANGUAGE` | `english` | Output language; substituted into `{language}` placeholder |
| `--api-key` | `MEMORIALISTE_API_KEY` | `""` | Bearer token for the LLM provider |
| `--doc-structure` | `MEMORIALISTE_DOC_STRUCTURE` | `docs/.docstructure.yaml` | Path to the doc structure manifest |
| `--repo` | `MEMORIALISTE_REPO` | `.` | Local git repository root |
| `--token-budget` | `MEMORIALISTE_TOKEN_BUDGET` | `12000` | Max diff tokens before summarisation kicks in |
| `--dry-run` | `MEMORIALISTE_DRY_RUN` | `true` | Write files locally; skip branch+commit+platform |
| `--branch-prefix` | `MEMORIALISTE_BRANCH_PREFIX` | `docs/memorialiste-` | Branch name prefix |
| `--ast-context` | `MEMORIALISTE_AST_CONTEXT` | `false` | Enable AST-enriched diff context via grep-ast |
| `--code-search` | `MEMORIALISTE_CODE_SEARCH` | `false` | Expose the AST `search_code` tool to the LLM (function calling) |
| `--code-search-max-turns` | `MEMORIALISTE_CODE_SEARCH_MAX_TURNS` | `10` | Max tool-call turns before aborting |
| `--repo-meta` | `MEMORIALISTE_REPO_META` | `basic` | Repo metadata level: `basic` or `extended` |
| `--platform` | `MEMORIALISTE_PLATFORM` | `gitlab` | `gitlab` or `github` |
| `--platform-url` | `MEMORIALISTE_PLATFORM_URL` | platform default | Base URL for self-hosted instances |
| `--platform-token` | `MEMORIALISTE_PLATFORM_TOKEN` | _required (non-dry-run)_ | Platform access token |
| `--project-id` | `MEMORIALISTE_PROJECT_ID` | _required (non-dry-run)_ | GitLab project ID or `owner/repo` |
| `--base-branch` | `MEMORIALISTE_BASE_BRANCH` | `main` | Target branch for the opened MR/PR |
| `--version` | — | — | Print version and exit |
| `--help` | — | — | Show grouped help |

## Watermark Format

Every generated doc file carries YAML frontmatter:

```markdown
---
generated_at: abc1234def5
---

# Your Doc Title
...
```

The tool reads `generated_at` to compute the diff since the last run.
A file without frontmatter is treated as never generated (full-repo diff
scoped to the entry's `covers` paths).

## Doc Structure Manifest

`docs/.docstructure.yaml` declares which docs exist and what source paths
each one covers:

```yaml
docs:
  - path: docs/architecture.md
    audience: developers
    covers:
      - context/
      - generate/
      - output/
      - platform/
    description: >
      Internal architecture: package layout, key abstractions, data flow.

  - path: docs/user/guide.md
    audience: end users
    covers:
      - cmd/
      - cliconfig/
    description: >
      User-facing usage guide.
```

Each entry runs independently and only sees the diff scoped to its `covers`.

## Repository Metadata

The LLM receives a compact metadata block prepended to its prompt so it can
write accurate version numbers:

```
=== Repository metadata ===
Latest tag: v0.2.1
HEAD: 8c9e7d2...
Short SHA: 8c9e7d2
=== End metadata ===
```

`--repo-meta=extended` adds remote URL (token-redacted), branch, last 5 tags
with dates — useful for CHANGELOG / release-notes documents.

## AST-Enriched Context

`--ast-context` runs every changed file through grep-ast's TreeContext
renderer, so the model sees enclosing function signatures and surrounding
code structure instead of raw `+`/`-` lines. Significantly improves quality
for code-heavy docs.

## AST Code Search

`--code-search` exposes a `search_code` function-calling tool to the LLM.
Mid-generation the model may ask for any Go declaration in the repo by
regex name match; the tool returns the matched function, method, type,
const, or var bodies with file paths and line ranges. Useful when the
diff alone lacks context (e.g. a doc covering one package references
symbols defined in another).

Bounded by `--code-search-max-turns` (default 10) and a per-file 5s parse
timeout. Provider must implement OpenAI-style function calling and emit
proper `tool_calls` (not stringified JSON in `content`). Verified working
on local Ollama: `qwen3:14b`, `qwen3.6:35b`, `gpt-oss:120b`. Models that
return `finish_reason: stop` with a JSON blob in content (e.g.
`qwen2.5-coder:7b`, sometimes `qwen3-coder:30b` with large contexts) do
not follow the API correctly — switch model if you see no `code-search: turn=`
log entries. If the provider rejects a tools-shaped request entirely,
memorialiste fails fast with an actionable error suggesting `--code-search=false`.

**Tip — when to combine with `--ast-context`**: AST context already
embeds the enclosing function/method around every changed line, so
tool-capable models often skip `search_code` entirely when AST is on.
Use `--code-search` ALONE when you want the model to pull in
declarations referenced by the diff but defined far away from it; use
both flags together for the most thorough context (the model picks
what it needs).

## Architecture Diagrams

The built-in system prompt encourages the LLM to emit Mermaid diagrams
(```` ```mermaid ```` fenced blocks) when the diff touches architecture, data
flow, or component relationships. GitLab and GitHub render Mermaid natively
in Markdown previews. No rendering toolchain required.

## Runtime Dependencies

The Docker image bundles:

| Tool | Version | Purpose |
|------|---------|---------|
| `grep-ast` | 0.5.0 | AST-enriched diff context (`--ast-context`) |
| `tree-sitter` | 0.20.4 | Required by grep-ast |
| `tree-sitter-languages` | 1.10.2 | Language grammars for grep-ast |

These are only invoked when `--ast-context` is enabled.

## Examples

See [`examples/`](examples/) for ready-to-run scenarios:

| Scenario | What it shows |
|----------|---------------|
| [`01-user-guide`](examples/01-user-guide/) | Plain end-user guide; built-in prompt; minimal config |
| [`02-architecture`](examples/02-architecture/) | Developer-facing architecture overview with AST + Mermaid |
| [`03-developer-onboarding`](examples/03-developer-onboarding/) | Custom system prompt for contributor onboarding |
| [`04-ai-readable`](examples/04-ai-readable/) | Dense LLM-readable project context (think `CLAUDE.md`) |
| [`05-russian-docs`](examples/05-russian-docs/) | `--language russian` (works for any language) |
| [`06-changelog`](examples/06-changelog/) | CHANGELOG via `--repo-meta=extended` |
| [`ci-gitlab`](examples/ci-gitlab/) | Drop-in `.gitlab-ci.yml` |
| [`ci-github`](examples/ci-github/) | Drop-in GitHub Actions workflow |

Every doc-scenario folder contains an executable `run.sh` that you can
invoke locally against a running Ollama.

## Library Usage

memorialiste is also a Go library — use `manifest`, `context`, `generate`,
`output`, and `platform` packages directly. See package godoc.

```go
import (
    "context"
    "github.com/inhuman/memorialiste/manifest"
    mctx "github.com/inhuman/memorialiste/context"
)

m, _ := manifest.Parse("docs/.docstructure.yaml")
dc, _ := mctx.Assemble(context.Background(), m.Docs[0], mctx.Options{
    RepoPath:    ".",
    ASTContext:  true,
    TokenBudget: 12000,
})
fmt.Println(dc.Diff)
```
