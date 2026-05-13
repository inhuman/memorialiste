package context

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ASTAnnotation is the structural context for one changed file.
type ASTAnnotation struct {
	// FilePath is the repo-relative path of the file.
	FilePath string
	// Scopes contains the deduplicated names of functions or methods that
	// contain at least one changed line.
	Scopes []string
	// FileLevel is true when at least one changed line falls outside any
	// named function.
	FileLevel bool
}

// ASTAnnotator determines which functions contain the changed lines of a file.
type ASTAnnotator interface {
	// Annotate returns the AST annotation for filePath.
	// changedLines contains 1-based line numbers that appear in the diff.
	// On timeout or analysis failure it returns an empty annotation and nil
	// error — the caller falls back to the unenriched diff.
	Annotate(ctx context.Context, filePath string, changedLines []int) (ASTAnnotation, error)
}

// grepASTAnnotator invokes the grep-ast CLI to determine function scopes.
type grepASTAnnotator struct {
	repoPath string
}

const astTimeout = 10 * time.Second

func (g *grepASTAnnotator) Annotate(ctx context.Context, filePath string, changedLines []int) (ASTAnnotation, error) {
	ann := ASTAnnotation{FilePath: filePath}
	seen := map[string]struct{}{}

	absPath := filePath
	if !strings.HasPrefix(filePath, "/") {
		absPath = g.repoPath + "/" + filePath
	}

	for _, line := range changedLines {
		tctx, cancel := context.WithTimeout(ctx, astTimeout)
		scopes, fileLevel, err := runGrepAST(tctx, absPath, line)
		cancel()
		if err != nil {
			// timeout or binary unavailable — return empty annotation
			return ASTAnnotation{FilePath: filePath}, nil
		}
		if fileLevel {
			ann.FileLevel = true
		}
		for _, s := range scopes {
			if _, ok := seen[s]; !ok {
				seen[s] = struct{}{}
				ann.Scopes = append(ann.Scopes, s)
			}
		}
	}
	return ann, nil
}

// runGrepAST runs `grep-ast --no-color -n <lineNum> <file>` and parses
// the enclosing function names from the output.
func runGrepAST(ctx context.Context, absPath string, lineNum int) (scopes []string, fileLevel bool, err error) {
	// grep-ast matches by pattern; use the line number as a line-number grep
	// via the -n flag is not a line filter — we pass a pattern that matches
	// everything and filter by line number ourselves using the output format.
	// Simpler: grep-ast accepts a pattern; pass "." (matches any line) and
	// use --no-color output to find the enclosing func for our line.
	// We use a Python one-liner to call TreeContext directly and get the
	// scope for the specific line.
	script := fmt.Sprintf(`
import sys
try:
    from grep_ast.grep_ast import TreeContext
    code = open(%q, encoding='utf-8', errors='replace').read()
    tc = TreeContext(%q, code, color=False, verbose=False, line_number=True)
    tc.add_lines_of_interest([%d])
    tc.add_context()
    print(tc.format())
except Exception as e:
    sys.stderr.write(str(e)+"\n")
    sys.exit(1)
`, absPath, absPath, lineNum)

	cmd := exec.CommandContext(ctx, "python3", "-c", script)
	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return nil, false, fmt.Errorf("grep-ast: %w", err)
	}

	return parseGrepASTOutput(out.String(), lineNum)
}

// parseGrepASTOutput extracts function names from grep-ast text output.
// grep-ast output looks like:
//
//	context/diff.go:
//	│ func computeDiff(...) {
//	│ ...
func parseGrepASTOutput(output string, _ int) (scopes []string, fileLevel bool, err error) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	seen := map[string]struct{}{}
	foundFunc := false

	for scanner.Scan() {
		line := scanner.Text()
		// Strip tree-drawing characters and leading whitespace.
		clean := strings.TrimLeft(line, "│├└─ \t")
		if strings.HasPrefix(clean, "func ") || strings.Contains(clean, ") ") && strings.HasPrefix(clean, "func ") {
			name := extractFuncName(clean)
			if name != "" {
				if _, ok := seen[name]; !ok {
					seen[name] = struct{}{}
					scopes = append(scopes, name)
				}
				foundFunc = true
			}
		}
	}

	if !foundFunc && output != "" {
		fileLevel = true
	}
	return scopes, fileLevel, nil
}

