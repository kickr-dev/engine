package parser //nolint:testpackage

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kickr-dev/engine/testutils"
)

func TestGit(t *testing.T) {
	t.Run("error_no_git", func(t *testing.T) {
		// Act
		_, err := Git(t.TempDir())

		// Assert
		assert.ErrorContains(t, err, "git origin URL")
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
	t.Run("error_no_git", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		// Act
		originURL, err := gitOriginURL(destdir)

		// Assert
		assert.ErrorContains(t, err, "open repository")
		assert.Empty(t, originURL)
	})

	t.Run("valid_git_repository", func(t *testing.T) {
		// Act
		originURL, err := gitOriginURL(filepath.Join(testutils.Testdata(t), ".."))

		// Assert
		require.NoError(t, err)
		assert.Contains(t, originURL, "kickr-dev/engine")
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

	t.Run("parse_ssh_remote", func(t *testing.T) {
		// Arrange
		rawRemote := "git@github.com:kickr-dev/engine.git"

		// Act
		host, subpath := gitParseRemote(rawRemote)

		// Assert
		assert.Equal(t, "github.com", host)
		assert.Equal(t, "kickr-dev/engine", subpath)
	})

	t.Run("parse_http_remote", func(t *testing.T) {
		// Arrange
		rawRemote := "https://github.com/kickr-dev/engine.git"

		// Act
		host, subpath := gitParseRemote(rawRemote)

		// Assert
		assert.Equal(t, "github.com", host)
		assert.Equal(t, "kickr-dev/engine", subpath)
	})
}
