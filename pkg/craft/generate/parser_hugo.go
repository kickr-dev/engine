package generate

import (
	"context"

	craft "github.com/kilianpaquier/craft/pkg/craft/configuration"
	"github.com/kilianpaquier/craft/pkg/engine"
	"github.com/kilianpaquier/craft/pkg/engine/parser"
)

// ParserHugo detects the presence of a hugo.* or theme.*
// and adds a custom struct HugoConfig as 'hugo' key in config languages.
func ParserHugo(_ context.Context, destdir string, config *craft.Config) error {
	hugoconfig, ok := parser.Hugo(destdir)
	if !ok {
		return nil
	}
	engine.GetLogger().Infof("hugo detected, theme or hugo files are present")
	config.SetLanguage("hugo", hugoconfig)
	return nil
}

var _ engine.Parser[craft.Config] = ParserHugo // ensure interface is implemented
