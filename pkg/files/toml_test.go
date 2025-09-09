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
	t.Run("error_nil_read", func(t *testing.T) {
		// Act
		err := files.ReadTOML("", "", nil)

		// Assert
		assert.ErrorIs(t, err, files.ErrNilRead)
	})

	t.Run("error_not_found", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "file.toml")

		// Act
		var c testconfig
		err := files.ReadTOML(src, &c, os.ReadFile)

		// Assert
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("error_read", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "file.toml")
		require.NoError(t, os.Mkdir(src, files.RwxRxRxRx))

		// Act
		var c testconfig
		err := files.ReadTOML(filepath.Dir(src), &c, os.ReadFile)

		// Assert
		assert.ErrorContains(t, err, "read file")
	})

	t.Run("error_unmarshal", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "file.toml")
		require.NoError(t, os.WriteFile(src, []byte(`key == 'some value'`), files.RwRR))

		// Act
		var c testconfig
		err := files.ReadTOML(src, &c, os.ReadFile)

		// Assert
		assert.ErrorContains(t, err, "unmarshal")
	})

	t.Run("success", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "file.toml")
		expected := testconfig{
			Slice:  []string{"value"},
			String: "value",
		}
		require.NoError(t, os.WriteFile(src, []byte("slice = [ 'value' ]\nstring = 'value'"), 0o644))

		// Act
		var actual testconfig
		err := files.ReadTOML(src, &actual, os.ReadFile)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
