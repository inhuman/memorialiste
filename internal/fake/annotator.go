package fake

import (
	"context"

	mctx "github.com/inhuman/memorialiste/context"
)

// Annotator is a test double for context.ASTAnnotator.
type Annotator struct {
	// AnnotateFunc is called by Annotate when non-nil.
	// When nil, Annotate returns an empty ASTAnnotation with no error.
	AnnotateFunc func(ctx context.Context, filePath string, changedLines []int) (mctx.ASTAnnotation, error)
}

// Annotate implements context.ASTAnnotator.
func (a *Annotator) Annotate(ctx context.Context, filePath string, changedLines []int) (mctx.ASTAnnotation, error) {
	if a.AnnotateFunc != nil {
		return a.AnnotateFunc(ctx, filePath, changedLines)
	}
	return mctx.ASTAnnotation{FilePath: filePath}, nil
}
