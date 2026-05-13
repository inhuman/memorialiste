package generate

import (
	"regexp"
	"strings"
)

var (
	// Preamble lines like "Here's the updated documentation:" or
	// "Here is the updated README:" at the very start of the response.
	preambleRE = regexp.MustCompile(`(?i)^here['']?s?\s+(?:is\s+)?the\s+updated[^\n]*\n+`)

	// Wrapping ```markdown / ```md / ``` fence pair around the whole body.
	// Matches a single leading fence and a single trailing fence.
	leadingFenceRE  = regexp.MustCompile("^```(?:markdown|md)?\\s*\\n")
	trailingFenceRE = regexp.MustCompile("\\n```\\s*$")
)

// Strip removes common LLM response artifacts from s:
//   - leading/trailing whitespace;
//   - leading preamble lines such as "Here's the updated documentation:";
//   - a single pair of wrapping triple-backtick code fences
//     (```, ```markdown, or ```md).
//
// Mid-content fences are left untouched.
func Strip(s string) string {
	s = strings.TrimSpace(s)
	s = preambleRE.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)

	// Only remove fences when both leading AND trailing match — otherwise the
	// content legitimately starts or ends with a fence and we shouldn't mangle it.
	leadLoc := leadingFenceRE.FindStringIndex(s)
	trailLoc := trailingFenceRE.FindStringIndex(s)
	if leadLoc != nil && trailLoc != nil && trailLoc[0] >= leadLoc[1] {
		s = s[leadLoc[1]:trailLoc[0]]
	}

	return strings.TrimSpace(s)
}
