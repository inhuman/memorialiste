# Example — GitHub Actions integration

Workflow that runs memorialiste on every push to `main` (or on schedule),
pushes the branch, opens a Pull Request.

## What's needed in GitHub repo settings

| Secret | Where to set | Value |
|--------|--------------|-------|
| `MEMORIALISTE_TOKEN` | Settings → Secrets and variables → Actions | Fine-grained PAT or classic token with `repo` scope (the default `GITHUB_TOKEN` lacks PR-creation permissions for actions/checkout-style runs without `pull-requests: write` in workflow `permissions:`) |
| `OLLAMA_URL` | Settings → Variables | Public-accessible Ollama URL OR set up via tailscale/cloudflare tunnel |

For self-hosted runners with local Ollama: set `OLLAMA_URL=http://host.docker.internal:11434`.

## Files

- `docs.yml` — copy to `.github/workflows/docs.yml` in your repo
