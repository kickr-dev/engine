package files_test

import (
	"os"
	"path/filepath"
	"strconv"
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
		// Arrange
		destdir := t.TempDir()
		targets := []string{
			filepath.Join(destdir, "node_modules", "file.txt"),
			filepath.Join(destdir, "subdir", "node_modules", "file.txt"),
		}
		for _, target := range targets {
			require.NoError(t, os.MkdirAll(filepath.Dir(target), files.RwxRxRxRx))
			file, err := os.Create(target)
			require.NoError(t, err)
			require.NoError(t, file.Close())
		}

		// Act
		matches := files.Glob(destdir, "*.txt", files.GlobExcludedDirectories("node_modules"))

		// Assert
		assert.Empty(t, matches)
	})

	t.Run("ignored_files", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		targets := []string{
			filepath.Join(destdir, "file.txt"),
			filepath.Join(destdir, "subdir", "file.txt"),
		}
		for _, target := range targets {
			require.NoError(t, os.MkdirAll(filepath.Dir(target), files.RwxRxRxRx))
			file, err := os.Create(target)
			require.NoError(t, err)
			require.NoError(t, file.Close())
		}

		// Act
		matches := files.Glob(destdir, "*.txt", files.GlobExcludedFiles("file.txt"))

		// Assert
		assert.Empty(t, matches)
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

	t.Run("success_nested_glob", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		targets := []string{
			filepath.Join(destdir, "sub", "root.tmpl"),
			filepath.Join(destdir, "path", "sub", "nested.tmpl"),
		}
		for _, target := range targets {
			require.NoError(t, os.MkdirAll(filepath.Dir(target), files.RwxRxRxRx))
			require.NoError(t, os.WriteFile(target, nil, files.RwRR))
		}
		require.NoError(t, os.WriteFile(filepath.Join(destdir, "path", "other.tmpl"), nil, files.RwRR))

		// Act
		matches := files.Glob(destdir, filepath.Join("sub", "*.tmpl"))

		// Assert
		assert.Equal(t, targets, matches)
	})
}

func BenchmarkGlob(b *testing.B) {
	// Arrange
	destdir := b.TempDir()
	for i := range 50 {
		dir := filepath.Join(destdir, strconv.Itoa(i%5), strconv.Itoa(i))
		require.NoError(b, os.MkdirAll(dir, files.RwxRxRxRx))
		for j := range 20 {
			require.NoError(b, os.WriteFile(filepath.Join(dir, strconv.Itoa(j)+".tmpl"), nil, files.RwRR))
		}
	}

	for b.Loop() {
		// Act
		_ = files.Glob(destdir, "*.tmpl",
			files.GlobExcludedDirectories(".git", "node_modules", "testdata"),
			files.GlobExcludedFiles("excluded.tmpl"))
	}
}
