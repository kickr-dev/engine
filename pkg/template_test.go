package engine_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	engine "gitlab.com/kickr-dev/engine/pkg"
	"gitlab.com/kickr-dev/engine/pkg/files"
	"gitlab.com/kickr-dev/engine/testutils"
)

func TestApplyTemplate(t *testing.T) {
	t.Run("error_symlink_remove_outside_root", func(t *testing.T) {
		// Arrange
		tmp := t.TempDir()
		destdir := filepath.Join(tmp, "dest")
		require.NoError(t, os.Mkdir(destdir, files.RwxRxRxRx))
		outside := filepath.Join(tmp, "outside", "sub")
		require.NoError(t, os.MkdirAll(outside, files.RwxRxRxRx))
		require.NoError(t, os.Symlink(filepath.Dir(outside), filepath.Join(destdir, "chart")))

		template := engine.Template[testconfig]{
			GeneratePolicy: engine.StaticPolicy[testconfig](engine.PolicyRemove),
			Out:            "chart/sub",
		}

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "lstat")
		assert.DirExists(t, outside)
	})

	t.Run("error_remove", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("running as root bypasses permission checks")
		}

		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			GeneratePolicy: engine.StaticPolicy[testconfig](engine.PolicyRemove),
			Out:            "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out), []byte("content"), files.RwRR))
		require.NoError(t, os.Chmod(destdir, 0o500))
		t.Cleanup(func() { _ = os.Chmod(destdir, files.RwxRxRxRx) })

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "remove")
	})

	t.Run("error_read_template_out", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{Out: "dir"}
		require.NoError(t, os.Mkdir(filepath.Join(destdir, template.Out), files.RwxRxRxRx))

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "is generated")
	})

	t.Run("error_invalid_policy", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			GeneratePolicy: engine.StaticPolicy[testconfig](engine.GeneratePolicy(42)),
			Out:            "file.txt",
		}

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "invalid generate policy")
	})

	t.Run("error_template_invalid_globs", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{Out: "file.txt"}

		buf := strings.Builder{}
		initial := engine.GetLogger()
		engine.SetLogger(engine.NewTestLogger(&buf))
		t.Cleanup(func() { engine.SetLogger(initial) })

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, buf.String(), fmt.Sprintf("empty template 'globs', skipping '%s' generation", template.Out))
	})

	t.Run("error_parse_template_globs", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs: []string{"invalid.txt"},
			Out:   "file.txt",
		}

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "parse template file(s)")
	})

	t.Run("error_symlink_outside_root", func(t *testing.T) {
		// Arrange
		tmp := t.TempDir()
		srcdir := t.TempDir()
		destdir := filepath.Join(tmp, "dest")
		require.NoError(t, os.Mkdir(destdir, files.RwxRxRxRx))
		require.NoError(t, os.Symlink(filepath.Join(tmp, "outside.txt"), filepath.Join(destdir, "file.txt")))

		template := engine.Template[testconfig]{
			Globs: []string{"file.txt" + engine.TmplExtension},
			Out:   "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), []byte("content"), files.RwRR))

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "path escapes from parent")
		assert.NoFileExists(t, filepath.Join(tmp, "outside.txt"))
	})

	t.Run("success_custom_func", func(t *testing.T) {
		// Arrange
		ctx := engine.ContextWithOptions(t.Context(), engine.Options{Funcs: template.FuncMap{"ping": func() string { return "pong" }}})

		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs: []string{"file.txt" + engine.TmplExtension},
			Out:   "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), []byte("{{ ping }}"), files.RwRR))

		// Act
		err := engine.ApplyTemplate(ctx, os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "pong", string(content))
	})

	t.Run("success_sections_existing_file", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs:    []string{"file.txt" + engine.TmplExtension},
			Out:      "file.txt",
			Sections: []engine.Section{{Begin: "<!-- BEGIN -->", End: "<!-- END -->"}},
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]),
			[]byte(`<!-- BEGIN -->{{ .Str }}<!-- END -->`),
			files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out),
			[]byte("custom <!-- BEGIN -->stale<!-- END --> custom"),
			files.RwRR))

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{Str: "value"})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "custom <!-- BEGIN -->value<!-- END --> custom", string(content))
	})

	t.Run("success_policy_always", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			GeneratePolicy: engine.StaticPolicy[testconfig](engine.PolicyAlways),
			Globs:          []string{"file.txt" + engine.TmplExtension},
			Out:            "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), []byte("generated"), files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out), []byte("live"), files.RwRR))

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "generated", string(content))
	})

	t.Run("success_policy_default_forced", func(t *testing.T) {
		// Arrange
		ctx := engine.ContextWithOptions(t.Context(), engine.Options{Force: true})

		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs: []string{"file.txt" + engine.TmplExtension},
			Out:   "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), []byte("generated"), files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out), []byte("live"), files.RwRR))

		// Act
		err := engine.ApplyTemplate(ctx, os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "generated", string(content))
	})

	t.Run("success_policy_skip", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			GeneratePolicy: engine.StaticPolicy[testconfig](engine.PolicySkip),
			Globs:          []string{"file.txt" + engine.TmplExtension},
			Out:            "file.txt",
			Patches:        []string{"missing.patch"},
			Sections:       []engine.Section{{Begin: "<!-- BEGIN -->", End: "<!-- END -->"}},
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]),
			[]byte(`<!-- BEGIN -->{{ .Str }}<!-- END -->`),
			files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out),
			[]byte("live <!-- BEGIN -->stale<!-- END --> live"),
			files.RwRR))

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{Str: "value"})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "live <!-- BEGIN -->stale<!-- END --> live", string(content))
	})

	t.Run("success_policy_skip_forced", func(t *testing.T) {
		// Arrange
		ctx := engine.ContextWithOptions(t.Context(), engine.Options{Force: true})

		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			GeneratePolicy: engine.StaticPolicy[testconfig](engine.PolicySkip),
			Globs:          []string{"file.txt" + engine.TmplExtension},
			Out:            "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), []byte("generated"), files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out), []byte("live"), files.RwRR))

		// Act
		err := engine.ApplyTemplate(ctx, os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "live", string(content))
	})

	t.Run("success_policy_remove_forced", func(t *testing.T) {
		// Arrange
		ctx := engine.ContextWithOptions(t.Context(), engine.Options{Force: true})

		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			GeneratePolicy: engine.StaticPolicy[testconfig](engine.PolicyRemove),
			Out:            "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out), []byte("live"), files.RwRR))

		// Act
		err := engine.ApplyTemplate(ctx, os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		assert.NoFileExists(t, filepath.Join(destdir, template.Out))
	})

	t.Run("success_policy_func_inputs", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		var gotPath string
		var gotConfig testconfig
		template := engine.Template[testconfig]{
			GeneratePolicy: func(path string, config testconfig) engine.GeneratePolicy {
				gotPath, gotConfig = path, config
				return engine.PolicySkip
			},
			Out: "file.txt",
		}

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{Str: "value"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(destdir, template.Out), gotPath)
		assert.Equal(t, testconfig{Str: "value"}, gotConfig)
	})

	t.Run("error_mkdir", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			GeneratePolicy: engine.StaticPolicy[testconfig](engine.PolicyAlways),
			Globs:          []string{"file.txt" + engine.TmplExtension},
			Out:            filepath.Join("file.txt", "file.txt"),
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), []byte("content"), files.RwRR))

		// create a file at midlevel to ensure mkdir fails
		require.NoError(t, os.WriteFile(filepath.Join(destdir, "file.txt"), nil, files.RwRR))

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "mkdir")
	})

	t.Run("error_execute", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs: []string{"file.txt" + engine.TmplExtension},
			Out:   "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), []byte("{{ .Unknown }}"), files.RwRR))

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "template execution")
	})

	t.Run("error_write_dir", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			GeneratePolicy: engine.StaticPolicy[testconfig](engine.PolicyAlways),
			Globs:          []string{"file.txt" + engine.TmplExtension},
			Out:            "dir",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), []byte("{{ .Str }}"), files.RwRR))

		// create a directory at out to ensure the file opening fails
		require.NoError(t, os.Mkdir(filepath.Join(destdir, template.Out), files.RwxRxRxRx))

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{Str: "value"})

		// Assert
		assert.ErrorContains(t, err, "open file")
	})

	t.Run("success_keep_notice", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs: []string{"file.txt" + engine.TmplExtension},
			Out:   "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]),
			[]byte("# Code generated by engine; DO NOT EDIT.\n\ncontent\n"),
			files.RwRR))

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "# Code generated by engine; DO NOT EDIT.\n\ncontent\n", string(content))
	})

	t.Run("success_strip_notice", func(t *testing.T) {
		// Arrange
		ctx := engine.ContextWithOptions(t.Context(), engine.Options{StripNotice: true})

		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs: []string{"file.txt" + engine.TmplExtension},
			Out:   "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]),
			[]byte("# Code generated by engine; DO NOT EDIT.\n\ncontent\n"),
			files.RwRR))

		// Act
		err := engine.ApplyTemplate(ctx, os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "content\n", string(content))
	})

	t.Run("success_unchanged_content", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			GeneratePolicy: engine.StaticPolicy[testconfig](engine.PolicyAlways),
			Globs:          []string{"file.txt" + engine.TmplExtension},
			Out:            "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), []byte("content"), files.RwRR))

		dest := filepath.Join(destdir, template.Out)
		require.NoError(t, os.WriteFile(dest, []byte("content"), files.Rw))
		mtime := time.Now().Add(-time.Hour).Truncate(time.Second)
		require.NoError(t, os.Chtimes(dest, mtime, mtime))

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		info, err := os.Stat(dest)
		require.NoError(t, err)
		assert.Equal(t, mtime, info.ModTime())
		assert.Equal(t, files.RwRR&^files.Umask(), info.Mode())
	})

	t.Run("success_mode_default", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs: []string{"file.txt" + engine.TmplExtension},
			Out:   "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), []byte("content"), files.RwRR))

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		info, err := os.Stat(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, files.RwRR&^files.Umask(), info.Mode())
	})

	t.Run("success_mode_explicit", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			EmptyPolicy: engine.PolicyKeepEmpty,
			Globs:       []string{"script.sh" + engine.TmplExtension},
			Mode:        files.RwxRxRxRx,
			Out:         "script.sh",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), []byte("#!/bin/..."), files.RwRR))

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		info, err := os.Stat(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, files.RwxRxRxRx&^files.Umask(), info.Mode())
	})

	t.Run("success_skip_empty", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			EmptyPolicy: engine.PolicyRemoveEmpty,
			Globs:       []string{"file.txt" + engine.TmplExtension},
			Out:         filepath.Join("dir", "file.txt"),
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), nil, files.RwRR))

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		assert.NoDirExists(t, filepath.Join(destdir, "dir"))
		assert.NoFileExists(t, filepath.Join(destdir, template.Out))
	})

	t.Run("success_keep_empty", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			EmptyPolicy: engine.PolicyKeepEmpty,
			Globs:       []string{"file.txt" + engine.TmplExtension},
			Out:         "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), nil, files.RwRR))

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Empty(t, content)
	})

	t.Run("error_remove_existing_empty", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("running as root bypasses permission checks")
		}

		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			EmptyPolicy:    engine.PolicyRemoveEmpty,
			GeneratePolicy: engine.StaticPolicy[testconfig](engine.PolicyAlways),
			Globs:          []string{"file.txt" + engine.TmplExtension},
			Out:            "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), nil, files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out), []byte("some not empty file"), files.RwRR))
		require.NoError(t, os.Chmod(destdir, 0o500))
		t.Cleanup(func() { _ = os.Chmod(destdir, files.RwxRxRxRx) })

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "remove")
	})

	t.Run("success_remove_existing_empty", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			EmptyPolicy:    engine.PolicyRemoveEmpty,
			GeneratePolicy: engine.StaticPolicy[testconfig](engine.PolicyAlways),
			Globs:          []string{"file.txt" + engine.TmplExtension},
			Out:            "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), nil, files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out), []byte("some not empty file"), files.RwRR))

		// Act
		err := engine.ApplyTemplate(t.Context(), os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		assert.NoFileExists(t, filepath.Join(destdir, template.Out))
	})
}

