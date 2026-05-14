// Package gitlab implements platform.Platform against the GitLab v4 REST API.
//
// Authentication uses the PRIVATE-TOKEN header for API calls and HTTP basic
// auth with username "oauth2" for git push over HTTPS. Tokens are never
// included in error messages or log output.
package gitlab
