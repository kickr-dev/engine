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
	Slice  []string `json:"slice,omitempty"  yaml:"slice,omitempty"`
	String string   `json:"string,omitempty" yaml:"string,omitempty"`
}

func TestReadYAML(t *testing.T) {
	t.Run("error_nil_read", func(t *testing.T) {
		// Act
		err := files.ReadYAML("", "", nil)

		// Assert
		assert.ErrorIs(t, err, files.ErrNilRead)
	})

	t.Run("error_not_found", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "file.yaml")

		// Act
		var c testconfig
		err := files.ReadYAML(src, &c, os.ReadFile)

		// Assert
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("error_read", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "file.yaml")
		require.NoError(t, os.Mkdir(src, files.RwxRxRxRx))

		// Act
		var c testconfig
		err := files.ReadYAML(filepath.Dir(src), &c, os.ReadFile)

		// Assert
		assert.ErrorContains(t, err, "read file")
	})

	t.Run("error_unmarshal", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "file.yaml")
		require.NoError(t, os.WriteFile(src, []byte(`{ "string":>> "value" }`), files.RwRR))

		// Act
		var c testconfig
		err := files.ReadYAML(src, &c, os.ReadFile)

		// Assert
		assert.ErrorContains(t, err, "unmarshal")
		assert.Zero(t, c)
	})

	t.Run("success", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "file.yaml")
		expected := testconfig{
			Slice:  []string{"value"},
			String: "value",
		}
		require.NoError(t, files.WriteYAML(src, expected))

		// Act
		var actual testconfig
		err := files.ReadYAML(src, &actual, os.ReadFile)

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
		src := filepath.Join(t.TempDir(), "file.yaml")
		expected := testconfig{
			Slice:  []string{"value"},
			String: "value",
		}

		// Act
		require.NoError(t, files.WriteYAML(src, expected))

		// Assert
		var actual testconfig
		err := files.ReadYAML(src, &actual, os.ReadFile)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
