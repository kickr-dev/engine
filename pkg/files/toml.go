package files

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"
)

// ReadTOML reads the input src from fsys
// and unmarshal with TOML format into the out configuration.
//
// Input src path must be relative to fsys (see io/fs.FS and io/fs.ValidPath), not absolute.
func ReadTOML(fsys fs.FS, src string, out any) error {
	content, err := fs.ReadFile(fsys, src)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	if err := toml.Unmarshal(content, out); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

// ReadTOMLFunc wraps ReadTOML as a delayed call, matching the func(out any) error signature expected by Validate.
//
// Input src path must be relative to fsys (see io/fs.FS and io/fs.ValidPath), not absolute.
func ReadTOMLFunc(fsys fs.FS, src string) func(out any) error {
	return func(out any) error {
		return ReadTOML(fsys, src, out)
	}
}

// WriteTOML writes the input configuration into the dest in TOML format.
func WriteTOML(out string, data any) error {
	content, err := toml.Marshal(data)
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
