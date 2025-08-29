package files_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kickr-dev/engine/pkg/files"
)

func TestExists(t *testing.T) {
	t.Run("error_not_exists", func(t *testing.T) {
		// Act
		ok := files.Exists(filepath.Join(t.TempDir(), "invalid.txt"))

		// Assert
		assert.False(t, ok)
	})

	t.Run("success_exists", func(t *testing.T) {
		// Arrange
		dest := filepath.Join(t.TempDir(), "file.txt")
		file, err := os.Create(dest)
		require.NoError(t, err)
		require.NoError(t, file.Close())

		// Act
		ok := files.Exists(dest)

		// Assert
		assert.True(t, ok)
	})
}

func TestGlob(t *testing.T) {
	t.Run("no_dir", func(t *testing.T) {
		// Act
		matches := files.Glob(filepath.Join(t.TempDir(), "invalid"), "*.tmpl")

		// Assert
		assert.Empty(t, matches)
	})

	t.Run("no_glob", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		file, err := os.Create(filepath.Join(destdir, "file.txt"))
		require.NoError(t, err)
		require.NoError(t, file.Close())

		// Act
		matches := files.Glob(destdir, "*.tmpl")

		// Assert
		assert.Empty(t, matches)
	})

	t.Run("ignored_directories", func(t *testing.T) {
		for _, directory := range []string{"node_modules"} {
			t.Run(directory, func(t *testing.T) {
				// Arrange
				destdir := t.TempDir()
				target := filepath.Join(destdir, directory, "file.txt")

				require.NoError(t, os.MkdirAll(filepath.Dir(target), files.RwxRxRxRx))
				file, err := os.Create(target)
				require.NoError(t, err)
				require.NoError(t, file.Close())

				// Act
				matches := files.Glob(destdir, "*.txt", files.GlobExcludedDirectories(directory))

				// Assert
				assert.Empty(t, matches)
			})
		}
	})

	t.Run("glob", func(t *testing.T) {
		for _, filename := range []string{"template.tmpl", "template.yaml.tmpl", "template-part.json.tmpl"} {
			t.Run(filename, func(t *testing.T) {
				// Arrange
				destdir := t.TempDir()
				target := filepath.Join(destdir, filename)

				file, err := os.Create(target)
				require.NoError(t, err)
				require.NoError(t, file.Close())

				// Act
				matches := files.Glob(destdir, "*.tmpl")

				// Assert
				assert.Equal(t, []string{target}, matches)
			})
		}
	})

	t.Run("sub_glob", func(t *testing.T) {
		for _, filename := range []string{"template.tmpl", "template.yaml.tmpl", "template-part.json.tmpl"} {
			t.Run(filename, func(t *testing.T) {
				// Arrange
				destdir := t.TempDir()
				target := filepath.Join(destdir, "path", "to", "dir", filename)

				require.NoError(t, os.MkdirAll(filepath.Dir(target), files.RwxRxRxRx))
				file, err := os.Create(target)
				require.NoError(t, err)
				require.NoError(t, file.Close())

				// Act
				matches := files.Glob(destdir, "*.tmpl")

				// Assert
				assert.Equal(t, []string{target}, matches)
			})
		}
	})
}
