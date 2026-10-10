package engine_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	engine "github.com/kickr-dev/engine/pkg"
	"github.com/kickr-dev/engine/pkg/files"
)

func TestApplySections(t *testing.T) {
	sections := []engine.Section{
		{Begin: "<!-- BEGIN_FIRST -->", End: "<!-- END_FIRST -->"},
		{Begin: "<!-- BEGIN_SECOND -->", End: "<!-- END_SECOND -->"},
	}

	t.Run("error_missing_out", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		// Act
		err := engine.ApplySections(os.DirFS(destdir), destdir, engine.Template[testconfig]{}, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "localize path")
	})

	t.Run("error_read_out", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{Out: "file.txt", Sections: sections}

		// Act
		err := engine.ApplySections(os.DirFS(srcdir), destdir, template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "read file")
		assert.NoFileExists(t, filepath.Join(destdir, template.Out))
	})

	t.Run("error_invalid_glob", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs:    []string{"["},
			Out:      "file.txt",
			Sections: sections,
		}
		file, err := os.Create(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		require.NoError(t, file.Close())

		// Act
		err = engine.ApplySections(os.DirFS(destdir), destdir, template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "parse template file(s)")
	})

	t.Run("error_parse_template", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs:    []string{"file.txt" + engine.TmplExtension},
			Out:      "file.txt",
			Sections: sections,
		}
		file, err := os.Create(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		require.NoError(t, file.Close())
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), []byte("{{ .Str"), files.RwRR))

		// Act
		err = engine.ApplySections(os.DirFS(srcdir), destdir, template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "parse template file(s)")
	})

	t.Run("error_execute_template", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs:    []string{"file.txt" + engine.TmplExtension},
			Out:      "file.txt",
			Sections: sections,
		}
		existing := "<!-- BEGIN_FIRST -->stale<!-- END_FIRST -->"
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), []byte("{{ .Unknown }}"), files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out), []byte(existing), files.RwRR))

		// Act
		err := engine.ApplySections(os.DirFS(srcdir), destdir, template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "template execution")
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, existing, string(content))
	})

	t.Run("success_existing_out", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs:    []string{"file.txt" + engine.TmplExtension},
			Out:      "file.txt",
			Sections: sections,
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]),
			[]byte(`<!-- BEGIN_FIRST -->{{ .Str }}<!-- END_FIRST --><!-- BEGIN_SECOND -->{{ .Str }}<!-- END_SECOND -->`),
			files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out),
			[]byte(`head {{ .Head }}<!-- BEGIN_FIRST -->stale<!-- END_FIRST -->middle {{ .Middle }}<!-- BEGIN_SECOND -->stale<!-- END_SECOND -->tail {{ .Tail }}`),
			files.RwRR))

		// Act
		err := engine.ApplySections(os.DirFS(srcdir), destdir, template, testconfig{Str: "value"})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		// only sections are rendered to avoid template injection
		assert.Equal(t, `head {{ .Head }}<!-- BEGIN_FIRST -->value<!-- END_FIRST -->middle {{ .Middle }}<!-- BEGIN_SECOND -->value<!-- END_SECOND -->tail {{ .Tail }}`, string(content))
	})

	t.Run("success_unchanged_out", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs:    []string{"file.txt" + engine.TmplExtension},
			Out:      "file.txt",
			Sections: sections,
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]),
			[]byte(`<!-- BEGIN_FIRST -->{{ .Str }}<!-- END_FIRST --><!-- BEGIN_SECOND -->{{ .Str }}<!-- END_SECOND -->`),
			files.RwRR))
		dest := filepath.Join(destdir, template.Out)
		require.NoError(t, os.WriteFile(dest,
			[]byte(`head<!-- BEGIN_FIRST -->value<!-- END_FIRST -->middle<!-- BEGIN_SECOND -->value<!-- END_SECOND -->tail`),
			files.RwRR))
		mtime := time.Now().Add(-time.Hour).Truncate(time.Second)
		require.NoError(t, os.Chtimes(dest, mtime, mtime))

		// Act
		err := engine.ApplySections(os.DirFS(srcdir), destdir, template, testconfig{Str: "value"})

		// Assert
		require.NoError(t, err)
		info, err := os.Stat(dest)
		require.NoError(t, err)
		assert.Equal(t, mtime, info.ModTime())
	})

	t.Run("success_partial_markers", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs:    []string{"file.txt" + engine.TmplExtension},
			Out:      "file.txt",
			Sections: sections,
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]),
			[]byte(`<!-- BEGIN_FIRST -->{{ .Str }}<!-- END_FIRST --><!-- BEGIN_SECOND -->{{ .Str }}<!-- END_SECOND -->`),
			files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out),
			[]byte("head <!-- BEGIN_FIRST -->stale<!-- END_FIRST --><!-- BEGIN_SECOND -->stale tail"),
			files.RwRR))

		// Act
		err := engine.ApplySections(os.DirFS(srcdir), destdir, template, testconfig{Str: "value"})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "head <!-- BEGIN_FIRST -->value<!-- END_FIRST --><!-- BEGIN_SECOND -->stale tail", string(content))
	})

	t.Run("success_section_not_rendered", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs:    []string{"file.txt" + engine.TmplExtension},
			Out:      "file.txt",
			Sections: sections,
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]),
			[]byte(`<!-- BEGIN_FIRST -->{{ .Str }}<!-- END_FIRST -->{{ if false }}<!-- BEGIN_SECOND -->{{ .Str }}<!-- END_SECOND -->{{ end }}`),
			files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out),
			[]byte("<!-- BEGIN_FIRST -->stale<!-- END_FIRST --><!-- BEGIN_SECOND -->stale<!-- END_SECOND -->"),
			files.RwRR))

		// Act
		err := engine.ApplySections(os.DirFS(srcdir), destdir, template, testconfig{Str: "value"})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "<!-- BEGIN_FIRST -->value<!-- END_FIRST --><!-- BEGIN_SECOND -->stale<!-- END_SECOND -->", string(content))
	})

	t.Run("success_section_half_rendered", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs:    []string{"file.txt" + engine.TmplExtension},
			Out:      "file.txt",
			Sections: sections,
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]),
			[]byte(`<!-- BEGIN_FIRST -->{{ .Str }}<!-- END_FIRST --><!-- BEGIN_SECOND -->{{ .Str }}`),
			files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out),
			[]byte("<!-- BEGIN_FIRST -->stale<!-- END_FIRST --><!-- BEGIN_SECOND -->stale<!-- END_SECOND -->"),
			files.RwRR))

		buf := strings.Builder{}
		initial := engine.GetLogger()
		engine.Configure(engine.WithLogger(engine.NewTestLogger(&buf)))
		t.Cleanup(func() { engine.Configure(engine.WithLogger(initial)) })

		// Act
		err := engine.ApplySections(os.DirFS(srcdir), destdir, template, testconfig{Str: "value"})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "<!-- BEGIN_FIRST -->value<!-- END_FIRST --><!-- BEGIN_SECOND -->stale<!-- END_SECOND -->", string(content))
		assert.Contains(t, buf.String(), "skipping section '<!-- BEGIN_SECOND -->' of 'file.txt', begin marker is rendered but not its end marker")
	})

	t.Run("success_markers_removed", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs:    []string{"file.txt" + engine.TmplExtension},
			Out:      "file.txt",
			Sections: sections,
		}
		existing := "head\ncustom content\n"
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]),
			[]byte(`<!-- BEGIN_FIRST -->{{ .Str }}<!-- END_FIRST --><!-- BEGIN_SECOND -->{{ .Str }}<!-- END_SECOND -->`),
			files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out), []byte(existing), files.RwRR))

		// Act
		err := engine.ApplySections(os.DirFS(srcdir), destdir, template, testconfig{Str: "value"})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, existing, string(content))
	})
}
