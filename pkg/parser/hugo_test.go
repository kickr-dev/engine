package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kickr-dev/engine/pkg/files"
	"github.com/kickr-dev/engine/pkg/parser"
)

func TestHugoComposeModule(t *testing.T) {
	module := parser.HugoModule{HugoVersion: parser.HugoVersion{Max: "0.160.0", Min: "0.150.0"}}

	t.Run("success_config", func(t *testing.T) {
		// Arrange
		compose := parser.HugoCompose{HugoConfig: &parser.HugoConfig{Module: module}}

		// Act
		actual := compose.Module()

		// Assert
		assert.Equal(t, module, actual)
	})

	t.Run("success_theme", func(t *testing.T) {
		// Arrange
		compose := parser.HugoCompose{HugoTheme: &parser.HugoTheme{Module: module}}

		// Act
		actual := compose.Module()

		// Assert
		assert.Equal(t, module, actual)
	})

	t.Run("success_empty", func(t *testing.T) {
		// Act
		actual := parser.HugoCompose{}.Module()

		// Assert
		assert.Equal(t, parser.HugoModule{}, actual)
	})
}

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

		err := os.WriteFile(filepath.Join(destdir, "hugo.toml"), []byte("title = 'Hugo Title'\nname = 'Should not be there'"), files.RwRR)
		require.NoError(t, err)

		expected := parser.HugoCompose{HugoConfig: &parser.HugoConfig{Title: "Hugo Title"}}

		// Act
		config, err := parser.ReadHugo(destdir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expected, config)
	})

	t.Run("detected_hugo_publish_dir", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		err := os.WriteFile(filepath.Join(destdir, "hugo.toml"), []byte("publishDir = 'build'"), files.RwRR)
		require.NoError(t, err)

		expected := parser.HugoCompose{HugoConfig: &parser.HugoConfig{PublishDir: "build"}}

		// Act
		config, err := parser.ReadHugo(destdir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expected, config)
	})

	t.Run("detected_hugo_version", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		content := "[module.hugoVersion]\nmax = '0.160.0'\nmin = '0.150.0'"
		err := os.WriteFile(filepath.Join(destdir, "hugo.toml"), []byte(content), files.RwRR)
		require.NoError(t, err)

		expected := parser.HugoCompose{HugoConfig: &parser.HugoConfig{
			Module: parser.HugoModule{HugoVersion: parser.HugoVersion{Max: "0.160.0", Min: "0.150.0"}},
		}}

		// Act
		config, err := parser.ReadHugo(destdir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expected, config)
	})

	t.Run("detected_hugo_theme", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		err := os.WriteFile(filepath.Join(destdir, "theme.toml"), []byte("name = 'Theme Title'\ntitle = 'Should not be there'"), files.RwRR)
		require.NoError(t, err)

		expected := parser.HugoCompose{HugoTheme: &parser.HugoTheme{Name: "Theme Title"}}

		// Act
		config, err := parser.ReadHugo(destdir)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expected, config)
	})
}
