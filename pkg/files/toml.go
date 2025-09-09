package files

import (
	"fmt"

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
