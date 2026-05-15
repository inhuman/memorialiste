package manifest

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DocEntry is one entry from the .docstructure.yaml manifest.
type DocEntry struct {
	Path        string   `yaml:"path"`
	Covers      []string `yaml:"covers"`
	Audience    string   `yaml:"audience"`
	Description string   `yaml:"description"`
}

// Manifest is the parsed content of a .docstructure.yaml file.
type Manifest struct {
	Docs []DocEntry `yaml:"docs"`
}

type rawManifest struct {
	Docs []DocEntry `yaml:"docs"`
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

	var raw rawManifest
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("manifest: cannot parse %q: %w", path, err)
	}

	if len(raw.Docs) == 0 {
		return nil, fmt.Errorf("manifest: no doc entries defined in %q", path)
	}

	for i, entry := range raw.Docs {
		if entry.Path == "" {
			return nil, fmt.Errorf("manifest: entry[%d].path is required", i)
		}
		if len(entry.Covers) == 0 {
			return nil, fmt.Errorf("manifest: entry[%d].covers must not be empty", i)
		}
	}

	return &Manifest{Docs: raw.Docs}, nil
}
