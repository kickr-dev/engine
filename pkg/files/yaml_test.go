package files_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kickr-dev/engine/pkg/files"
)

type testconfig struct {
	Slice  []string `json:"slice,omitempty"  yaml:"slice,omitempty"  toml:"slice,omitempty"`
	String string   `json:"string,omitempty" yaml:"string,omitempty" toml:"string,omitempty"`
}

func TestReadYAML(t *testing.T) {
	t.Run("error_not_found", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()

		// Act
		var c testconfig
		err := files.ReadYAML(os.DirFS(dir), "file.yaml", &c)

		// Assert
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("error_read", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, "file.yaml"), files.RwxRxRxRx))

		// Act
		var c testconfig
		err := files.ReadYAML(os.DirFS(dir), "file.yaml", &c)

		// Assert
		assert.ErrorContains(t, err, "read file")
	})

	t.Run("error_unmarshal", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file.yaml"), []byte(`{ "string":>> "value" }`), files.RwRR))

		// Act
		var c testconfig
		err := files.ReadYAML(os.DirFS(dir), "file.yaml", &c)

		// Assert
		assert.ErrorContains(t, err, "unmarshal")
		assert.Zero(t, c)
	})

	t.Run("success", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		src := filepath.Join(dir, "file.yaml")
		expected := testconfig{
			Slice:  []string{"value"},
			String: "value",
		}
		require.NoError(t, files.WriteYAML(src, expected))

		// Act
		var actual testconfig
		err := files.ReadYAML(os.DirFS(dir), "file.yaml", &actual)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}

func TestReadYAMLFunc(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		src := filepath.Join(dir, "file.yaml")
		expected := testconfig{
			Slice:  []string{"value"},
			String: "value",
		}
		require.NoError(t, files.WriteYAML(src, expected))

		// Act
		var actual testconfig
		err := files.ReadYAMLFunc(os.DirFS(dir), "file.yaml")(&actual)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}

func TestWriteYAML(t *testing.T) {
	t.Run("error_open_file", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "file.yaml")
		require.NoError(t, os.Mkdir(src, files.RwxRxRxRx))

		// Act
		err := files.WriteYAML(src, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "write file")
	})

	t.Run("success", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "dir", "file.yaml")
		expected := testconfig{
			Slice:  []string{"value"},
			String: "value",
		}

		// Act
		require.NoError(t, files.WriteYAML(src, expected))

		// Assert
		var actual testconfig
		err := files.ReadYAML(os.DirFS(filepath.Dir(src)), filepath.Base(src), &actual)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
