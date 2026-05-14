# Example — GitLab CI integration

Drop-in `.gitlab-ci.yml` that runs memorialiste on every push to `main`,
pushes the resulting branch to GitLab, and opens a Merge Request.

## What's needed in GitLab project settings

| Variable | Where to set | Value |
|----------|--------------|-------|
| `GITLAB_TOKEN` | CI/CD → Variables (Protected, Masked) | Personal access token with `api` + `write_repository` scopes |
| `OLLAMA_URL` | CI/CD → Variables | Internal Ollama URL reachable by the runner |

## Files

- `gitlab-ci.yml` — copy to `.gitlab-ci.yml` in your repo root
