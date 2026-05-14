// Package platform defines the abstraction over Git hosting platforms
// (GitLab and GitHub).
//
// A Platform pushes a local branch over HTTPS using an OAuth-style token
// and opens a merge or pull request via the platform's REST API.
//
// Adapter packages: platform/gitlab and platform/github.
//
// Authentication tokens MUST NOT appear in any error message or log line
// produced by this package or its adapters.
package platform
