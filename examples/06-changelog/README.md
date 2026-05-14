# Example 06 — CHANGELOG from tag history

Generate a `CHANGELOG.md` using `--repo-meta=extended` so the model sees
the last 5 git tags with their dates and can write release-by-release
entries.

## What this example demonstrates

- **`--repo-meta=extended`** — model receives a metadata block listing
  recent tags with commit dates.
- **Custom system prompt** explicitly structured around the
  Keep-a-Changelog format.
- **No `covers`-based diff** strategy is special here — the source paths
  are all of `cmd/` + `cliconfig/` + core packages because the changelog
  describes user-visible changes.

## Files

- `docstructure.yaml`
- `prompt.md` — Keep-a-Changelog-style prompt
- `run.sh`

## Output

`CHANGELOG.md` at repo root — sections per recent tag (Added / Changed /
Fixed / Deprecated), with dates pulled from real git tags.

## Caveat

The model only sees tag NAMES + dates via the metadata block, not the
per-tag commit messages. For richer changelogs you can pipe `git log
v0.2.0..v0.2.1` output into `--prompt` manually, or wait for a future
US that adds per-tag commit summaries to extended meta.
