package parser

import "strings"

const (
	// Bitbucket is just the bitbucket constant.
	Bitbucket = "bitbucket"
	// Gitea is just the gitea constant.
	Gitea = "gitea"
	// GitHub is just the github constant.
	GitHub = "github"
	// GitLab is just the gitlab constant.
	GitLab = "gitlab"
)

// VCS structure contains all properties related to VCS (Version Control System).
type VCS struct {
	// Name is the version control system name.
	Name string

	// Platform represents the vcs platform hosting the project.
	//
	// Self-hosted or aliased hosts that don't contain 'bitbucket', 'gitea', 'github', 'gitlab' or 'stash'
	// won't have their platform detected.
	// In such cases, override manually the attribute.
	Platform string

	// ProjectHost represents the host where the project is hosted.
	ProjectHost string

	// ProjectName is the project name being generated.
	ProjectName string

	// ProjectPath is the project path.
	ProjectPath string

	// Tags is the slice of repository tags.
	Tags []string
}

// parsePlatform returns the platform name associated to input host.
func parsePlatform(host string) (string, bool) {
	matchers := []struct {
		platform   string
		candidates []string
	}{
		{GitHub, []string{GitHub}},
		{GitLab, []string{GitLab}},
		{Gitea, []string{Gitea}},
		{Bitbucket, []string{Bitbucket, "stash"}},
	}
	for _, m := range matchers {
		for _, candidate := range m.candidates {
			if strings.Contains(host, candidate) {
				return m.platform, true
			}
		}
	}
	return "", false
}
