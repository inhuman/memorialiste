package context

import "context"

// ExportedEnrichDiff exposes enrichDiff for external package tests.
func ExportedEnrichDiff(ctx context.Context, repoPath, rawDiff string, annotator ASTAnnotator) (string, bool, error) {
	return enrichDiff(ctx, repoPath, rawDiff, annotator)
}
