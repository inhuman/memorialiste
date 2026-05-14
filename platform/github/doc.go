// Package github implements platform.Platform against the GitHub v3 REST API.
//
// Authentication uses a Bearer token for API calls and HTTP basic auth with
// username "oauth2" for git push over HTTPS. Tokens are never included in
// error messages or log output.
package github
