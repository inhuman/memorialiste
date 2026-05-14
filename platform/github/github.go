package github

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/inhuman/memorialiste/platform"
)

// DefaultBaseURL is the public GitHub API base URL.
const DefaultBaseURL = "https://api.github.com"

// Config configures a GitHub Platform adapter.
type Config struct {
	// BaseURL is the GitHub REST API base URL. Defaults to DefaultBaseURL.
	BaseURL string
	// Token is the OAuth-style access token.
	Token string
	// Repository is the "owner/repo" identifier.
	Repository string
	// RepoPath is the local repository path used by Push.
	RepoPath string
	// Timeout caps HTTP and git push operations. Defaults to 60s when zero.
	Timeout time.Duration
	// HTTPClient overrides the default HTTP client used for API calls.
	HTTPClient *http.Client
}

// New returns a GitHub Platform adapter configured by cfg.
func New(cfg Config) platform.Platform {
	return &client{
		baseURL:    strings.TrimRight(cmp.Or(cfg.BaseURL, DefaultBaseURL), "/"),
		token:      cfg.Token,
		repository: cfg.Repository,
		repoPath:   cfg.RepoPath,
		timeout:    cmp.Or(cfg.Timeout, 60*time.Second),
		httpClient: cmp.Or(cfg.HTTPClient, http.DefaultClient),
	}
}

type client struct {
	baseURL    string
	token      string
	repository string
	repoPath   string
	timeout    time.Duration
	httpClient *http.Client
}

// Push pushes the named branch from the local repository to the canonical
// HTTPS git URL derived from the configured BaseURL and Repository.
func (c *client) Push(ctx context.Context, branch, _ string) error {
	if c.token == "" {
		return platform.ErrTokenRequired
	}
	remoteURL, err := c.remoteURL()
	if err != nil {
		return err
	}
	return platform.HTTPSPush(ctx, c.repoPath, remoteURL, branch, c.token, c.timeout)
}

// OpenChangeRequest creates a GitHub pull request and returns its HTML URL
// and number.
func (c *client) OpenChangeRequest(ctx context.Context, req platform.ChangeRequest) (*platform.ChangeResult, error) {
	if c.token == "" {
		return nil, platform.ErrTokenRequired
	}
	endpoint := fmt.Sprintf("%s/repos/%s/pulls", c.baseURL, c.repository)
	payload := map[string]string{
		"head":  req.SourceBranch,
		"base":  req.TargetBranch,
		"title": req.Title,
		"body":  req.Body,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("github: marshal PR payload: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("github: build PR request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("github: create PR: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &platform.HTTPError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var out struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("github: decode PR response: %w", err)
	}
	return &platform.ChangeResult{URL: out.HTMLURL, Number: out.Number}, nil
}

// remoteURL derives the canonical HTTPS git URL from BaseURL and Repository.
// For api.github.com the host becomes github.com; for Enterprise hosts of the
// shape "api.<host>" the "api." prefix is stripped; for Enterprise hosts that
// expose the API under "/api/v3" the suffix is stripped from the path.
func (c *client) remoteURL() (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("github: parse base URL: %w", err)
	}
	host := u.Host
	if host == "" {
		return "", fmt.Errorf("github: base URL missing host")
	}
	if strings.HasPrefix(host, "api.") {
		host = strings.TrimPrefix(host, "api.")
	}
	return fmt.Sprintf("https://%s/%s.git", host, c.repository), nil
}
