package codesearch

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// binaryExtensions mirrors context.binaryExtensions. Copied per
// research.md §4 to avoid coupling search and diff exclusion logic.
var binaryExtensions = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".svg": {},
	".pdf": {}, ".zip": {}, ".bin": {}, ".exe": {}, ".so": {}, ".dylib": {},
	".woff": {}, ".woff2": {}, ".ttf": {}, ".eot": {}, ".ico": {},
}

// walkGoFiles returns absolute paths of every .go file under root that is
// not excluded by the standard exclusion list. Directory walk errors are
// surfaced (e.g. permission denied on root); per-entry errors are skipped.
func walkGoFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "migrations" || name == "docs" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		if isExcluded(rel) {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// isExcluded mirrors context.isExcluded but is decoupled per research.md §4.
func isExcluded(path string) bool {
	if strings.HasPrefix(path, "vendor/") || strings.Contains(path, "/vendor/") {
		return true
	}
	if strings.HasSuffix(path, "_test.go") {
		return true
	}
	if strings.HasSuffix(path, ".gen.go") {
		return true
	}
	if strings.HasPrefix(path, "migrations/") {
		return true
	}
	if strings.HasPrefix(path, "docs/") {
		return true
	}
	idx := strings.LastIndexByte(path, '.')
	if idx >= 0 {
		if _, isBinary := binaryExtensions[path[idx:]]; isBinary {
			return true
		}
	}
	return false
}
