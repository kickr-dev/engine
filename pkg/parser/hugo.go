package parser

import "path/filepath"

// HugoConfig represents the parse'd hugo.* or theme.* file associated to hugo configuration file.
type HugoConfig struct {
	// IsTheme expresses whether a theme.* configuration is present,
	// meaning current hugo repository is a theme one.
	IsTheme bool
}

// Hugo detects if the project is a Hugo project.
//
// Detection consists of looking for hugo.* or theme.* files in the given destdir.
//
// It can be used as a simple function, calling it directly,
// but can also be used as its expected usage with engine.Generate:
//
//	type config struct { ... }
//
//	func ParserHugo(ctx context.Context, destdir string, c *config) error {
//		hugoc, ok := parser.Hugo(destdir)
//		if !ok {
//			return nil
//		}
//		engine.GetLogger().Infof("hugo detected, theme or hugo files are present")
//		// do something with hugo config (e.g. update config since it's a pointer)
//		return nil
//	}
func Hugo(destdir string) (HugoConfig, bool) {
	// detect hugo project
	configs, _ := filepath.Glob(filepath.Join(destdir, "hugo.*"))

	// detect hugo theme
	themes, _ := filepath.Glob(filepath.Join(destdir, "theme.*"))

	if len(configs) == 0 && len(themes) == 0 {
		return HugoConfig{}, false
	}
	return HugoConfig{IsTheme: len(themes) > 0}, true
}
