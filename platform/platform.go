package platform

import (
	"context"
	"errors"
	"fmt"
)

// ChangeRequest is the payload for opening an MR (GitLab) or PR (GitHub).
type ChangeRequest struct {
	SourceBranch string
	TargetBranch string
	Title        string
	Body         string
}

// ChangeResult is the metadata returned after a successful MR/PR creation.
type ChangeResult struct {
	URL    string
	Number int
}

// Platform is the abstraction over GitLab and GitHub.
//
// Implementations must never include their authentication token in error
// messages or log output.
type Platform interface {
	Push(ctx context.Context, branch, headSHA string) error
	OpenChangeRequest(ctx context.Context, req ChangeRequest) (*ChangeResult, error)
}

// ErrSSHRemoteNotSupported is returned when the derived remote URL is not HTTPS.
var ErrSSHRemoteNotSupported = errors.New("platform: only HTTPS remotes are supported")

// ErrTokenRequired is returned when a non-dry-run call is attempted without a token.
var ErrTokenRequired = errors.New("platform: token is required for non-dry-run mode")

// HTTPError carries a non-2xx response from the platform's REST API.
type HTTPError struct {
	StatusCode int
	Body       string
}

// Error implements error.
func (e *HTTPError) Error() string {
	return fmt.Sprintf("platform: HTTP %d: %s", e.StatusCode, e.Body)
}
