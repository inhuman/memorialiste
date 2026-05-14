package platform

import (
	"cmp"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-git/go-git/v6"
	gitcfg "github.com/go-git/go-git/v6/config"
	gitclient "github.com/go-git/go-git/v6/plumbing/client"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
)

// HTTPSPush pushes a single branch from a local repository to a remote over HTTPS.
//
// The token is sent as HTTP Basic auth with username "oauth2" and the token as
// the password, which is the canonical scheme accepted by both GitLab and
// GitHub for OAuth-style access tokens.
//
// remoteURL must use the https:// scheme. SSH URLs are rejected with
// ErrSSHRemoteNotSupported. The token never appears in the returned error.
func HTTPSPush(ctx context.Context, repoPath, remoteURL, branch, token string, timeout time.Duration) error {
	if !strings.HasPrefix(remoteURL, "https://") {
		return ErrSSHRemoteNotSupported
	}
	return doPush(ctx, repoPath, remoteURL, branch, token, timeout)
}

// doPush is the inner push implementation, exposed for tests so they can use
// non-HTTPS schemes (e.g. file://) against an in-memory bare repo.
func doPush(ctx context.Context, repoPath, remoteURL, branch, token string, timeout time.Duration) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("platform: open repo %q: %w", repoPath, err)
	}

	remote, err := repo.CreateRemoteAnonymous(&gitcfg.RemoteConfig{
		Name: "anonymous",
		URLs: []string{remoteURL},
	})
	if err != nil {
		return fmt.Errorf("platform: create anonymous remote: %w", err)
	}

	pushCtx, cancel := context.WithTimeout(ctx, cmp.Or(timeout, 60*time.Second))
	defer cancel()

	opts := &git.PushOptions{
		RemoteName: "anonymous",
		RefSpecs:   []gitcfg.RefSpec{gitcfg.RefSpec("refs/heads/" + branch + ":refs/heads/" + branch)},
	}
	if token != "" {
		opts.ClientOptions = []gitclient.Option{
			gitclient.WithHTTPAuth(&githttp.BasicAuth{Username: "oauth2", Password: token}),
		}
	}

	if err := remote.PushContext(pushCtx, opts); err != nil {
		return fmt.Errorf("platform: push branch %q: %w", branch, redactToken(err, token))
	}
	return nil
}

// redactToken scrubs the token from an error's message so it never leaks into
// logs. The wrapped error is replaced with its redacted string form.
func redactToken(err error, token string) error {
	if err == nil || token == "" {
		return err
	}
	msg := err.Error()
	if !strings.Contains(msg, token) {
		return err
	}
	return fmt.Errorf("%s", strings.ReplaceAll(msg, token, "[REDACTED]"))
}
