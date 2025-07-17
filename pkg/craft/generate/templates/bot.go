package templates

import (
	"path"
	"slices"

	craft "github.com/kilianpaquier/craft/pkg/craft/configuration"
	"github.com/kilianpaquier/craft/pkg/engine"
	"github.com/kilianpaquier/craft/pkg/engine/parser"
)

// Dependabot returns the slice of templates related to dependabot configuration.
func Dependabot() []engine.Template[craft.Config] {
	return []engine.Template[craft.Config]{
		{
			Delimiters: engine.DelimitersBracket(),
			Globs:      []string{path.Join(".github", "dependabot.yml"+engine.TmplExtension)},
			Out:        path.Join(".github", "dependabot.yml"),
			Remove: func(config craft.Config) bool {
				return config.Bot != craft.Dependabot || config.Platform != parser.GitHub
			},
		},
		{
			Delimiters: engine.DelimitersBracket(),
			Globs:      []string{path.Join(".gitlab", "dependabot.yml"+engine.TmplExtension)},
			Out:        path.Join(".gitlab", "dependabot.yml"),
			Remove: func(config craft.Config) bool {
				return config.Bot != craft.Dependabot || !config.IsCI(parser.GitLab)
			},
		},
	}
}

// Renovate returns the slice of templates related to renovate configuration.
func Renovate() []engine.Template[craft.Config] {
	return []engine.Template[craft.Config]{
		{
			Delimiters: engine.DelimitersChevron(),
			Globs:      []string{path.Join(".github", "workflows", "renovate.yml"+engine.TmplExtension)},
			Out:        path.Join(".github", "workflows", "renovate.yml"),
			Remove: func(config craft.Config) bool {
				return config.Bot != craft.Renovate || !config.IsCI(parser.GitHub)
			},
		},
		{
			Delimiters: engine.DelimitersBracket(),
			Globs:      []string{path.Join("scripts", "sh", "renovate.sh"+engine.TmplExtension)},
			Out:        path.Join("scripts", "sh", "renovate.sh"),
			Remove: func(config craft.Config) bool {
				return config.Bot != craft.Renovate || !slices.Contains(config.Include, craft.RenovatePostUpgrade)
			},
		},
		{
			Delimiters: engine.DelimitersBracket(),
			Globs:      []string{"renovate.json5" + engine.TmplExtension},
			Out:        "renovate.json5",
			Remove:     func(config craft.Config) bool { return config.Bot != craft.Renovate },
		},
	}
}
