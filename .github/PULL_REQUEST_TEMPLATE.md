## What

<!-- One paragraph: what does this PR change? -->

## Why

<!-- The motivation. Link to the issue if one exists. -->

Closes #

## How

<!-- Brief implementation notes. What did you change architecturally? Any tricky bits the reviewer should pay attention to? -->

## Testing

- [ ] `go vet ./...` clean
- [ ] `go test ./... -count=1` all green
- [ ] New tests added for new behaviour (unit / integration)
- [ ] If image build is affected: `docker build .` succeeds locally
- [ ] If release-relevant: local Docker smoke against real Ollama passed (per Constitution Principle VIII)

## Backward compatibility

<!-- Does this change any default? Break an existing flag / env var / manifest field? If yes — explain the migration path. -->

## Checklist

- [ ] Commit message follows Conventional Commits (`feat(scope):`, `fix:`, `docs:`, etc.)
- [ ] Godoc added/updated for new exported identifiers
- [ ] README updated if behaviour visible to users changed
- [ ] No new core dependencies (or: amendment to Constitution included)
- [ ] No secrets / tokens in the diff
