package github_test

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
	"github.com/inhuman/memorialiste/platform/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testToken = "ghp_XXXX-test-token"

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
	p := github.New(github.Config{Repository: "o/r"})
	err := p.Push(t.Context(), "feature", "deadbeef")
	require.ErrorIs(t, err, platform.ErrTokenRequired)
	assert.NotContains(t, logBuf.String(), testToken)
}

func TestOpenChangeRequest_HappyPath(t *testing.T) {
	logBuf := captureLog(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/repos/o/r/pulls", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer "+testToken, r.Header.Get("Authorization"))
		assert.Equal(t, "application/vnd.github+json", r.Header.Get("Accept"))
		assert.Equal(t, "2022-11-28", r.Header.Get("X-GitHub-Api-Version"))

		body, _ := io.ReadAll(r.Body)
		var got map[string]string
		require.NoError(t, json.Unmarshal(body, &got))
		assert.Equal(t, "feature", got["head"])
		assert.Equal(t, "main", got["base"])
		assert.Equal(t, "Title", got["title"])
		assert.Equal(t, "Body", got["body"])

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"html_url":"https://github.com/o/r/pull/7","number":7}`))
	}))
	defer srv.Close()

	p := github.New(github.Config{
		BaseURL: srv.URL, Token: testToken, Repository: "o/r",
	})

	res, err := p.OpenChangeRequest(t.Context(), platform.ChangeRequest{
		SourceBranch: "feature", TargetBranch: "main", Title: "Title", Body: "Body",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/o/r/pull/7", res.URL)
	assert.Equal(t, 7, res.Number)
	assert.NotContains(t, logBuf.String(), testToken)
}

func TestOpenChangeRequest_Non2xx_ReturnsHTTPError(t *testing.T) {
	logBuf := captureLog(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Validation Failed"}`))
	}))
	defer srv.Close()

	p := github.New(github.Config{BaseURL: srv.URL, Token: testToken, Repository: "o/r"})

	_, err := p.OpenChangeRequest(t.Context(), platform.ChangeRequest{
		SourceBranch: "feature", TargetBranch: "main", Title: "T",
	})
	require.Error(t, err)
	httpErr, ok := errors.AsType[*platform.HTTPError](err)
	require.True(t, ok)
	assert.Equal(t, 422, httpErr.StatusCode)
	assert.Contains(t, httpErr.Body, "Validation Failed")
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

	p := github.New(github.Config{
		BaseURL: srv.URL, Token: testToken, Repository: "o/r",
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
	p := github.New(github.Config{Repository: "o/r"})
	_, err := p.OpenChangeRequest(t.Context(), platform.ChangeRequest{})
	require.ErrorIs(t, err, platform.ErrTokenRequired)
}
