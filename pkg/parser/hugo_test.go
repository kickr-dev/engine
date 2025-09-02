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
		_, ok := parser.Hugo(destdir)

		// Assert
		assert.False(t, ok)
	})

	t.Run("detected_hugo", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		hugo, err := os.Create(filepath.Join(destdir, "hugo.toml"))
		require.NoError(t, err)
		require.NoError(t, hugo.Close())

		expected := parser.HugoConfig{IsTheme: false}

		// Act
		config, ok := parser.Hugo(destdir)

		// Assert
		assert.True(t, ok)
		assert.Equal(t, expected, config)
	})

	t.Run("detected_hugo_theme", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		hugo, err := os.Create(filepath.Join(destdir, "theme.toml"))
		require.NoError(t, err)
		require.NoError(t, hugo.Close())

		expected := parser.HugoConfig{IsTheme: true}

		// Act
		config, ok := parser.Hugo(destdir)

		// Assert
		assert.True(t, ok)
		assert.Equal(t, expected, config)
	})
}
