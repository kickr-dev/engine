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

	t.Run("error_read_template_out", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{Out: "dir"}
		require.NoError(t, os.Mkdir(filepath.Join(destdir, template.Out), files.RwxRxRxRx))

		// Act
		err := engine.Generate(ctx, destdir, testconfig{},
			[]engine.Parser[testconfig]{nooparser},
			[]engine.Generator[testconfig]{engine.GeneratorTemplates(os.DirFS(destdir), []engine.Template[testconfig]{template})})

		// Assert
		assert.ErrorIs(t, err, engine.ErrFailedGeneration)
	})

	t.Run("error_parse_template_globs", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs: []string{"invalid.txt"},
			Out:   "file.txt",
		}

		// Act
		err := engine.Generate(ctx, destdir, testconfig{},
			[]engine.Parser[testconfig]{nooparser},
			[]engine.Generator[testconfig]{engine.GeneratorTemplates(os.DirFS(destdir), []engine.Template[testconfig]{template})})

		// Assert
		assert.ErrorIs(t, err, engine.ErrFailedGeneration)
	})

	t.Run("success_template", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs: []string{"file.txt.tmpl"},
			Out:   "file.txt",
		}
		out := filepath.Join(destdir, template.Out)

		require.NoError(t, os.WriteFile(
			filepath.Join(destdir, template.Globs[0]),
			[]byte("value {{ .Str }} is empty, since no parser updated it"), files.RwRR))
		file, err := os.Create(out)
		require.NoError(t, err)
		require.NoError(t, file.Close())

		// Act
		err = engine.Generate(ctx, destdir, testconfig{},
			[]engine.Parser[testconfig]{nooparser},
			[]engine.Generator[testconfig]{engine.GeneratorTemplates(os.DirFS(destdir), []engine.Template[testconfig]{template})})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(out)
		require.NoError(t, err)
		assert.Equal(t, "value  is empty, since no parser updated it", string(content))
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
