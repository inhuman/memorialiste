# Contributing to memorialiste

Thanks for considering a contribution. This guide covers the dev loop,
project conventions, and how to ship a change.

## Quick start

```sh
git clone https://github.com/inhuman/memorialiste.git
cd memorialiste

# Run unit tests (no network required)
go test ./...

# Run go vet
go vet ./...

# Build the binary locally (no Docker)
go build -o /tmp/memorialiste ./cmd/memorialiste
/tmp/memorialiste --help

# Build the production Docker image
docker build -t memorialiste:dev --build-arg VERSION=dev .
docker run --rm memorialiste:dev --version
```

Go 1.26+ required. Dependencies are vendored — `vendor/` is committed.

## Reporting bugs and requesting features

Use the GitHub issue templates:

- [Bug report](.github/ISSUE_TEMPLATE/bug_report.md) — for things that
  don't work as documented.
- [Feature request](.github/ISSUE_TEMPLATE/feature_request.md) — for
  changes to behaviour, new flags, new providers, etc.

Before opening:

1. Search [existing issues](https://github.com/inhuman/memorialiste/issues)
   to avoid duplicates.
2. Try the latest tagged release (`docker pull idconstruct/memorialiste:latest`).
3. For bugs, capture the exact CLI invocation + the relevant log lines.

## Pull requests

1. Open an issue first for non-trivial changes — saves both of us a
   reroll.
2. Fork → branch off `main` → commit → push → open PR against `main`.
3. Use the PR template; describe **what** changed and **why**.
4. CI must pass: `go vet`, `go test`, the Docker image must build.
5. Maintainer review + squash merge.

## Project conventions

These are enforced in review:

- **Go style**: standard `gofmt`, Go 1.26 idioms (`cmp.Or`, `errors.AsType`,
  `slices`, `t.Context()` in tests). Run `go vet ./...` before pushing.
- **No comments on obvious code**. Godoc on every exported identifier.
- **No silent error discards** outside `defer x.Close()`.
- **No speculative abstractions**. Three similar lines beats a premature
  interface. An interface is introduced only after a third concrete
  consumer.
- **Vendored dependencies**: after any `go get`, run `go mod vendor` and
  commit the `vendor/` tree.
- **No new core dependencies** without an [amendment to the
  Constitution](.specify/memory/constitution.md). The core depends on
  Go stdlib + `go-git/v6` + `yaml.v3` + `kong`. Test code may use
  `testify`.

## Testing

- **Unit tests** — `_test.go` next to the package. Hermetic; no real HTTP,
  no real LLM. Use `internal/fake` for `Provider` / `Platform` /
  `Summariser` / `ToolingProvider` doubles.
- **Integration tests** — guarded by `//go:build integration`. Live LLM /
  network. Not run by default.
- Run all unit tests: `go test ./... -count=1`.

## Local smoke before tagging a release

Per the project Constitution (Principle VIII), before pushing a release
tag that triggers Docker Hub publication, the contributor MUST:

1. Build the image locally:
   ```sh
   docker build -t memorialiste:demo --build-arg VERSION=test-build .
   ```
2. Run an end-to-end doc-generation smoke against a real LLM endpoint
   (typically local Ollama) on a temporary clone of the target
   repository.
3. Verify the generated documentation file is produced and reasonable
   (not empty, watermark stamped correctly).

Only after the smoke passes should the release tag be pushed.

## Commit messages

Conventional Commits style:

- `feat(scope): ...` — new functionality
- `fix(scope): ...` — bug fix
- `docs(scope): ...` — README / spec / godoc changes
- `refactor(scope): ...` — code change without functional impact
- `test(scope): ...` — test-only changes
- `ci(scope): ...` — workflow / Dockerfile / build changes

Scope is the package or feature area (e.g. `feat(us-codesearch): ...`).

## Release process

Maintainers only:

1. `go vet ./...` clean, `go test ./...` all green.
2. Local Docker smoke per the section above.
3. `git tag vX.Y.Z -m "..."` (semver: MAJOR breaking, MINOR feature, PATCH fix).
4. `git push origin main && git push origin vX.Y.Z`.
5. GitHub Actions builds and pushes the image to Docker Hub with both
   `:vX.Y.Z` and `:latest` tags.
6. Verify the published image: `docker pull idconstruct/memorialiste:vX.Y.Z`.
7. Optionally create a GitHub release with `gh release create vX.Y.Z`.

## Specs

Larger features are designed via SpecKit before implementation. Specs
live under `specs/` (gitignored — kept local), but the resulting code
and tests are committed. If you're proposing a substantial feature,
open an issue describing the intent — the maintainer can run the
SpecKit flow with you.

## Code of conduct

Be kind, be specific, focus on the code not the person. Disagreement
about design is expected; rudeness is not.
