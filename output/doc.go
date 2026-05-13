// Package output writes generated documentation to disk with bumped YAML
// frontmatter watermarks and — in non-dry-run mode — creates a fresh local
// git branch and a single commit containing only the updated doc files.
//
// Apply is the single entry point. Pushing the branch and opening a
// merge/pull request is the responsibility of the platform package.
package output
