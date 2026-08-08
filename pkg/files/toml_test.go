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

func TestReadTOML(t *testing.T) {
	t.Run("error_not_found", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()

		// Act
		var c testconfig
		err := files.ReadTOML(os.DirFS(dir), "file.toml", &c)

		// Assert
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("error_read", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, "file.toml"), files.RwxRxRxRx))

		// Act
		var c testconfig
		err := files.ReadTOML(os.DirFS(dir), "file.toml", &c)

		// Assert
		assert.ErrorContains(t, err, "read file")
	})

	t.Run("error_unmarshal", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file.toml"), []byte(`key == 'some value'`), files.RwRR))

		// Act
		var c testconfig
		err := files.ReadTOML(os.DirFS(dir), "file.toml", &c)

		// Assert
		assert.ErrorContains(t, err, "unmarshal")
	})

	t.Run("success", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		expected := testconfig{
			Slice:  []string{"value"},
			String: "value",
		}
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file.toml"), []byte("slice = [ 'value' ]\nstring = 'value'"), 0o644))

		// Act
		var actual testconfig
		err := files.ReadTOML(os.DirFS(dir), "file.toml", &actual)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}

func TestReadTOMLFunc(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		expected := testconfig{
			Slice:  []string{"value"},
			String: "value",
		}
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file.toml"), []byte("slice = [ 'value' ]\nstring = 'value'"), 0o644))

		// Act
		var actual testconfig
		err := files.ReadTOMLFunc(os.DirFS(dir), "file.toml")(&actual)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}

func TestWriteTOML(t *testing.T) {
	t.Run("error_open_file", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "file.toml")
		require.NoError(t, os.Mkdir(src, files.RwxRxRxRx))

		// Act
		err := files.WriteTOML(src, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "write file")
	})

	t.Run("success", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "dir", "file.toml")
		expected := testconfig{
			Slice:  []string{"value"},
			String: "value",
		}

		// Act
		require.NoError(t, files.WriteTOML(src, expected))

		// Assert
		var actual testconfig
		err := files.ReadTOML(os.DirFS(filepath.Dir(src)), filepath.Base(src), &actual)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