func TestGeneratorTemplates(t *testing.T) {
	ctx := t.Context()

	t.Run("error_parse_template_globs", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		generator := engine.GeneratorTemplates(os.DirFS(destdir),
			[]engine.Template[testconfig]{{Globs: []string{"invalid.txt"}, Out: "file.txt"}})

		// Act
		err := generator(ctx, testutils.Root(t, destdir), testconfig{})

		// Assert
		assert.ErrorIs(t, err, engine.ErrFailedGeneration)
	})

	t.Run("success", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs: []string{"file.txt" + engine.TmplExtension},
			Out:   "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Globs[0]), []byte("{{ .Str }}"), files.RwRR))

		generator := engine.GeneratorTemplates(os.DirFS(destdir), []engine.Template[testconfig]{template})

		// Act
		err := generator(ctx, testutils.Root(t, destdir), testconfig{Str: "value"})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "value", string(content))
	})
}

type testmodule struct {
	directory string
}

func (m testmodule) Dir() string { return m.directory }

func TestGeneratorModules(t *testing.T) {
	ctx := t.Context()

	modules := func(config []testmodule) []testmodule { return config }

	t.Run("error_parse_template_globs", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		generator := engine.GeneratorModules(os.DirFS(destdir), modules,
			[]engine.Template[testmodule]{{Globs: []string{"invalid.txt"}, Out: "file.txt"}})

		// Act
		err := generator(ctx, testutils.Root(t, destdir), []testmodule{{directory: "."}})

		// Assert
		assert.ErrorIs(t, err, engine.ErrFailedGeneration)
	})

	t.Run("error_module_outside_destdir", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := filepath.Join(t.TempDir(), "dest")
		require.NoError(t, os.Mkdir(destdir, files.RwxRxRxRx))
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, "file.txt"+engine.TmplExtension), []byte("{{ .Dir }}"), files.RwRR))

		generator := engine.GeneratorModules(os.DirFS(srcdir), modules,
			[]engine.Template[testmodule]{{Globs: []string{"file.txt" + engine.TmplExtension}, Out: "file.txt"}})

		// Act
		err := generator(ctx, testutils.Root(t, destdir), []testmodule{{directory: "../outside"}})

		// Assert
		assert.ErrorIs(t, err, engine.ErrFailedGeneration)
		assert.NoFileExists(t, filepath.Join(destdir, "..", "outside", "file.txt"))
	})

	t.Run("success_no_module", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		generator := engine.GeneratorModules(os.DirFS(destdir), modules,
			[]engine.Template[testmodule]{{Globs: []string{"invalid.txt"}, Out: "file.txt"}})

		// Act
		err := generator(ctx, testutils.Root(t, destdir), nil)

		// Assert
		require.NoError(t, err)
		assert.NoFileExists(t, filepath.Join(destdir, "file.txt"))
	})

	t.Run("success_multiple_modules", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, "file.txt"+engine.TmplExtension), []byte("{{ .Dir }}"), files.RwRR))

		generator := engine.GeneratorModules(os.DirFS(srcdir), modules,
			[]engine.Template[testmodule]{{Globs: []string{"file.txt" + engine.TmplExtension}, Out: "file.txt"}})

		require.NoError(t, os.MkdirAll(filepath.Join(destdir, "apps", "api"), files.RwxRxRxRx))

		// Act
		err := generator(ctx, testutils.Root(t, destdir), []testmodule{{directory: "."}, {directory: "apps/api"}})

		// Assert
		require.NoError(t, err)
		for _, dir := range []string{".", "apps/api"} {
			content, err := os.ReadFile(filepath.Join(destdir, dir, "file.txt"))
			require.NoError(t, err)
			assert.Equal(t, dir, string(content))
		}
	})
}

func BenchmarkApplyTemplate(b *testing.B) {
	// Arrange
	srcdir := b.TempDir()
	destdir := b.TempDir()
	tmpl := engine.Template[testconfig]{
		GeneratePolicy: engine.StaticPolicy[testconfig](engine.PolicyAlways),
		Globs:          []string{"file.txt" + engine.TmplExtension},
		Out:            "file.txt",
	}
	require.NoError(b, os.WriteFile(filepath.Join(srcdir, tmpl.Globs[0]),
		[]byte("# Code generated by kickr; DO NOT EDIT.\n{{ .Str }}\n"),
		files.RwRR))

	for b.Loop() {
		// Act
		err := engine.ApplyTemplate(b.Context(), os.DirFS(srcdir), testutils.Root(b, destdir), tmpl, testconfig{Str: "value"})

		// Assert
		require.NoError(b, err)
	}
}
