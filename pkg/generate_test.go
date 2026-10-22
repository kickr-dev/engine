package engine_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	engine "github.com/kickr-dev/engine/pkg"
	"github.com/kickr-dev/engine/pkg/files"
)

func TestGenerate(t *testing.T) {
	ctx := t.Context()

	nooparser := func(context.Context, string, *testconfig) error { return nil }

	t.Run("error_open_root", func(t *testing.T) {
		// Arrange
		destdir := filepath.Join(t.TempDir(), "missing")

		// Act
		err := engine.Generate(ctx, destdir, testconfig{}, []engine.Parser[testconfig]{nooparser}, nil)

		// Assert
		assert.ErrorContains(t, err, "open root")
	})

	t.Run("error_parsing", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		parser := func(context.Context, string, *testconfig) error { return errors.New("some error") }

		// Act
		err := engine.Generate(ctx, destdir, testconfig{}, []engine.Parser[testconfig]{parser}, nil)

		// Assert
		assert.ErrorContains(t, err, "some error")
	})

	t.Run("success_concurrent_generators", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		one := func(_ context.Context, root *os.Root, _ testconfig) error {
			return root.WriteFile("one.txt", []byte("one"), files.RwRR)
		}
		two := func(_ context.Context, root *os.Root, _ testconfig) error {
			return root.WriteFile("two.txt", []byte("two"), files.RwRR)
		}
		failing := func(context.Context, *os.Root, testconfig) error { return errors.New("some error") }

		// Act
		err := engine.Generate(ctx, destdir, testconfig{},
			[]engine.Parser[testconfig]{nooparser},
			[]engine.Generator[testconfig]{one, two, failing})

		// Assert
		assert.ErrorIs(t, err, engine.ErrFailedGeneration)
		oneContent, err := os.ReadFile(filepath.Join(destdir, "one.txt"))
		require.NoError(t, err)
		assert.Equal(t, "one", string(oneContent))
		twoContent, err := os.ReadFile(filepath.Join(destdir, "two.txt"))
		require.NoError(t, err)
		assert.Equal(t, "two", string(twoContent))
	})
}