// extractFuncName extracts the function or method name from a Go func declaration line.
func extractFuncName(line string) string {
	// Handles: "func Foo(...)" and "func (r *Receiver) Bar(...)"
	after, ok := strings.CutPrefix(line, "func ")
	if !ok {
		return ""
	}
	// Skip receiver: "(r *Receiver) Bar" → "Bar"
	if strings.HasPrefix(after, "(") {
		close := strings.Index(after, ")")
		if close < 0 {
			return ""
		}
		after = strings.TrimSpace(after[close+1:])
	}
	// Take up to first "(" — that's the function name.
	paren := strings.IndexByte(after, '(')
	if paren < 0 {
		return strings.TrimSpace(after)
	}
	return strings.TrimSpace(after[:paren])
}

// enrichDiff enriches rawDiff with AST scope headers per file.
// Returns the enriched diff text, whether any file was enriched, and any error.
func enrichDiff(ctx context.Context, repoPath string, rawDiff string, annotator ASTAnnotator) (string, bool, error) {
	files := parseDiffFiles(rawDiff)
	if len(files) == 0 {
		return rawDiff, false, nil
	}

	var sb strings.Builder
	anyEnriched := false

	for _, f := range files {
		ann, err := annotator.Annotate(ctx, f.path, f.changedLines)
		if err != nil {
			return "", false, err
		}

		header := buildHeader(f.path, ann)
		enriched := len(ann.Scopes) > 0 || ann.FileLevel
		if enriched {
			anyEnriched = true
		}

		sb.WriteString(header)
		sb.WriteString("\n")
		sb.WriteString(f.raw)
	}

	return sb.String(), anyEnriched, nil
}

type diffFile struct {
	path         string
	raw          string
	changedLines []int
}

// parseDiffFiles splits a unified diff into per-file sections and extracts
// changed line numbers (lines starting with "+", excluding "+++").
func parseDiffFiles(diff string) []diffFile {
	var files []diffFile
	var cur *diffFile
	lineNum := 0

	scanner := bufio.NewScanner(strings.NewReader(diff))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "diff --git ") {
			if cur != nil {
				files = append(files, *cur)
			}
			// Extract b-side filename: "diff --git a/foo b/foo" → "foo"
			parts := strings.Fields(line)
			path := ""
			if len(parts) >= 4 {
				path = strings.TrimPrefix(parts[3], "b/")
			}
			cur = &diffFile{path: path, raw: line + "\n"}
			lineNum = 0
			continue
		}

		if cur == nil {
			continue
		}

		cur.raw += line + "\n"

		// Track current line position from @@ hunk headers.
		if strings.HasPrefix(line, "@@ ") {
			lineNum = parseHunkStart(line)
			continue
		}

		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			cur.changedLines = append(cur.changedLines, lineNum)
		}
		if !strings.HasPrefix(line, "-") {
			lineNum++
		}
	}

	if cur != nil {
		files = append(files, *cur)
	}
	return files
}

// parseHunkStart extracts the starting line number of the +++ side from a
// unified diff hunk header like "@@ -1,4 +10,6 @@ func Foo()".
func parseHunkStart(line string) int {
	// Format: @@ -a,b +c,d @@
	plus := strings.Index(line, " +")
	if plus < 0 {
		return 0
	}
	rest := line[plus+2:]
	comma := strings.IndexAny(rest, ", @")
	numStr := rest
	if comma >= 0 {
		numStr = rest[:comma]
	}
	n := 0
	fmt.Sscanf(numStr, "%d", &n)
	return n
}

func buildHeader(path string, ann ASTAnnotation) string {
	if len(ann.Scopes) == 0 && !ann.FileLevel {
		return "=== " + path + " ==="
	}
	parts := make([]string, 0, len(ann.Scopes)+1)
	parts = append(parts, ann.Scopes...)
	if ann.FileLevel {
		parts = append(parts, "(package-level)")
	}
	return "=== " + path + " [" + strings.Join(parts, ", ") + "] ==="
}
