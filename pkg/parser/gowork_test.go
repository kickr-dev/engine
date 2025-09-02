package parser_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kickr-dev/engine/pkg/files"
	"github.com/kickr-dev/engine/pkg/parser"
)

func TestReadGowork(t *testing.T) {
	t.Run("no_gowork", func(t *testing.T) {
		// Act
		_, err := parser.ReadGowork("")

		// Assert
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.ErrorIs(t, err, parser.ErrNoGowork)
	})

	t.Run("invalid_gowork", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		gowork := filepath.Join(destdir, parser.FileGowork)
		err := os.WriteFile(gowork, []byte("an invalid go.work file"), files.RwRR)
		require.NoError(t, err)

		// Act
		_, err = parser.ReadGowork(destdir)

		// Assert
		assert.ErrorContains(t, err, "parse modfile")
	})

	t.Run("missing_go_statement", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		err := os.WriteFile(filepath.Join(destdir, parser.FileGowork), []byte(""), files.RwRR)
		require.NoError(t, err)

		// Act
		_, err = parser.ReadGowork(destdir)

		// Assert
		assert.ErrorIs(t, err, parser.ErrMissingGoStatement)
	})

	t.Run("invalid_use_gomod", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		err := os.WriteFile(filepath.Join(destdir, parser.FileGowork), []byte(
			`go 1.22
			toolchain go1.23.5
			use (
				./lib1
				./lib2
			)`,
		), files.RwRR)
		require.NoError(t, err)

		// Act
		_, err = parser.ReadGowork(destdir)

		// Assert
		require.ErrorContains(t, err, "read gomod in './lib1'")
		require.ErrorContains(t, err, "read gomod in './lib2'")
	})

	t.Run("golang_detected", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		err := os.WriteFile(filepath.Join(destdir, parser.FileGowork), []byte(
			`go 1.22
			toolchain go1.23.5
			use (
				./lib1
				./lib2
			)`,
		), files.RwRR)
		require.NoError(t, err)

		require.NoError(t, os.MkdirAll(filepath.Join(destdir, "lib1"), files.RwxRxRxRx))
		err = os.WriteFile(filepath.Join(destdir, "lib1", parser.FileGomod), []byte("module github.com/kickr-dev/engine/lib1\ngo 1.22\ntoolchain go1.23.5"), files.RwRR)
		require.NoError(t, err)

		require.NoError(t, os.MkdirAll(filepath.Join(destdir, "lib2"), files.RwxRxRxRx))
		err = os.WriteFile(filepath.Join(destdir, "lib2", parser.FileGomod), []byte("module github.com/kickr-dev/engine/lib2\ngo 1.23\ntoolchain go1.25.0"), files.RwRR)
		require.NoError(t, err)

		expectedMod := parser.Gowork{
			Go:        "1.22",
			Toolchain: "1.23.5",
			Uses: []parser.GoworkUse{
				{
					Gomod: parser.Gomod{
						Go:        "1.22",
						Module:    "github.com/kickr-dev/engine/lib1",
						Toolchain: "1.23.5",
						Tools:     []string{},
					},
					ModulePath: "",
					Use:        "./lib1",
				},
				{
					Gomod: parser.Gomod{
						Go:        "1.23",
						Module:    "github.com/kickr-dev/engine/lib2",
						Toolchain: "1.25.0",
						Tools:     []string{},
					},
					ModulePath: "",
					Use:        "./lib2",
				},
			},
		}

		// Act
		mod, err := parser.ReadGowork(destdir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expectedMod, mod)
	})
}
