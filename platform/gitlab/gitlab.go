package gitlab

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

// DefaultBaseURL is the public GitLab instance.
const DefaultBaseURL = "https://gitlab.com"

// Config configures a GitLab Platform adapter.
type Config struct {
	// BaseURL is the GitLab API base URL. Defaults to DefaultBaseURL.
	BaseURL string
	// Token is the PRIVATE-TOKEN value (personal access token).
	Token string
	// ProjectID is either a numeric project ID or the URL-style
	// "group/project" path.
	ProjectID string
	// RepoPath is the local repository path used by Push.
	RepoPath string
	// Timeout caps HTTP and git push operations. Defaults to 60s when zero.
	Timeout time.Duration
	// HTTPClient overrides the default HTTP client used for API calls.
	HTTPClient *http.Client
}

// New returns a GitLab Platform adapter configured by cfg.
func New(cfg Config) platform.Platform {
	return &client{
		baseURL:    strings.TrimRight(cmp.Or(cfg.BaseURL, DefaultBaseURL), "/"),
		token:      cfg.Token,
		projectID:  cfg.ProjectID,
		repoPath:   cfg.RepoPath,
		timeout:    cmp.Or(cfg.Timeout, 60*time.Second),
		httpClient: cmp.Or(cfg.HTTPClient, http.DefaultClient),
	}
}

type client struct {
	baseURL    string
	token      string
	projectID  string
	repoPath   string
	timeout    time.Duration
	httpClient *http.Client
}

// Push resolves the project's HTTPS remote URL via the GitLab v4 API and
// pushes the named branch from the local repository to that remote.
func (c *client) Push(ctx context.Context, branch, _ string) error {
	if c.token == "" {
		return platform.ErrTokenRequired
	}
	remoteURL, err := c.resolveRemoteURL(ctx)
	if err != nil {
		return err
	}
	return platform.HTTPSPush(ctx, c.repoPath, remoteURL, branch, c.token, c.timeout)
}

// OpenChangeRequest creates a GitLab merge request and returns its web URL
// and internal ID.
func (c *client) OpenChangeRequest(ctx context.Context, req platform.ChangeRequest) (*platform.ChangeResult, error) {
	if c.token == "" {
		return nil, platform.ErrTokenRequired
	}
	endpoint := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests", c.baseURL, encodeProjectID(c.projectID))
	payload := map[string]string{
		"source_branch": req.SourceBranch,
		"target_branch": req.TargetBranch,
		"title":         req.Title,
		"description":   req.Body,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("gitlab: marshal MR payload: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gitlab: build MR request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("PRIVATE-TOKEN", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gitlab: create MR: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &platform.HTTPError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var out struct {
		WebURL string `json:"web_url"`
		IID    int    `json:"iid"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("gitlab: decode MR response: %w", err)
	}
	return &platform.ChangeResult{URL: out.WebURL, Number: out.IID}, nil
}

// resolveRemoteURL fetches the project's path_with_namespace via the API and
// builds the canonical HTTPS git URL.
func (c *client) resolveRemoteURL(ctx context.Context) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v4/projects/%s", c.baseURL, encodeProjectID(c.projectID))

	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("gitlab: build project request: %w", err)
	}
	httpReq.Header.Set("PRIVATE-TOKEN", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("gitlab: resolve project: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &platform.HTTPError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var out struct {
		PathWithNamespace string `json:"path_with_namespace"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("gitlab: decode project response: %w", err)
	}
	if out.PathWithNamespace == "" {
		return "", fmt.Errorf("gitlab: project response missing path_with_namespace")
	}

	host, err := apiHost(c.baseURL)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s/%s.git", host, out.PathWithNamespace), nil
}

// encodeProjectID URL-encodes a GitLab project identifier so that "group/sub"
// becomes "group%2Fsub", which is required by GitLab v4 when the project ID
// is passed as a path parameter.
func encodeProjectID(id string) string {
	return url.QueryEscape(id)
}

// apiHost extracts the bare hostname from a BaseURL like
// "https://gitlab.com" or "https://gitlab.example.com/api/v4".
func apiHost(baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("gitlab: parse base URL: %w", err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("gitlab: base URL missing host")
	}
	return u.Host, nil
}
