package watermarks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Record is one entry in the sidecar YAML.
type Record struct {
	Path        string `yaml:"path"`
	GeneratedAt string `yaml:"generated_at"`
}

// File represents the parsed sidecar contents.
type File struct {
	Records []Record
}

// Load parses the sidecar YAML at sidecarPath. Returns an empty *File (no
// error) when the file does not exist — the first-write path. Malformed
// YAML returns a wrapped error naming sidecarPath.
func Load(sidecarPath string) (*File, error) {
	data, err := os.ReadFile(sidecarPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &File{}, nil
		}
		return nil, fmt.Errorf("sidecar %q: read error: %w", sidecarPath, err)
	}
	if len(data) == 0 {
		return &File{}, nil
	}
	var recs []Record
	if err := yaml.Unmarshal(data, &recs); err != nil {
		return nil, fmt.Errorf("sidecar %q: parse error: %w", sidecarPath, err)
	}
	return &File{Records: recs}, nil
}

// Lookup returns generated_at for docPath, or ("", false) when absent.
func (f *File) Lookup(docPath string) (string, bool) {
	for _, r := range f.Records {
		if r.Path == docPath {
			return r.GeneratedAt, true
		}
	}
	return "", false
}

// Upsert inserts-or-updates the record for docPath. Existing records keep
// their position; new records are appended.
func (f *File) Upsert(docPath, generatedAt string) {
	for i := range f.Records {
		if f.Records[i].Path == docPath {
			f.Records[i].GeneratedAt = generatedAt
			return
		}
	}
	f.Records = append(f.Records, Record{Path: docPath, GeneratedAt: generatedAt})
}

// Save writes the sidecar back to sidecarPath, creating parent directories
// as needed.
func (f *File) Save(sidecarPath string) error {
	if err := os.MkdirAll(filepath.Dir(sidecarPath), 0o755); err != nil {
		return fmt.Errorf("sidecar %q: mkdir: %w", sidecarPath, err)
	}
	data, err := yaml.Marshal(f.Records)
	if err != nil {
		return fmt.Errorf("sidecar %q: marshal: %w", sidecarPath, err)
	}
	if err := os.WriteFile(sidecarPath, data, 0o644); err != nil {
		return fmt.Errorf("sidecar %q: write: %w", sidecarPath, err)
	}
	return nil
}
