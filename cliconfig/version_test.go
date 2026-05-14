package cliconfig

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

func TestVersion_PrintsStampedVersion(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })

	Version = "v1.2.3"

	var stdout, stderr bytes.Buffer
	exitCode := -1

	_, _ = parseWithOptions([]string{"--version"}, func(string) string { return "" },
		kong.Writers(&stdout, &stderr),
		kong.Exit(func(code int) { exitCode = code }),
	)

	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "memorialiste v1.2.3") {
		t.Errorf("expected version output to contain %q, got %q", "memorialiste v1.2.3", out)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestVersion_DefaultDev(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })
	Version = "dev"

	var stdout, stderr bytes.Buffer
	_, _ = parseWithOptions([]string{"--version"}, func(string) string { return "" },
		kong.Writers(&stdout, &stderr),
		kong.Exit(func(int) {}),
	)
	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "memorialiste dev") {
		t.Errorf("expected %q in output, got %q", "memorialiste dev", out)
	}
}
