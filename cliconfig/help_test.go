package cliconfig

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

func TestHelp_ListsAllFlagsAndEnvVars(t *testing.T) {
	var stdout, stderr bytes.Buffer
	_, err := parseWithOptions([]string{"--help"}, func(string) string { return "" },
		kong.Writers(&stdout, &stderr),
		kong.Exit(func(int) {}),
	)
	// kong returns no error on --help (it just exits, which we swallowed).
	_ = err

	out := stdout.String() + stderr.String()

	flags := []string{
		"--provider-url", "--model", "--model-params", "--api-key",
		"--system-prompt", "--prompt", "--language",
		"--doc-structure", "--repo",
		"--token-budget", "--dry-run", "--branch-prefix", "--ast-context",
		"--platform", "--platform-url", "--platform-token", "--project-id", "--base-branch",
		"--version",
	}
	for _, f := range flags {
		if !strings.Contains(out, f) {
			t.Errorf("help output missing flag %q\n---\n%s", f, out)
		}
	}

	envs := []string{
		"MEMORIALISTE_PROVIDER_URL", "MEMORIALISTE_MODEL", "MEMORIALISTE_MODEL_PARAMS",
		"MEMORIALISTE_API_KEY", "MEMORIALISTE_SYSTEM_PROMPT", "MEMORIALISTE_PROMPT",
		"MEMORIALISTE_LANGUAGE", "MEMORIALISTE_DOC_STRUCTURE", "MEMORIALISTE_REPO",
		"MEMORIALISTE_TOKEN_BUDGET", "MEMORIALISTE_DRY_RUN", "MEMORIALISTE_BRANCH_PREFIX",
		"MEMORIALISTE_AST_CONTEXT", "MEMORIALISTE_PLATFORM", "MEMORIALISTE_PLATFORM_URL",
		"MEMORIALISTE_PLATFORM_TOKEN", "MEMORIALISTE_PROJECT_ID", "MEMORIALISTE_BASE_BRANCH",
	}
	for _, e := range envs {
		if !strings.Contains(out, e) {
			t.Errorf("help output missing env var %q", e)
		}
	}

	groups := []string{"Provider", "Doc Structure", "Output", "Platform"}
	for _, g := range groups {
		if !strings.Contains(out, g) {
			t.Errorf("help output missing group header %q", g)
		}
	}
}
