package generate

import (
	"context"
	"errors"

	"github.com/go-git/go-git/v5"
	craft "github.com/kilianpaquier/craft/pkg/craft/configuration"
	"github.com/kilianpaquier/craft/pkg/engine"
	"github.com/kilianpaquier/craft/pkg/engine/parser"
)

// ParserGit adds git configuration (if the current repository is a git repository)
// to the configuration.
func ParserGit(_ context.Context, destdir string, config *craft.Config) error {
	vcs, err := parser.Git(destdir)
	if err != nil {
		for _, is := range []error{git.ErrRepositoryNotExists, git.ErrRemoteNotFound} {
			if errors.Is(err, is) {
				engine.GetLogger().Warnf("failed to retrieve git vcs configuration: %v", err)
				return nil
			}
		}
		return err // not really an added value to wrap here
	}
	engine.GetLogger().Infof("git repository detected")

	config.VCS = vcs
	return nil
}

var _ engine.Parser[craft.Config] = ParserGit // ensure interface is implemented
