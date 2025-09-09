package parser

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kickr-dev/engine/pkg/files"
)

// HugoCompose is the composition of both hugo.(toml|yaml|yml) configuration files
// and theme.(toml|yaml|yml) hugo theme configuration files.
//
// Note that when retrieving a HugoCompose with ReadHugo, either HugoTheme or HugoConfig will be provided but never both.
type HugoCompose struct {
	*HugoTheme
	*HugoConfig
}

// HugoConfig is the representation of a hugo.(toml|yaml|yml) configuration file.
type HugoConfig struct {
	BaseURL   string `yaml:"baseURL,omitempty"   toml:"baseurl,omitempty"`
	Copyright string `yaml:"copyright,omitempty" toml:"copyright,omitempty"`
	Title     string `yaml:"title,omitempty"     toml:"title,omitempty"`
}

// HugoTheme is the representation of a theme.(toml|yaml|yml) hugo theme configuration file.
type HugoTheme struct {
	DemoSite    string `yaml:"demosite,omitempty"    toml:"demosite,omitempty"`
	Description string `yaml:"description,omitempty" toml:"description,omitempty"`
	HomePage    string `yaml:"homepage,omitempty"    toml:"homepage,omitempty"`
	License     string `yaml:"license,omitempty"     toml:"license,omitempty"`
	Name        string `yaml:"name,omitempty"        toml:"name,omitempty"`
}

// ErrNoHugo is returned by ReadHugo when neither a hugo.(toml|yaml|yml) configuration file
// or a theme.(toml|yaml|yml) hugo theme configuration file is found.
//
// It's a convenient error to use with errors.Is.
var ErrNoHugo = errors.New("no hugo or hugo theme configuration found")

// ReadHugo detects if the project is a GoHugo project.
//
// Detection consists of reading hugo.(toml|yaml|yml)
// or theme.(toml|yaml|yml) files in the given destdir.
//
// The following checking order is made and the function will return on first success match:
//   - theme.toml, theme.yaml, theme.yml
//   - hugo.toml, hugo.yaml, hugo.yml
//
// It can be used as a simple function, calling it directly,
// but can also be used as its expected usage with engine.Generate:
//
//	type config struct { ... }
//
//	func ParserHugo(ctx context.Context, destdir string, c *config) error {
//		hugoc, err := parser.ReadHugo(destdir)
//		if err != nil {
//			if errors.Is(err, parser.ErrNoHugo) {
//				return nil
//			}
//			return fmt.Errorf("read hugo: %w", err)
//		}
//		engine.GetLogger().Infof("hugo detected, theme or hugo files are present")
//		// do something with hugo config (e.g. update config since it's a pointer)
//		return nil
//	}
func ReadHugo(destdir string) (HugoCompose, error) {
	type read struct {
		Name string
		Read func(src string, out any, read func(src string) ([]byte, error)) error
	}

	// read themes
	themes := []read{
		{Name: "theme.toml", Read: files.ReadTOML},
		{Name: "theme.yaml", Read: files.ReadYAML},
		{Name: "theme.yml", Read: files.ReadYAML},
	}
	for _, file := range themes {
		var config HugoTheme
		if err := file.Read(filepath.Join(destdir, file.Name), &config, os.ReadFile); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return HugoCompose{}, fmt.Errorf("read '%s': %w", file.Name, err)
		}
		return HugoCompose{HugoTheme: &config}, nil
	}

	// read configs
	configs := []read{
		{Name: "hugo.toml", Read: files.ReadTOML},
		{Name: "hugo.yaml", Read: files.ReadYAML},
		{Name: "hugo.yml", Read: files.ReadYAML},
	}
	for _, file := range configs {
		var config HugoConfig
		if err := file.Read(filepath.Join(destdir, file.Name), &config, os.ReadFile); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return HugoCompose{}, fmt.Errorf("read '%s': %w", file.Name, err)
		}
		return HugoCompose{HugoConfig: &config}, nil
	}

	return HugoCompose{}, ErrNoHugo
}
