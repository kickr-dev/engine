package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kickr-dev/engine/pkg/parser"
)

func TestHugo(t *testing.T) {
	t.Run("no_hugo", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		// Act
		_, err := parser.ReadHugo(destdir)

		// Assert
		assert.ErrorIs(t, err, parser.ErrNoHugo)
	})

	t.Run("detected_hugo", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		err := os.WriteFile(filepath.Join(destdir, "hugo.toml"), []byte("title = 'Hugo Title'\nname = 'Should not be there'"), 0o644)
		require.NoError(t, err)

		expected := parser.HugoCompose{HugoConfig: &parser.HugoConfig{Title: "Hugo Title"}}

		// Act
		config, err := parser.ReadHugo(destdir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expected, config)
	})

	t.Run("detected_hugo_theme", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		err := os.WriteFile(filepath.Join(destdir, "theme.toml"), []byte("name = 'Theme Title'\ntitle = 'Should not be there'"), 0o644)
		require.NoError(t, err)

		expected := parser.HugoCompose{HugoTheme: &parser.HugoTheme{Name: "Theme Title"}}

		// Act
		config, err := parser.ReadHugo(destdir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expected, config)
	})
}
