package codesearch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSearch_HappyPath_MixedDecls(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", `package a
func Foo() {}
func Bar() {}
type Baz struct{}
const Qux = 1
var Quux = 2
`)
	writeFile(t, dir, "sub/b.go", `package sub
func (r *T) Method() {}
type T struct{}
`)
	writeFile(t, dir, "ignored/c.py", `print("hi")`)

	res, err := Search(t.Context(), SearchRequest{RepoRoot: dir, Pattern: ".*"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) == 0 {
		t.Fatalf("expected hits, got 0")
	}
	gotKinds := map[string]string{}
	for _, h := range res.Hits {
		gotKinds[h.Name] = h.Kind
	}
	want := map[string]string{
		"Foo": "function", "Bar": "function",
		"Baz": "type", "Qux": "const", "Quux": "var",
		"Method": "method", "T": "type",
	}
	for n, k := range want {
		if gotKinds[n] != k {
			t.Errorf("kind for %q: got %q, want %q", n, gotKinds[n], k)
		}
	}
}

func TestSearch_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	_, err := Search(t.Context(), SearchRequest{RepoRoot: dir, Path: "../outside", Pattern: "Foo"})
	if err == nil {
		t.Fatalf("expected error for path traversal")
	}
	if !strings.Contains(err.Error(), "escapes") {
		t.Errorf("error should mention escape: %v", err)
	}
}

func TestSearch_InvalidRegex(t *testing.T) {
	dir := t.TempDir()
	_, err := Search(t.Context(), SearchRequest{RepoRoot: dir, Pattern: "["})
	if err == nil {
		t.Fatalf("expected error for invalid pattern")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("error should say invalid: %v", err)
	}
}

func TestSearch_NoGoFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "x.py", "print('hi')")
	res, err := Search(t.Context(), SearchRequest{RepoRoot: dir, Pattern: ".*"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) != 0 {
		t.Errorf("expected 0 hits, got %d", len(res.Hits))
	}
}

func TestSearch_Excluded(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "vendor/x.go", `package x
func InVendor() {}
`)
	writeFile(t, dir, "x_test.go", `package x
func InTest() {}
`)
	writeFile(t, dir, "y.gen.go", `package x
func InGen() {}
`)
	writeFile(t, dir, "main.go", `package main
func Main() {}
`)
	res, err := Search(t.Context(), SearchRequest{RepoRoot: dir, Pattern: ".*"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	for _, h := range res.Hits {
		if h.Name != "Main" {
			t.Errorf("unexpected hit: %+v", h)
		}
	}
	if len(res.Hits) != 1 {
		t.Errorf("expected only Main, got %d hits", len(res.Hits))
	}
}

func TestSearch_LimitTruncation(t *testing.T) {
	dir := t.TempDir()
	var sb strings.Builder
	sb.WriteString("package a\n")
	for i := range 30 {
		sb.WriteString(fmt.Sprintf("func F%d() {}\n", i))
	}
	writeFile(t, dir, "a.go", sb.String())

	res, err := Search(t.Context(), SearchRequest{RepoRoot: dir, Pattern: "^F", Limit: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) != 10 {
		t.Errorf("expected 10 hits, got %d", len(res.Hits))
	}
	if !res.Truncated {
		t.Errorf("expected Truncated=true")
	}
}

func TestSearch_LineCap(t *testing.T) {
	dir := t.TempDir()
	var sb strings.Builder
	sb.WriteString("package a\nfunc Big() {\n")
	for range 250 {
		sb.WriteString("\t_ = 1\n")
	}
	sb.WriteString("}\n")
	writeFile(t, dir, "a.go", sb.String())

	res, err := Search(t.Context(), SearchRequest{RepoRoot: dir, Pattern: "^Big$"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) != 1 {
		t.Fatalf("want 1 hit, got %d", len(res.Hits))
	}
	if !res.Hits[0].Truncated {
		t.Errorf("expected Truncated=true on big hit")
	}
	if !strings.Contains(res.Hits[0].Source, "truncated, original is") {
		t.Errorf("expected truncation footer")
	}
}

func TestSearch_SyntaxErrorSkipped(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bad.go", `package broken
this is not go code 123 ###`)
	writeFile(t, dir, "good.go", `package good
func Good() {}
`)
	res, err := Search(t.Context(), SearchRequest{RepoRoot: dir, Pattern: "Good"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Hits) != 1 || res.Hits[0].Name != "Good" {
		t.Errorf("expected Good hit only, got %+v", res.Hits)
	}
	if res.FilesSkipped < 1 {
		t.Errorf("expected FilesSkipped ≥ 1, got %d", res.FilesSkipped)
	}
}

func TestSearch_ConstVarMultipleNames(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", `package a
const (
	Alpha = 1
	Beta  = 2
)
var Gamma, Delta = 3, 4
`)
	res, err := Search(t.Context(), SearchRequest{RepoRoot: dir, Pattern: ".*"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	found := map[string]string{}
	for _, h := range res.Hits {
		found[h.Name] = h.Kind
	}
	for _, n := range []string{"Alpha", "Beta"} {
		if found[n] != "const" {
			t.Errorf("%q: want const, got %q", n, found[n])
		}
	}
	for _, n := range []string{"Gamma", "Delta"} {
		if found[n] != "var" {
			t.Errorf("%q: want var, got %q", n, found[n])
		}
	}
}
