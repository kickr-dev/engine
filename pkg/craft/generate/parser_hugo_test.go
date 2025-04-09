package generate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	craft "github.com/kilianpaquier/craft/pkg/craft/configuration"
	"github.com/kilianpaquier/craft/pkg/craft/generate"
	"github.com/kilianpaquier/craft/pkg/engine/parser"
)

func TestParserHugo(t *testing.T) {
	ctx := t.Context()

	t.Run("success_hugo", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		hugoconfig, err := os.Create(filepath.Join(destdir, "hugo.toml"))
		require.NoError(t, err)
		require.NoError(t, hugoconfig.Close())

		expected := craft.Config{
			Languages: map[string]any{
				"hugo": parser.HugoConfig{},
			},
		}
		config := craft.Config{}

		// Act
		err = generate.ParserHugo(ctx, destdir, &config)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expected, config)
	})
}
