package generate_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inhuman/memorialiste/generate"
	"github.com/inhuman/memorialiste/internal/fake"
	"github.com/inhuman/memorialiste/provider"
)

func writeGoFile(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestToolLoop_NoToolCalls(t *testing.T) {
	dir := t.TempDir()
	f := &fake.Provider{
		CompleteWithToolsFunc: func(ctx context.Context, msgs []provider.Message, tools []provider.ToolSchema) (provider.Step, provider.TokenUsage, error) {
			return provider.Step{Content: "OK"}, provider.TokenUsage{TotalTokens: 5}, nil
		},
	}
	res, err := generate.Generate(t.Context(), generate.Input{
		DocBody: "doc", Diff: "diff", CodeSearch: true, RepoRoot: dir,
	}, f)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if res.Content != "OK" {
		t.Errorf("Content: got %q, want OK", res.Content)
	}
	if res.TokenUsage.TotalTokens != 5 {
		t.Errorf("TokenUsage: %+v", res.TokenUsage)
	}
}

func TestToolLoop_OneCallThenText(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "x.go", "package x\nfunc Foo() {}\n")
	var capturedMessages [][]provider.Message
	calls := 0
	f := &fake.Provider{
		CompleteWithToolsFunc: func(ctx context.Context, msgs []provider.Message, tools []provider.ToolSchema) (provider.Step, provider.TokenUsage, error) {
			capturedMessages = append(capturedMessages, msgs)
			calls++
			if calls == 1 {
				return provider.Step{ToolCalls: []provider.ToolCall{{
					ID: "call_1", Name: "search_code", Arguments: `{"pattern":"Foo"}`,
				}}}, provider.TokenUsage{}, nil
			}
			return provider.Step{Content: "docs body"}, provider.TokenUsage{}, nil
		},
	}
	res, err := generate.Generate(t.Context(), generate.Input{
		CodeSearch: true, RepoRoot: dir,
	}, f)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if res.Content != "docs body" {
		t.Errorf("Content: got %q", res.Content)
	}
	if calls != 2 {
		t.Errorf("expected 2 provider calls, got %d", calls)
	}
	// Second-call messages must contain the tool result.
	second := capturedMessages[1]
	var foundTool bool
	for _, m := range second {
		if m.Role == "tool" && m.ToolCallID == "call_1" {
			foundTool = true
			if !strings.Contains(m.Content, "Foo") {
				t.Errorf("tool result must contain Foo: %s", m.Content)
			}
		}
	}
	if !foundTool {
		t.Errorf("tool message missing from second-call messages")
	}
}

func TestToolLoop_MultipleCallsInOneTurn(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "x.go", "package x\nfunc Foo() {}\nfunc Bar() {}\n")
	turns := 0
	f := &fake.Provider{
		CompleteWithToolsFunc: func(ctx context.Context, msgs []provider.Message, tools []provider.ToolSchema) (provider.Step, provider.TokenUsage, error) {
			turns++
			if turns == 1 {
				return provider.Step{ToolCalls: []provider.ToolCall{
					{ID: "a", Name: "search_code", Arguments: `{"pattern":"Foo"}`},
					{ID: "b", Name: "search_code", Arguments: `{"pattern":"Bar"}`},
				}}, provider.TokenUsage{}, nil
			}
			return provider.Step{Content: "done"}, provider.TokenUsage{}, nil
		},
	}
	res, err := generate.Generate(t.Context(), generate.Input{CodeSearch: true, RepoRoot: dir}, f)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if res.Content != "done" {
		t.Errorf("Content: %q", res.Content)
	}
	if turns != 2 {
		t.Errorf("turns: got %d, want 2", turns)
	}
}

func TestToolLoop_MaxTurnsExceeded(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "x.go", "package x\nfunc Foo() {}\n")
	f := &fake.Provider{
		CompleteWithToolsFunc: func(ctx context.Context, msgs []provider.Message, tools []provider.ToolSchema) (provider.Step, provider.TokenUsage, error) {
			return provider.Step{ToolCalls: []provider.ToolCall{
				{ID: "a", Name: "search_code", Arguments: `{"pattern":"Foo"}`},
			}}, provider.TokenUsage{}, nil
		},
	}
	_, err := generate.Generate(t.Context(), generate.Input{CodeSearch: true, RepoRoot: dir}, f)
	if err == nil {
		t.Fatalf("expected MaxTurns error")
	}
	if !strings.Contains(err.Error(), "10") {
		t.Errorf("expected error mentioning 10: %v", err)
	}
}

func TestToolLoop_CustomMaxTurns(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "x.go", "package x\nfunc Foo() {}\n")
	turns := 0
	f := &fake.Provider{
		CompleteWithToolsFunc: func(ctx context.Context, msgs []provider.Message, tools []provider.ToolSchema) (provider.Step, provider.TokenUsage, error) {
			turns++
			return provider.Step{ToolCalls: []provider.ToolCall{
				{ID: fmt.Sprintf("c%d", turns), Name: "search_code", Arguments: `{"pattern":"Foo"}`},
			}}, provider.TokenUsage{}, nil
		},
	}
	_, err := generate.Generate(t.Context(), generate.Input{CodeSearch: true, RepoRoot: dir, MaxTurns: 3}, f)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "3") {
		t.Errorf("expected error mentioning 3: %v", err)
	}
	if turns != 3 {
		t.Errorf("turns: got %d, want 3", turns)
	}
}

