package files

import (
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

// ReadTOML reads the input src
// and unmarshal with TOML format into the out configuration.
func ReadTOML(src string, out any, read func(src string) ([]byte, error)) error {
	if read == nil {
		return ErrNilRead
	}

	content, err := read(src)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	if err := toml.Unmarshal(content, out); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

// WriteTOML writes the input configuration into the dest in TOML format.
func WriteTOML(out string, data any) error {
	content, err := toml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(out, content, RwRR); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}
