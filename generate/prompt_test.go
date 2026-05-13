package generate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/inhuman/memorialiste/generate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltInSystemPrompt_ContainsLanguagePlaceholder(t *testing.T) {
	p := generate.BuiltInSystemPrompt()
	assert.Contains(t, p, "{language}")
	assert.Contains(t, p, "documentation")
}

// loadSystemPrompt is exercised indirectly via Generate; the integration
// tests below build a fake provider that captures the system message.

func TestLoadSystemPrompt_BuiltInWithLanguage(t *testing.T) {
	system := callGenerateAndCaptureSystem(t, "", "english")
	assert.Contains(t, system, "Write in english.")
	assert.NotContains(t, system, "{language}")
}

func TestLoadSystemPrompt_LiteralWithLanguage(t *testing.T) {
	system := callGenerateAndCaptureSystem(t, "literal {language}", "russian")
	assert.Equal(t, "literal russian", system)
}

func TestLoadSystemPrompt_FileSubstitutesLanguage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.txt")
	require.NoError(t, os.WriteFile(path, []byte("from file in {language}"), 0o644))

	system := callGenerateAndCaptureSystem(t, "@"+path, "spanish")
	assert.Equal(t, "from file in spanish", system)
}

func TestLoadSystemPrompt_MissingFile_Error(t *testing.T) {
	_, err := generate.Generate(
		t.Context(),
		generate.Input{SystemPrompt: "@/nonexistent/path/to/prompt.txt"},
		&fakeNoOpProvider{},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "/nonexistent/path/to/prompt.txt")
}
