package files

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

// ReadYAML reads the input src from fsys
// and unmarshal it with YAML format into the out configuration.
//
// Input src path must be relative to fsys (see io/fs.FS and io/fs.ValidPath), not absolute.
func ReadYAML(fsys fs.FS, src string, out any) error {
	content, err := fs.ReadFile(fsys, src)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	if err := yaml.Unmarshal(content, out); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

// ReadYAMLFunc wraps ReadYAML as a delayed call, matching the func(out any) error signature expected by Validate.
//
// Input src path must be relative to fsys (see io/fs.FS and io/fs.ValidPath), not absolute.
func ReadYAMLFunc(fsys fs.FS, src string) func(out any) error {
	return func(out any) error {
		return ReadYAML(fsys, src, out)
	}
}

// WriteYAML writes the input configuration into the dest in YAML format.
func WriteYAML(out string, data any, opts ...yaml.EncodeOption) error {
	content, err := yaml.MarshalWithOptions(data, opts...)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(out), RwxRxRxRx); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(out, content, RwRR&^Umask()); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}
