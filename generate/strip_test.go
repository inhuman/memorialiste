package generate_test

import (
	"testing"

	"github.com/inhuman/memorialiste/generate"
	"github.com/stretchr/testify/assert"
)

func TestStrip_RemovesLeadingPreamble(t *testing.T) {
	in := "Here's the updated documentation:\n\n# Title\n\nBody"
	out := generate.Strip(in)
	assert.Equal(t, "# Title\n\nBody", out)
}

func TestStrip_RemovesPreambleVariations(t *testing.T) {
	cases := []string{
		"Here is the updated README:\n# Title",
		"Here's the updated docs:\n# Title",
		"HERE'S THE UPDATED DOCUMENTATION:\n# Title",
	}
	for _, in := range cases {
		out := generate.Strip(in)
		assert.Equal(t, "# Title", out, "input: %q", in)
	}
}

func TestStrip_RemovesMarkdownFences(t *testing.T) {
	in := "```markdown\n# Title\n\nBody\n```"
	out := generate.Strip(in)
	assert.Equal(t, "# Title\n\nBody", out)
}

func TestStrip_RemovesMdFences(t *testing.T) {
	in := "```md\n# Title\n```"
	out := generate.Strip(in)
	assert.Equal(t, "# Title", out)
}

func TestStrip_RemovesBareFences(t *testing.T) {
	in := "```\n# Title\n\nBody\n```"
	out := generate.Strip(in)
	assert.Equal(t, "# Title\n\nBody", out)
}

func TestStrip_LeavesMidContentFencesAlone(t *testing.T) {
	in := "# Title\n\n```go\nfunc main() {}\n```\n\nMore text"
	out := generate.Strip(in)
	assert.Equal(t, in, out, "mid-content fences must not be stripped")
}

func TestStrip_TrimsWhitespace(t *testing.T) {
	in := "\n\n   # Title\n\nBody   \n\n"
	out := generate.Strip(in)
	assert.Equal(t, "# Title\n\nBody", out)
}

func TestStrip_OnlyLeadingFenceNoTrailing_Untouched(t *testing.T) {
	in := "```markdown\n# Title without closing fence"
	out := generate.Strip(in)
	// Should NOT strip since pair doesn't match
	assert.Equal(t, in, out)
}
