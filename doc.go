// Package memorialiste is a one-shot CLI tool and embeddable Go library
// for keeping documentation up-to-date with source code changes.
//
// Each run reads a .docstructure.yaml manifest, computes a filtered git diff
// since the last documentation update, calls an OpenAI-compatible LLM to
// update the docs, and optionally opens a merge/pull request on GitLab or
// GitHub.
//
// Usage as a library:
//
//	m, err := manifest.Parse("docs/.docstructure.yaml")
//	dc, err := context.Assemble(ctx, m.Docs[0], context.Options{
//	    RepoPath:    ".",
//	    TokenBudget: 12000,
//	    Summariser:  myProvider,
//	})
package memorialiste
