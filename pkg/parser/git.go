package parser

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Git reads the input destdir directory remote.origin.url
// to retrieve various project information (git host, project name, etc.).
//
// It can be used as a simple function, calling it directly,
// but can also be used as its expected usage with engine.Generate:
//
//	type config struct { ... }
//
//	func ParserGit(ctx context.Context, destdir string, c *config) error {
//		vcs, err := parser.Git(destdir)
//		if err != nil {
//			engine.GetLogger().Warnf("failed to retrieve git vcs configuration: %v", err)
//			return nil // a repository may not be a git repository
//		}
//		engine.GetLogger().Infof("git repository detected")
//		// do something with vcs (e.g. update config since it's a pointer)
//		return nil
//	}
//
// Two errors may be checked with errors.Is when one is returned:
//   - git.ErrRepositoryNotExists
//   - git.ErrRemoteNotFound
func Git(destdir string) (VCS, error) {
	repository, err := git.PlainOpenWithOptions(destdir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return VCS{}, fmt.Errorf("open repository: %w", err)
	}

	rawRemote, err := gitOriginURL(repository)
	if err != nil {
		return VCS{}, fmt.Errorf("git origin URL: %w", err)
	}

	host, subpath := gitParseRemote(rawRemote)
	if host == "" || subpath == "" {
		return VCS{}, fmt.Errorf("invalid git remote URL '%s'", rawRemote)
	}

	tags, err := gitTags(repository)
	if err != nil {
		return VCS{}, fmt.Errorf("get tags: %w", err)
	}

	platform, _ := parsePlatform(host)
	return VCS{
		ProjectHost: host,
		ProjectName: path.Base(subpath),
		ProjectPath: subpath,
		Platform:    platform,
		Tags:        tags,
	}, nil
}

// gitOriginURL returns input directory remote origin by using go-git.
func gitOriginURL(repository *git.Repository) (string, error) {
	origin, err := repository.Remote("origin")
	if err != nil {
		return "", fmt.Errorf("get remote 'origin': %w", err)
	}
	if len(origin.Config().URLs) == 0 {
		return "", fmt.Errorf("no URL associated to remote: %w", git.ErrRemoteNotFound)
	}
	return origin.Config().URLs[0], nil
}

// gitParseRemote returns the current repository host and path to repository on the given host's platform.
func gitParseRemote(rawRemote string) (host, subpath string) {
	originURL := rawRemote
	if originURL == "" {
		return "", ""
	}

	originURL = strings.TrimSuffix(originURL, "\n")
	originURL = strings.TrimSuffix(originURL, ".git")

	// add ssh scheme to ssh based remotes
	if strings.HasPrefix(originURL, "git@") {
		originURL = "ssh://" + originURL
	}

	// try to parse originURL as a real URL, we should work in most cases (with port)
	u, err := url.Parse(originURL)
	if err == nil {
		return u.Host, u.Path[1:]
	}

	// an identified case is when there's a ':' with SSH remotes (git@github.com:kickr-dev/engine)
	// in this case we just replace ':' with a '/' and call the parsing again
	if strings.Contains(rawRemote, ":") {
		return gitParseRemote(strings.Replace(rawRemote, ":", "/", 1))
	}
	return "", ""
}

// gitTags returns the slice of known tags (locally) for the input destdir git repository.
func gitTags(repository *git.Repository) ([]string, error) {
	tags, err := repository.Tags()
	if err != nil {
		return nil, fmt.Errorf("get tags: %w", err)
	}
	defer tags.Close()

	var rawTags []string
	_ = tags.ForEach(func(r *plumbing.Reference) error {
		rawTags = append(rawTags, r.Name().Short())
		return nil
	})
	return rawTags, nil
}
