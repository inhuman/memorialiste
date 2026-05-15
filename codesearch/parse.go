package codesearch

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
)

// parseGoFile parses path with a per-file timeout. The underlying
// parser.ParseFile is not context-aware, so it runs in a goroutine that
// may outlive the call when parsing is genuinely slow — abandoning a
// goroutine is acceptable here because it terminates as soon as parsing
// finishes and is bounded by file size.
func parseGoFile(parent context.Context, fset *token.FileSet, path string, src []byte) (*ast.File, error) {
	ctx, cancel := context.WithTimeout(parent, DefaultParseTimeout)
	defer cancel()

	type result struct {
		f   *ast.File
		err error
	}
	ch := make(chan result, 1)
	go func() {
		f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		ch <- result{f, err}
	}()

	select {
	case r := <-ch:
		return r.f, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