func TestGenerate_CodeSearchDisabled_TakesSingleShot(t *testing.T) {
	dir := t.TempDir()
	called := false
	f := &fake.Provider{
		CompleteFunc: func(ctx context.Context, msgs []provider.Message) (string, provider.TokenUsage, error) {
			called = true
			return "single-shot", provider.TokenUsage{}, nil
		},
		CompleteWithToolsFunc: func(ctx context.Context, msgs []provider.Message, tools []provider.ToolSchema) (provider.Step, provider.TokenUsage, error) {
			t.Fatalf("CompleteWithTools called when CodeSearch=false")
			return provider.Step{}, provider.TokenUsage{}, nil
		},
	}
	res, err := generate.Generate(t.Context(), generate.Input{
		DocBody: "doc", Diff: "diff", CodeSearch: false, RepoRoot: dir,
	}, f)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !called {
		t.Errorf("CompleteFunc not called")
	}
	if res.Content != "single-shot" {
		t.Errorf("Content: %q", res.Content)
	}
}

// plainProvider implements only provider.Provider (not ToolingProvider).
type plainProvider struct {
	called bool
}

func (p *plainProvider) Complete(ctx context.Context, messages []provider.Message) (string, provider.TokenUsage, error) {
	p.called = true
	return "plain-out", provider.TokenUsage{}, nil
}

func TestGenerate_CodeSearchEnabled_ProviderLacksTooling(t *testing.T) {
	dir := t.TempDir()
	p := &plainProvider{}
	res, err := generate.Generate(t.Context(), generate.Input{
		DocBody: "d", Diff: "diff", CodeSearch: true, RepoRoot: dir,
	}, p)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !p.called {
		t.Errorf("plain provider Complete not called (expected graceful fallthrough)")
	}
	if res.Content != "plain-out" {
		t.Errorf("Content: %q", res.Content)
	}
}

// US2 — logging
func TestToolLoop_LogEntriesPerCall(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "x.go", "package x\nfunc Foo() {}\n")

	var buf bytes.Buffer
	origOut := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(origOut) })

	turns := 0
	f := &fake.Provider{
		CompleteWithToolsFunc: func(ctx context.Context, msgs []provider.Message, tools []provider.ToolSchema) (provider.Step, provider.TokenUsage, error) {
			turns++
			if turns <= 3 {
				return provider.Step{ToolCalls: []provider.ToolCall{
					{ID: fmt.Sprintf("c%d", turns), Name: "search_code", Arguments: `{"pattern":"Foo","path":"."}`},
				}}, provider.TokenUsage{}, nil
			}
			return provider.Step{Content: "done"}, provider.TokenUsage{}, nil
		},
	}
	_, err := generate.Generate(t.Context(), generate.Input{CodeSearch: true, RepoRoot: dir}, f)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	count := strings.Count(buf.String(), "code-search: turn=")
	if count != 3 {
		t.Errorf("expected 3 log lines, got %d (buf=%s)", count, buf.String())
	}
}

func TestToolLoop_LogIncludesPatternAndPath(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "x.go", "package x\nfunc Foo() {}\n")

	var buf bytes.Buffer
	origOut := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(origOut) })

	turns := 0
	f := &fake.Provider{
		CompleteWithToolsFunc: func(ctx context.Context, msgs []provider.Message, tools []provider.ToolSchema) (provider.Step, provider.TokenUsage, error) {
			turns++
			if turns == 1 {
				return provider.Step{ToolCalls: []provider.ToolCall{
					{ID: "x", Name: "search_code", Arguments: `{"pattern":"MyPattern","path":"sub/"}`},
				}}, provider.TokenUsage{}, nil
			}
			return provider.Step{Content: "done"}, provider.TokenUsage{}, nil
		},
	}
	_, err := generate.Generate(t.Context(), generate.Input{CodeSearch: true, RepoRoot: dir}, f)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "MyPattern") || !strings.Contains(out, "sub/") {
		t.Errorf("log missing pattern or path: %s", out)
	}
}

func TestToolLoop_UnparseableFileSkipped(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "good.go", "package g\nfunc Good() {}\n")
	writeGoFile(t, dir, "bad.go", "this is not go ###")

	var resultContent string
	turns := 0
	f := &fake.Provider{
		CompleteWithToolsFunc: func(ctx context.Context, msgs []provider.Message, tools []provider.ToolSchema) (provider.Step, provider.TokenUsage, error) {
			turns++
			if turns == 1 {
				return provider.Step{ToolCalls: []provider.ToolCall{
					{ID: "x", Name: "search_code", Arguments: `{"pattern":".*"}`},
				}}, provider.TokenUsage{}, nil
			}
			// capture last tool result for assertion
			for _, m := range msgs {
				if m.Role == "tool" {
					resultContent = m.Content
				}
			}
			return provider.Step{Content: "done"}, provider.TokenUsage{}, nil
		},
	}
	_, err := generate.Generate(t.Context(), generate.Input{CodeSearch: true, RepoRoot: dir}, f)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(resultContent, "Good") {
		t.Errorf("tool result should include Good: %s", resultContent)
	}
	if !strings.Contains(resultContent, "files_skipped") {
		t.Errorf("tool result must report files_skipped: %s", resultContent)
	}
}

// US3
func TestGenerate_ToolsUnsupported_ActionableError(t *testing.T) {
	dir := t.TempDir()
	f := &fake.Provider{
		CompleteWithToolsFunc: func(ctx context.Context, msgs []provider.Message, tools []provider.ToolSchema) (provider.Step, provider.TokenUsage, error) {
			return provider.Step{}, provider.TokenUsage{}, fmt.Errorf("wrap: %w", provider.ErrToolsNotSupported)
		},
	}
	_, err := generate.Generate(t.Context(), generate.Input{CodeSearch: true, RepoRoot: dir}, f)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, provider.ErrToolsNotSupported) {
		t.Errorf("expected errors.Is(err, ErrToolsNotSupported)")
	}
	if !strings.Contains(err.Error(), "--code-search=false") {
		t.Errorf("expected actionable hint: %v", err)
	}
}
