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
	// Rendered is the full grep-ast TreeContext rendering of the file with
	// changed lines marked. Empty when the file is unsupported or the
	// renderer failed — callers should fall back to the raw diff in that case.
	Rendered string
}

// ASTAnnotator determines which functions contain the changed lines of a file.
type ASTAnnotator interface {
	// Annotate returns the AST annotation for filePath.
	// changedLines contains 1-based line numbers that appear in the diff.
	// On timeout or analysis failure it returns an empty annotation and nil
	// error — the caller falls back to the unenriched diff.
	Annotate(ctx context.Context, filePath string, changedLines []int) (ASTAnnotation, error)
}

// grepASTAnnotator invokes grep-ast's TreeContext via a Python helper to
// render a structural view of the file with changed lines marked.
type grepASTAnnotator struct {
	repoPath string
}

const astTimeout = 10 * time.Second

func (g *grepASTAnnotator) Annotate(ctx context.Context, filePath string, changedLines []int) (ASTAnnotation, error) {
	ann := ASTAnnotation{FilePath: filePath}

	if len(changedLines) == 0 {
		return ann, nil
	}

	absPath := filePath
	if !strings.HasPrefix(filePath, "/") {
		absPath = g.repoPath + "/" + filePath
	}

	tctx, cancel := context.WithTimeout(ctx, astTimeout)
	defer cancel()

	rendered, err := runGrepAST(tctx, absPath, changedLines)
	if err != nil {
		// timeout, parser unavailable, unsupported language — return empty.
		return ASTAnnotation{FilePath: filePath}, nil
	}
	ann.Rendered = rendered
	return ann, nil
}

// runGrepAST renders absPath using grep-ast's TreeContext with changedLines
// (1-based) marked as lines of interest. Returns the rendered output.
//
// grep-ast 0.5.0 with tree-sitter 0.20.4 + tree-sitter-languages 1.10.2 is the
// known-working pinned combo; newer versions break TreeContext.
func runGrepAST(ctx context.Context, absPath string, changedLines []int) (string, error) {
	// Build a Python list literal of 0-indexed line numbers.
	var lineParts []string
	for _, l := range changedLines {
		lineParts = append(lineParts, fmt.Sprintf("%d", l-1))
	}
	linesLit := "[" + strings.Join(lineParts, ", ") + "]"

	script := fmt.Sprintf(`
import sys
try:
    from grep_ast import TreeContext
    code = open(%q, encoding='utf-8', errors='replace').read()
    tc = TreeContext(%q, code, color=False, line_number=True)
    tc.add_lines_of_interest(%s)
    tc.add_context()
    sys.stdout.write(tc.format())
except Exception as e:
    sys.stderr.write(str(e)+"\n")
    sys.exit(1)
`, absPath, absPath, linesLit)

	cmd := exec.CommandContext(ctx, "python3", "-c", script)
	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("grep-ast: %w (stderr: %s)", err, errBuf.String())
	}

	return out.String(), nil
}

// enrichDiff enriches rawDiff with per-file AST rendering. When a file's
// annotation has non-empty Rendered, the rendering replaces the raw diff
// hunks for that file (TreeContext already shows changed lines in context).
// Otherwise the raw diff is emitted as-is.
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

		sb.WriteString("=== ")
		sb.WriteString(f.path)
		sb.WriteString(" ===\n")
		if ann.Rendered != "" {
			anyEnriched = true
			sb.WriteString(ann.Rendered)
			if !strings.HasSuffix(ann.Rendered, "\n") {
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString(f.raw)
		}
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
