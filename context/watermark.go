package context

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type frontmatter struct {
	GeneratedAt string `yaml:"generated_at"`
}

// ReadWatermark returns the generated_at SHA from the YAML frontmatter of the
// file at path. Returns ("", nil) when the file does not exist, has no
// frontmatter, or the generated_at key is absent.
func ReadWatermark(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("context: read watermark %q: %w", path, err)
	}
	block, _, ok := parseFrontmatterBlock(string(data))
	if !ok {
		return "", nil
	}
	var fm frontmatter
	if err := yaml.Unmarshal([]byte(block), &fm); err != nil {
		return "", fmt.Errorf("context: parse frontmatter %q: %w", path, err)
	}
	return fm.GeneratedAt, nil
}

// StripFrontmatter returns the body of a doc file with the leading YAML
// frontmatter block removed. Returns content unchanged when no frontmatter
// is present.
func StripFrontmatter(content string) string {
	_, body, ok := parseFrontmatterBlock(content)
	if !ok {
		return content
	}
	return body
}

// WriteFrontmatter prepends a YAML frontmatter block with sha to body.
func WriteFrontmatter(body string, sha string) string {
	return "---\ngenerated_at: " + sha + "\n---\n\n" + body
}

// parseFrontmatterBlock splits content into (frontmatterYAML, body, found).
// found is false when content does not start with "---\n".
func parseFrontmatterBlock(content string) (block, body string, found bool) {
	const sep = "---"
	r := strings.NewReader(content)
	lines := readLines(r)

	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != sep {
		return "", content, false
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == sep {
			block = strings.Join(lines[1:i], "\n")
			body = strings.Join(lines[i+1:], "\n")
			// Trim a single leading newline from body.
			body = strings.TrimPrefix(body, "\n")
			return block, body, true
		}
	}
	// No closing ---, treat as no frontmatter.
	return "", content, false
}

func readLines(r io.Reader) []string {
	data, _ := io.ReadAll(r)
	if len(data) == 0 {
		return nil
	}
	return strings.Split(string(data), "\n")
}
