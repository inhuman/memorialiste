# memorialiste

> La mémorialiste visits your repository, reads what changed since its last visit, writes the missing chapters of your project's story, and leaves a merge request behind.

A one-shot CLI tool that keeps documentation up-to-date with source code changes.
Each run computes a `git diff` since the last documentation update, calls an
OpenAI-compatible LLM to rewrite the affected docs, and opens a Merge/Pull Request.

## Installation

```sh
docker pull <your-dockerhub-username>/memorialiste:latest
```

## Usage

### GitLab CI

```yaml
update-docs:
  image: <your-dockerhub-username>/memorialiste:latest
  script:
    - memorialiste
      --provider-url "$OLLAMA_URL"
      --model "qwen3:8b"
      --platform gitlab
      --platform-token "$GITLAB_TOKEN"
      --project-id "$CI_PROJECT_ID"
  rules:
    - if: $CI_COMMIT_BRANCH == "main"
```

### Local dry-run

```sh
docker run --rm -v $(pwd):/repo \
  <your-dockerhub-username>/memorialiste:latest \
  --dry-run --repo /repo --provider-url http://host.docker.internal:11434
```

## CLI Flags & Environment Variables

All flags can be set via environment variables (uppercase snake_case with
`MEMORIALISTE_` prefix). Flags take precedence over env vars.

| Flag | Env var | Default | Description |
|------|---------|---------|-------------|
| `--provider-url` | `MEMORIALISTE_PROVIDER_URL` | `http://localhost:11434` | OpenAI-compatible base URL |
| `--model` | `MEMORIALISTE_MODEL` | `qwen3:8b` | Model tag |
| `--model-params` | `MEMORIALISTE_MODEL_PARAMS` | `""` | Extra model params JSON |
| `--system-prompt` | `MEMORIALISTE_SYSTEM_PROMPT` | built-in | System prompt or `@path/to/file` |
| `--prompt` | `MEMORIALISTE_PROMPT` | `""` | Additional user prompt |
| `--language` | `MEMORIALISTE_LANGUAGE` | `english` | Output language |
| `--doc-structure` | `MEMORIALISTE_DOC_STRUCTURE` | `docs/.docstructure.yaml` | Manifest path |
| `--repo` | `MEMORIALISTE_REPO` | `.` | Local git repository root |
| `--platform` | `MEMORIALISTE_PLATFORM` | `gitlab` | `gitlab` or `github` |
| `--platform-url` | `MEMORIALISTE_PLATFORM_URL` | platform default | Base URL for self-hosted |
| `--platform-token` | `MEMORIALISTE_PLATFORM_TOKEN` | _required_ | Personal access token |
| `--project-id` | `MEMORIALISTE_PROJECT_ID` | _required_ | GitLab project ID or `owner/repo` |
| `--api-key` | `MEMORIALISTE_API_KEY` | `""` | Bearer token for LLM provider |
| `--token-budget` | `MEMORIALISTE_TOKEN_BUDGET` | `12000` | Max diff tokens before summarisation |
| `--dry-run` | `MEMORIALISTE_DRY_RUN` | `false` | Write files locally, skip MR |
| `--branch-prefix` | `MEMORIALISTE_BRANCH_PREFIX` | `docs/memorialiste-` | Branch name prefix |
| `--ast-context` | `MEMORIALISTE_AST_CONTEXT` | `false` | Enable AST-enriched diff context |

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
A file without frontmatter is treated as never generated (full-repo diff).

## Runtime Dependencies

The Docker image includes:

| Tool | Version | Purpose |
|------|---------|---------|
| `grep-ast` | 0.8.4 | AST-enriched diff context (`--ast-context`) |
| `tree-sitter-language-pack` | 0.3.4 | Language grammars for grep-ast |

These are only used when `--ast-context` is enabled.
