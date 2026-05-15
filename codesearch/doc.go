// Package codesearch performs AST-based identifier search across Go source
// files in a repository. It walks a scoped subtree, parses every .go file
// with the standard library's go/parser, and returns declarations whose
// identifier names match a caller-supplied regex.
//
// The package is the backend for the search_code LLM tool exposed by the
// generate package. It is stdlib-only, has no external dependencies, and
// is safe for concurrent calls — each Search receives its own FileSet.
package codesearch
