package parser //nolint:testpackage

import (
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kickr-dev/engine/testutils"
)

func TestGit(t *testing.T) {
	t.Run("error_no_git", func(t *testing.T) {
		// Act
		_, err := Git(t.TempDir())

		// Assert
		assert.ErrorContains(t, err, "open repository")
	})

	t.Run("success", func(t *testing.T) {
		// Arrange
		expected := VCS{
			ProjectHost: "github.com",
			ProjectName: "engine",
			ProjectPath: "kickr-dev/engine",
			Platform:    GitHub,
		}

		// Act
		vcs, err := Git(filepath.Join(testutils.Testdata(t), ".."))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expected, vcs)
	})
}

func TestGitOriginURL(t *testing.T) {
	t.Run("error_no_remote", func(t *testing.T) {
		// Arrange
		repository, err := git.Init(memory.NewStorage(), memfs.New())
		require.NoError(t, err)

		// Act
		_, err = gitOriginURL(repository)

		// Assert
		assert.ErrorContains(t, err, "get remote 'origin'")
	})

	t.Run("valid_git_repository", func(t *testing.T) {
		// Arrange
		repository, err := git.Init(memory.NewStorage(), memfs.New())
		require.NoError(t, err)
		_, err = repository.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{"https://github.com/kickr-dev/engine"}})
		require.NoError(t, err)

		// Act
		originURL, err := gitOriginURL(repository)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "https://github.com/kickr-dev/engine", originURL)
	})
}

func TestGitTags(t *testing.T) {
	t.Run("no_tags", func(t *testing.T) {
		// Arrange
		repository, err := git.Init(memory.NewStorage(), memfs.New())
		require.NoError(t, err)

		// Act
		tags, err := gitTags(repository)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, tags)
	})

	t.Run("has_tags", func(t *testing.T) {
		// Arrange
		repository, err := git.Init(memory.NewStorage(), memfs.New())
		require.NoError(t, err)
		worktree, err := repository.Worktree()
		require.NoError(t, err)
		hash, err := worktree.Commit("message", &git.CommitOptions{AllowEmptyCommits: true, Author: &object.Signature{}})
		require.NoError(t, err)
		_, err = repository.CreateTag("v0.1.0", hash, &git.CreateTagOptions{
			Message: "message",
			Tagger:  &object.Signature{},
		})
		require.NoError(t, err)
		_, err = repository.CreateTag("v1.0.0", hash, &git.CreateTagOptions{
			Message: "message",
			Tagger:  &object.Signature{},
		})
		require.NoError(t, err)

		// Act
		tags, err := gitTags(repository)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"v0.1.0", "v1.0.0"}, tags)
	})
}

func TestGitParseRemote(t *testing.T) {
	t.Run("empty_remote", func(t *testing.T) {
		// Act
		host, subpath := gitParseRemote("")

		// Assert
		assert.Empty(t, host)
		assert.Empty(t, subpath)
	})

	t.Run("success_no_port", func(t *testing.T) {
		// Arrange
		remotes := []string{
			"git@github.com:kickr-dev/engine.git",
			"http://github.com/kickr-dev/engine.git",
			"http://x-access-token:ghp_xxx@github.com/kickr-dev/engine.git",
			"https://github.com/kickr-dev/engine.git",
			"https://x-access-token:ghp_xxx@github.com/kickr-dev/engine.git",
			"ssh://git@github.com/kickr-dev/engine.git",
		}
		for _, remote := range remotes {
			t.Run(remote, func(t *testing.T) {
				// Act
				host, subpath := gitParseRemote(remote)

				// Assert
				assert.Equal(t, "github.com", host)
				assert.Equal(t, "kickr-dev/engine", subpath)
			})
		}
	})

	t.Run("success_port", func(t *testing.T) {
		// Arrange
		remotes := []string{
			"git@github.com:22/kickr-dev/engine.git",
			"http://github.com:22/kickr-dev/engine.git",
			"http://x-access-token:ghp_xxx@github.com:22/kickr-dev/engine.git",
			"https://github.com:22/kickr-dev/engine.git",
			"https://x-access-token:ghp_xxx@github.com:22/kickr-dev/engine.git",
			"ssh://git@github.com:22/kickr-dev/engine.git",
		}
		for _, remote := range remotes {
			t.Run(remote, func(t *testing.T) {
				// Act
				host, subpath := gitParseRemote(remote)

				// Assert
				assert.Equal(t, "github.com:22", host)
				assert.Equal(t, "kickr-dev/engine", subpath)
			})
		}
	})
}
