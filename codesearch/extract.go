package codesearch

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"
)

// extractHits walks file.Decls and returns one SearchHit per matched
// declaration. For const/var GenDecls with multiple names per spec, one
// hit is emitted per Name. The last argument is unused but kept for
// future symbol-position filtering.
func extractHits(fset *token.FileSet, file *ast.File, src []byte, relPath string, pattern *regexp.Regexp, _ token.Pos) []SearchHit {
	if file == nil {
		return nil
	}
	var hits []SearchHit
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name == nil {
				continue
			}
			if !pattern.MatchString(d.Name.Name) {
				continue
			}
			kind := "function"
			if d.Recv != nil {
				kind = "method"
			}
			hits = append(hits, buildHit(fset, src, relPath, d.Name.Name, kind, d.Pos(), d.End()))
		case *ast.GenDecl:
			switch d.Tok {
			case token.TYPE:
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || ts.Name == nil {
						continue
					}
					if !pattern.MatchString(ts.Name.Name) {
						continue
					}
					hits = append(hits, buildHit(fset, src, relPath, ts.Name.Name, "type", d.Pos(), d.End()))
				}
			case token.CONST, token.VAR:
				kind := "const"
				if d.Tok == token.VAR {
					kind = "var"
				}
				for _, spec := range d.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, name := range vs.Names {
						if name == nil {
							continue
						}
						if !pattern.MatchString(name.Name) {
							continue
						}
						hits = append(hits, buildHit(fset, src, relPath, name.Name, kind, d.Pos(), d.End()))
					}
				}
			}
		}
	}
	return hits
}

func buildHit(fset *token.FileSet, src []byte, relPath, name, kind string, pos, end token.Pos) SearchHit {
	startPos := fset.Position(pos)
	endPos := fset.Position(end)
	startOff := startPos.Offset
	endOff := endPos.Offset
	if startOff < 0 {
		startOff = 0
	}
	if endOff > len(src) {
		endOff = len(src)
	}
	body := ""
	if startOff <= endOff && startOff >= 0 && endOff <= len(src) {
		body = string(src[startOff:endOff])
	}
	hit := SearchHit{
		Name:      name,
		Kind:      kind,
		FilePath:  relPath,
		StartLine: startPos.Line,
		EndLine:   endPos.Line,
		Source:    body,
	}
	if total := hit.EndLine - hit.StartLine + 1; total > DefaultLineCap {
		hit.Source = capLines(body, DefaultLineCap, total)
		hit.Truncated = true
	}
	return hit
}

func capLines(body string, cap, original int) string {
	lines := strings.SplitN(body, "\n", cap+1)
	if len(lines) > cap {
		lines = lines[:cap]
	}
	return strings.Join(lines, "\n") + fmt.Sprintf("\n... (truncated, original is %d lines)", original)
}
