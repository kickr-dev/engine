package files

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ReadJSON reads the input src from fsys
// and unmarshal it with JSON format into the out configuration.
//
// Input src path must be relative to fsys (see io/fs.FS and io/fs.ValidPath), not absolute.
func ReadJSON(fsys fs.FS, src string, out any) error {
	content, err := fs.ReadFile(fsys, src)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	if err := json.Unmarshal(content, out); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

// ReadJSONFunc wraps ReadJSON as a delayed call, matching the func(out any) error signature expected by Validate.
//
// Input src path must be relative to fsys (see io/fs.FS and io/fs.ValidPath), not absolute.
func ReadJSONFunc(fsys fs.FS, src string) func(out any) error {
	return func(out any) error {
		return ReadJSON(fsys, src, out)
	}
}

// WriteJSON writes the input configuration into the dest in JSON format.
func WriteJSON(out string, data any) error {
	content, err := json.Marshal(data)
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
