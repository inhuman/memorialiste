---
name: Bug report
about: Something doesn't work as documented
title: "bug: "
labels: bug
---

## Summary

<!-- One sentence: what's broken? -->

## Environment

- memorialiste version: <!-- run `memorialiste --version`, or "v0.4.1" / "main@abc1234" -->
- How you run it: <!-- Docker / built locally / library import -->
- Docker image tag: <!-- e.g. idconstruct/memorialiste:v0.4.1 -->
- Host OS / arch: <!-- linux/amd64, macOS arm64, etc. -->
- Ollama / LLM provider: <!-- "Ollama qwen3-coder:30b on localhost", "OpenRouter / claude-3-5-sonnet", etc. -->

## Reproduction

Minimal CLI invocation that reproduces the issue:

```sh
docker run --rm --network=host \
  -v "$(pwd)":/repo \
  idconstruct/memorialiste:v0.4.1 \
  --repo /repo \
  --doc-structure /repo/docs/.docstructure.yaml \
  --provider-url http://localhost:11434 \
  --model qwen3-coder:30b \
  --dry-run
```

Minimal `.docstructure.yaml` (strip secrets / large covers):

```yaml
docs:
  - path: docs/x.md
    audience: developers
    covers: [.]
    description: ...
```

## Expected behaviour

<!-- What you thought should happen. -->

## Actual behaviour

<!-- What happened instead. Paste relevant log lines (NOT full traces unless they're short). -->

```
<paste log output here>
```

## Additional context

<!-- Watermark file contents (if sidecar mode), manifest size, repo size, anything else relevant. -->
