package gitlab_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/inhuman/memorialiste/platform"
	"github.com/inhuman/memorialiste/platform/gitlab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testToken = "glpat-XXXX-test-token"

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := log.Writer()
	log.SetOutput(buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return buf
}

func TestPush_EmptyToken_ReturnsErrTokenRequired(t *testing.T) {
	logBuf := captureLog(t)
	p := gitlab.New(gitlab.Config{ProjectID: "x/y"})
	err := p.Push(t.Context(), "feature", "deadbeef")
	require.ErrorIs(t, err, platform.ErrTokenRequired)
	assert.NotContains(t, logBuf.String(), testToken)
}

func TestPush_ResolvesPathWithNamespace(t *testing.T) {
	logBuf := captureLog(t)

	var capturedTokenHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTokenHeader = r.Header.Get("PRIVATE-TOKEN")
		assert.Equal(t, "/api/v4/projects/123", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"path_with_namespace": "group/sub/project",
		})
	}))
	defer srv.Close()

	p := gitlab.New(gitlab.Config{
		BaseURL:   srv.URL,
		Token:     testToken,
		ProjectID: "123",
		RepoPath:  t.TempDir(), // not a real repo; push will fail after resolution
	})

	err := p.Push(t.Context(), "feature", "deadbeef")
	// We expect failure because RepoPath is not a real repo, but the API
	// resolution must have happened first.
	require.Error(t, err)
	assert.Equal(t, testToken, capturedTokenHeader)
	assert.NotContains(t, err.Error(), testToken)
	assert.NotContains(t, logBuf.String(), testToken)
}

func TestOpenChangeRequest_HappyPath(t *testing.T) {
	logBuf := captureLog(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v4/projects/g%2Fp/merge_requests", r.URL.EscapedPath())
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, testToken, r.Header.Get("PRIVATE-TOKEN"))

		body, _ := io.ReadAll(r.Body)
		var got map[string]string
		require.NoError(t, json.Unmarshal(body, &got))
		assert.Equal(t, "feature", got["source_branch"])
		assert.Equal(t, "main", got["target_branch"])
		assert.Equal(t, "Title", got["title"])
		assert.Equal(t, "Body", got["description"])

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"web_url":"https://gitlab.com/g/p/-/merge_requests/42","iid":42}`))
	}))
	defer srv.Close()

	p := gitlab.New(gitlab.Config{
		BaseURL:   srv.URL,
		Token:     testToken,
		ProjectID: "g/p",
	})

	res, err := p.OpenChangeRequest(t.Context(), platform.ChangeRequest{
		SourceBranch: "feature", TargetBranch: "main", Title: "Title", Body: "Body",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://gitlab.com/g/p/-/merge_requests/42", res.URL)
	assert.Equal(t, 42, res.Number)
	assert.NotContains(t, logBuf.String(), testToken)
}

func TestOpenChangeRequest_Non2xx_ReturnsHTTPError(t *testing.T) {
	logBuf := captureLog(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"validation failed"}`))
	}))
	defer srv.Close()

	p := gitlab.New(gitlab.Config{BaseURL: srv.URL, Token: testToken, ProjectID: "g/p"})

	_, err := p.OpenChangeRequest(t.Context(), platform.ChangeRequest{
		SourceBranch: "feature", TargetBranch: "main", Title: "T",
	})
	require.Error(t, err)
	httpErr, ok := errors.AsType[*platform.HTTPError](err)
	require.True(t, ok)
	assert.Equal(t, 422, httpErr.StatusCode)
	assert.Contains(t, httpErr.Body, "validation failed")
	assert.NotContains(t, logBuf.String(), testToken)
}

func TestOpenChangeRequest_Timeout(t *testing.T) {
	logBuf := captureLog(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer srv.Close()

	p := gitlab.New(gitlab.Config{
		BaseURL: srv.URL, Token: testToken, ProjectID: "g/p",
		Timeout: 50 * time.Millisecond,
	})

	_, err := p.OpenChangeRequest(t.Context(), platform.ChangeRequest{
		SourceBranch: "feature", TargetBranch: "main", Title: "T",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "deadline"))
	assert.NotContains(t, logBuf.String(), testToken)
}

func TestOpenChangeRequest_EmptyToken(t *testing.T) {
	p := gitlab.New(gitlab.Config{ProjectID: "g/p"})
	_, err := p.OpenChangeRequest(t.Context(), platform.ChangeRequest{})
	require.ErrorIs(t, err, platform.ErrTokenRequired)
}
