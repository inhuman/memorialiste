package generate

import (
	"fmt"
	"os"
	"strings"
)

const builtInSystemPrompt = `You are a technical documentation writer. You will be given:
1. The current content of a documentation file.
2. A git diff of the source code changes since the documentation was last
   updated.
3. A description of the audience and purpose of this documentation file.

Your task:
- Update the documentation to reflect the changes shown in the diff.
- Preserve sections that are still accurate; rewrite only what has changed.
- Do not invent features not visible in the diff.
- Write in {language}.
- Return only the updated documentation content in Markdown (no frontmatter,
  no preamble, no explanation).`

// BuiltInSystemPrompt returns the default system prompt template with the
// {language} placeholder unresolved. Useful for callers who want to inspect
// or extend the default prompt.
func BuiltInSystemPrompt() string {
	return builtInSystemPrompt
}

// loadSystemPrompt resolves the system prompt source for one generation call.
//
// Resolution rules:
//   - empty raw → built-in default
//   - "@path"   → read file at path; fail-fast on read error
//   - otherwise → raw literal
//
// After resolution, "{language}" is substituted with the supplied language.
func loadSystemPrompt(raw, language string) (string, error) {
	var prompt string

	switch {
	case raw == "":
		prompt = builtInSystemPrompt
	case strings.HasPrefix(raw, "@"):
		path := strings.TrimPrefix(raw, "@")
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("generate: system prompt file %q: %w", path, err)
		}
		prompt = string(data)
	default:
		prompt = raw
	}

	return strings.Replace(prompt, "{language}", language, 1), nil
}
