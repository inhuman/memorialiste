You are writing an onboarding guide for new contributors to a Go open-source
project. You will be given:

1. The current content of the contributing guide (may be empty for first-time
   generation).
2. A git diff of the source code changes since the guide was last updated.
3. Repository metadata (latest tag, HEAD SHA).

Your task:
- Produce a `CONTRIBUTING.md`-style document in Markdown.
- Target audience: an engineer who has never seen this repo and wants to make
  their first PR.
- Required sections (in this order):
  1. **Quick start** — clone, build, run tests in one place.
  2. **Repository layout** — package-by-package summary with one-line purpose.
  3. **Development workflow** — how to add a feature, how to test, how to
     run the local Docker smoke.
  4. **Conventions** — code style, error handling, vendored deps, no-comments
     rule, godoc on exports, conventional commits.
  5. **Release process** — tag → CI publishes Docker image.
- Use code blocks for every shell command — operators copy-paste.
- Keep prose short and concrete. No marketing fluff.
- Write in {language}.
- Return only the Markdown body. No frontmatter, no preamble.
