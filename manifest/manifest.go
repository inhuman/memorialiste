package manifest

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// DocEntry is one entry from the .docstructure.yaml manifest.
type DocEntry struct {
	Path        string   `yaml:"path"`
	Covers      []string `yaml:"covers"`
	Audience    string   `yaml:"audience,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Overrides   `yaml:",inline"`
}

// Manifest is the parsed content of a .docstructure.yaml file.
type Manifest struct {
	Defaults Overrides  `yaml:"defaults,omitempty"`
	Docs     []DocEntry `yaml:"docs"`
}

// ErrManifestNotFound is returned when the manifest file does not exist
// on disk. Callers can use errors.Is to detect this specific case and
// surface a user-friendly hint (e.g. "create docs/.docstructure.yaml or
// pass --doc-structure ...").
var ErrManifestNotFound = errors.New("manifest: file not found")

// Parse reads and validates the manifest at path.
// Returns ErrManifestNotFound when the file does not exist;
// other errors for malformed YAML or failed validation.
func Parse(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %q (create the file or pass --doc-structure with a different path)", ErrManifestNotFound, path)
		}
		return nil, fmt.Errorf("manifest: cannot read %q: %w", path, err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest %q: parse error: %w", path, err)
	}

	if len(m.Docs) == 0 {
		return nil, fmt.Errorf("manifest: no doc entries defined in %q", path)
	}

	if err := validateOverrides(path, "defaults", m.Defaults); err != nil {
		return nil, err
	}

	for i, entry := range m.Docs {
		if entry.Path == "" {
			return nil, fmt.Errorf("manifest: entry[%d].path is required", i)
		}
		if len(entry.Covers) == 0 {
			return nil, fmt.Errorf("manifest: entry[%d].covers must not be empty", i)
		}
		label := fmt.Sprintf("entry %q", entry.Path)
		if err := validateOverrides(path, label, entry.Overrides); err != nil {
			return nil, err
		}
	}

	return &m, nil
}

func validateOverrides(manifestPath, label string, o Overrides) error {
	if o.RepoMeta != "" && o.RepoMeta != "basic" && o.RepoMeta != "extended" {
		return fmt.Errorf("manifest %q: %s: repo_meta must be %q or %q (got %q)",
			manifestPath, label, "basic", "extended", o.RepoMeta)
	}
	if o.TokenBudget != nil && *o.TokenBudget <= 0 {
		return fmt.Errorf("manifest %q: %s: token_budget must be > 0 (got %d)",
			manifestPath, label, *o.TokenBudget)
	}
	if strings.HasPrefix(o.SystemPrompt, "@") {
		p := strings.TrimPrefix(o.SystemPrompt, "@")
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("manifest %q: %s: system_prompt file %q not found: %w",
				manifestPath, label, p, err)
		}
	}
	return nil
}
