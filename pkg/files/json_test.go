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

func TestReadJSON(t *testing.T) {
	t.Run("error_not_found", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()

		// Act
		var c testconfig
		err := files.ReadJSON(os.DirFS(dir), "file.json", &c)

		// Assert
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("error_read", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, "file.json"), files.RwxRxRxRx))

		// Act
		var c testconfig
		err := files.ReadJSON(os.DirFS(dir), "file.json", &c)

		// Assert
		assert.ErrorContains(t, err, "read file")
	})

	t.Run("error_unmarshal", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "file.json"), []byte(`{ "key":: "value" }`), files.RwRR))

		// Act
		var c testconfig
		err := files.ReadJSON(os.DirFS(dir), "file.json", &c)

		// Assert
		assert.ErrorContains(t, err, "unmarshal")
	})

	t.Run("success", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		src := filepath.Join(dir, "file.json")
		expected := testconfig{
			Slice:  []string{"value"},
			String: "value",
		}
		require.NoError(t, files.WriteJSON(src, expected))

		// Act
		var actual testconfig
		err := files.ReadJSON(os.DirFS(dir), "file.json", &actual)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}

func TestReadJSONFunc(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		src := filepath.Join(dir, "file.json")
		expected := testconfig{
			Slice:  []string{"value"},
			String: "value",
		}
		require.NoError(t, files.WriteJSON(src, expected))

		// Act
		var actual testconfig
		err := files.ReadJSONFunc(os.DirFS(dir), "file.json")(&actual)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}

func TestWriteJSON(t *testing.T) {
	t.Run("error_open_file", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "file.json")
		require.NoError(t, os.Mkdir(src, files.RwxRxRxRx))

		// Act
		err := files.WriteJSON(src, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "write file")
	})

	t.Run("success", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "dir", "file.json")
		expected := testconfig{
			Slice:  []string{"value"},
			String: "value",
		}

		// Act
		require.NoError(t, files.WriteJSON(src, expected))

		// Assert
		var actual testconfig
		err := files.ReadJSON(os.DirFS(filepath.Dir(src)), filepath.Base(src), &actual)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
