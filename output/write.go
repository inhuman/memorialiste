package output

import (
	"fmt"
	"os"
	"path/filepath"

	mctx "github.com/inhuman/memorialiste/context"
)

// writeFile writes body to absPath with a generated_at frontmatter header,
// creating parent directories as needed.
func writeFile(absPath, body, sha string) error {
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return fmt.Errorf("output: mkdir %q: %w", filepath.Dir(absPath), err)
	}
	content := mctx.WriteFrontmatter(body, sha)
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("output: write %q: %w", absPath, err)
	}
	return nil
}

// writeFileNoFrontmatter writes body verbatim to absPath (no frontmatter
// prepended), creating parent directories as needed. Used in sidecar
// watermark mode.
func writeFileNoFrontmatter(absPath, body string) error {
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return fmt.Errorf("output: mkdir %q: %w", filepath.Dir(absPath), err)
	}
	if err := os.WriteFile(absPath, []byte(body), 0o644); err != nil {
		return fmt.Errorf("output: write %q: %w", absPath, err)
	}
	return nil
}
