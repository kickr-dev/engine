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

	engine "github.com/kickr-dev/engine/pkg"
	"github.com/kickr-dev/engine/pkg/files"
	"github.com/kickr-dev/engine/testutils"
)

func TestApplyTemplate(t *testing.T) {
	configure := func(t *testing.T, l engine.Logger) {
		t.Helper()

		funcs := engine.WithFuncMap(template.FuncMap{"ping": func() string { return "pong" }})

		initial := engine.GetLogger()
		engine.Configure(engine.WithLogger(l), funcs)
		t.Cleanup(func() { engine.Configure(engine.WithLogger(initial), funcs) })
	}

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
		err := engine.ApplyTemplate(os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{})

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
		err := engine.ApplyTemplate(os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "remove")
	})

	t.Run("error_read_template_out", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{Out: "dir"}
		require.NoError(t, os.Mkdir(filepath.Join(destdir, template.Out), files.RwxRxRxRx))

		// Act
		err := engine.ApplyTemplate(os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{})

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
		err := engine.ApplyTemplate(os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "invalid generate policy")
	})

	t.Run("error_template_invalid_globs", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{Out: "file.txt"}

		buf := strings.Builder{}
		logger := engine.NewTestLogger(&buf)
		configure(t, logger)

		// Act
		err := engine.ApplyTemplate(os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{})

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
		err := engine.ApplyTemplate(os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{})

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
		err := engine.ApplyTemplate(os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "path escapes from parent")
		assert.NoFileExists(t, filepath.Join(tmp, "outside.txt"))
	})

	t.Run("success_custom_func", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs: []string{"file.txt" + engine.TmplExtension},
			Out:   "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), []byte("{{ ping }}"), files.RwRR))

		buf := strings.Builder{}
		logger := engine.NewTestLogger(&buf)
		configure(t, logger)

		// Act
		err := engine.ApplyTemplate(os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

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
		err := engine.ApplyTemplate(os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{Str: "value"})

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
		err := engine.ApplyTemplate(os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "generated", string(content))
	})

	t.Run("success_policy_default_forced", func(t *testing.T) {
		// Arrange
		engine.Configure(engine.WithForce(true))
		t.Cleanup(func() { engine.Configure(engine.WithForce(false)) })

		srcdir := t.TempDir()
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Globs: []string{"file.txt" + engine.TmplExtension},
			Out:   "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, template.Globs[0]), []byte("generated"), files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out), []byte("live"), files.RwRR))

		// Act
		err := engine.ApplyTemplate(os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

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
		err := engine.ApplyTemplate(os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{Str: "value"})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "live <!-- BEGIN -->stale<!-- END --> live", string(content))
	})

	t.Run("success_policy_skip_forced", func(t *testing.T) {
		// Arrange
		engine.Configure(engine.WithForce(true))
		t.Cleanup(func() { engine.Configure(engine.WithForce(false)) })

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
		err := engine.ApplyTemplate(os.DirFS(srcdir), testutils.Root(t, destdir), template, testconfig{})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "live", string(content))
	})

	t.Run("success_policy_remove_forced", func(t *testing.T) {
		// Arrange
		engine.Configure(engine.WithForce(true))
		t.Cleanup(func() { engine.Configure(engine.WithForce(false)) })

		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			GeneratePolicy: engine.StaticPolicy[testconfig](engine.PolicyRemove),
			Out:            "file.txt",
		}
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out), []byte("live"), files.RwRR))

		// Act
		err := engine.ApplyTemplate(os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{})

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
		err := engine.ApplyTemplate(os.DirFS(destdir), testutils.Root(t, destdir), template, testconfig{Str: "value"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(destdir, template.Out), gotPath)
		assert.Equal(t, testconfig{Str: "value"}, gotConfig)
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

func TestExecuteTemplate(t *testing.T) {
	t.Run("error_mkdir", func(t *testing.T) {
		// Arrange
		dir := filepath.Join(t.TempDir(), "dir")
		require.NoError(t, os.Mkdir(dir, files.RwxRxRxRx))

		// create empty file (at midlevel) to ensure os.MkdirAll fails
		dest := filepath.Join(dir, "file.txt", "file.txt")
		file, err := os.Create(filepath.Dir(dest))
		require.NoError(t, err)
		require.NoError(t, file.Close())

		tmpl, err := template.New("template.txt").Parse("content")
		require.NoError(t, err)

		// Act
		err = engine.ExecuteTemplate(tmpl, nil, testutils.Root(t, dir), filepath.Join("file.txt", "file.txt"), engine.PolicyRemoveEmpty, 0)

		// Assert
		assert.ErrorContains(t, err, "mkdir")
	})

	t.Run("error_execute", func(t *testing.T) {
		// Arrange
		tmp := t.TempDir()
		dest := filepath.Join(tmp, "template-result.txt")

		// not parsing any file with template to ensure tmpl.Execute fails
		tmpl := template.New("template.txt").Funcs(engine.FuncMap())

		// Act
		err := engine.ExecuteTemplate(tmpl, nil, testutils.Root(t, filepath.Dir(dest)), filepath.Base(dest), engine.PolicyRemoveEmpty, 0)

		// Assert
		assert.ErrorContains(t, err, "template execution")
		assert.ErrorContains(t, err, `"template.txt" is an incomplete or empty template`)
	})

	t.Run("error_write_dir", func(t *testing.T) {
		// Arrange
		tmp := t.TempDir()

		// create template file
		src := filepath.Join(tmp, "template.txt")
		require.NoError(t, os.WriteFile(src, []byte("{{ .name }}"), files.RwRR))

		// create a file in dest to ensure WriteFile fails since it's a directory
		dest := filepath.Join(tmp, "dir")
		require.NoError(t, os.MkdirAll(filepath.Dir(dest), files.RwxRxRxRx))

		data := map[string]string{"name": "hey ! A name"}

		tmpl, err := template.New("template.txt").
			Funcs(engine.FuncMap()).
			ParseFiles(src)
		require.NoError(t, err)

		// Act
		err = engine.ExecuteTemplate(tmpl, data, testutils.Root(t, filepath.Dir(dest)), ".", engine.PolicyRemoveEmpty, 0)

		// Assert
		assert.ErrorContains(t, err, "open file")
	})

	t.Run("success_dest_exists", func(t *testing.T) {
		// Arrange
		tmp := t.TempDir()

		// create template file
		src := filepath.Join(tmp, "template.txt")
		require.NoError(t, os.WriteFile(src, []byte("{{ .name }}"), files.RwRR))

		// create dest to ensure os.Remove works
		dest := filepath.Join(tmp, "template-result.txt")
		file, err := os.Create(dest)
		require.NoError(t, err)
		require.NoError(t, file.Close())

		data := map[string]string{"name": "hey ! A name"}

		tmpl, err := template.New("template.txt").
			Funcs(engine.FuncMap()).
			ParseFiles(src)
		require.NoError(t, err)

		// Act
		err = engine.ExecuteTemplate(tmpl, data, testutils.Root(t, filepath.Dir(dest)), filepath.Base(dest), engine.PolicyRemoveEmpty, 0)

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(dest)
		require.NoError(t, err)
		assert.Equal(t, "hey ! A name", string(content))
	})

	t.Run("success_unchanged_content", func(t *testing.T) {
		// Arrange
		tmp := t.TempDir()

		src := filepath.Join(tmp, "template.txt")
		require.NoError(t, os.WriteFile(src, []byte("content"), files.RwRR))

		dest := filepath.Join(tmp, "result.txt")
		require.NoError(t, os.WriteFile(dest, []byte("content"), files.Rw))
		mtime := time.Now().Add(-time.Hour).Truncate(time.Second)
		require.NoError(t, os.Chtimes(dest, mtime, mtime))

		tmpl, err := template.New("template.txt").
			Funcs(engine.FuncMap()).
			ParseFiles(src)
		require.NoError(t, err)

		// Act
		err = engine.ExecuteTemplate(tmpl, nil, testutils.Root(t, filepath.Dir(dest)), filepath.Base(dest), engine.PolicyRemoveEmpty, 0)

		// Assert
		require.NoError(t, err)
		info, err := os.Stat(dest)
		require.NoError(t, err)
		assert.Equal(t, mtime, info.ModTime())
		assert.Equal(t, files.RwRR&^files.Umask(), info.Mode())
	})

	t.Run("success_mode_default", func(t *testing.T) {
		// Arrange
		tmp := t.TempDir()

		src := filepath.Join(tmp, "template.txt")
		require.NoError(t, os.WriteFile(src, []byte("content"), files.RwRR))

		dest := filepath.Join(tmp, "result.txt")

		tmpl, err := template.New("template.txt").
			Funcs(engine.FuncMap()).
			ParseFiles(src)
		require.NoError(t, err)

		// Act
		err = engine.ExecuteTemplate(tmpl, nil, testutils.Root(t, filepath.Dir(dest)), filepath.Base(dest), engine.PolicyRemoveEmpty, 0)

		// Assert
		require.NoError(t, err)
		info, err := os.Stat(dest)
		require.NoError(t, err)
		assert.Equal(t, files.RwRR&^files.Umask(), info.Mode())
	})

	t.Run("success_mode_explicit", func(t *testing.T) {
		// Arrange
		tmp := t.TempDir()

		src := filepath.Join(tmp, "template.sh")
		require.NoError(t, os.WriteFile(src, []byte("#!/bin/..."), files.RwRR))

		dest := filepath.Join(tmp, "script.sh")

		tmpl, err := template.New("template.sh").
			Funcs(engine.FuncMap()).
			ParseFiles(src)
		require.NoError(t, err)

		// Act
		err = engine.ExecuteTemplate(tmpl, map[string]any{}, testutils.Root(t, filepath.Dir(dest)), filepath.Base(dest), engine.PolicyKeepEmpty, files.RwxRxRxRx)

		// Assert
		require.NoError(t, err)
		info, err := os.Stat(dest)
		require.NoError(t, err)
		assert.Equal(t, files.RwxRxRxRx&^files.Umask(), info.Mode())
	})

	t.Run("success_skip_empty", func(t *testing.T) {
		// Arrange
		tmp := t.TempDir()

		src := filepath.Join(tmp, "template.txt")
		require.NoError(t, os.WriteFile(src, nil, files.RwRR))

		dest := filepath.Join(tmp, "dir", "template-result.txt")

		tmpl, err := template.New("template.txt").
			Funcs(engine.FuncMap()).
			ParseFiles(src)
		require.NoError(t, err)

		// Act
		err = engine.ExecuteTemplate(tmpl, nil, testutils.Root(t, tmp), filepath.Join("dir", "template-result.txt"), engine.PolicyRemoveEmpty, 0)

		// Assert
		require.NoError(t, err)
		assert.NoDirExists(t, filepath.Dir(dest))
		assert.NoFileExists(t, dest)
	})

	t.Run("success_keep_empty", func(t *testing.T) {
		// Arrange
		tmp := t.TempDir()

		src := filepath.Join(tmp, "template.txt")
		err := os.WriteFile(src, nil, files.RwRR)
		require.NoError(t, err)

		dest := filepath.Join(tmp, "template-result.txt")

		tmpl, err := template.New("template.txt").
			Funcs(engine.FuncMap()).
			ParseFiles(src)
		require.NoError(t, err)

		// Act
		err = engine.ExecuteTemplate(tmpl, nil, testutils.Root(t, filepath.Dir(dest)), filepath.Base(dest), engine.PolicyKeepEmpty, 0)

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(dest)
		require.NoError(t, err)
		assert.Empty(t, content)
	})

	t.Run("error_remove_existing_empty", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("running as root bypasses permission checks")
		}

		// Arrange
		tmp := t.TempDir()

		src := filepath.Join(tmp, "template.txt")
		require.NoError(t, os.WriteFile(src, nil, files.RwRR))

		destdir := filepath.Join(tmp, "destdir")
		require.NoError(t, os.Mkdir(destdir, files.RwxRxRxRx))
		dest := filepath.Join(destdir, "template-result.txt")
		require.NoError(t, os.WriteFile(dest, []byte("some not empty file"), files.RwRR))
		require.NoError(t, os.Chmod(destdir, 0o500))
		t.Cleanup(func() { _ = os.Chmod(destdir, files.RwxRxRxRx) })

		tmpl, err := template.New("template.txt").
			Funcs(engine.FuncMap()).
			ParseFiles(src)
		require.NoError(t, err)

		// Act
		err = engine.ExecuteTemplate(tmpl, nil, testutils.Root(t, filepath.Dir(dest)), filepath.Base(dest), engine.PolicyRemoveEmpty, 0)

		// Assert
		assert.ErrorContains(t, err, "remove")
	})

	t.Run("success_remove_existing_empty", func(t *testing.T) {
		// Arrange
		tmp := t.TempDir()

		src := filepath.Join(tmp, "template.txt")
		require.NoError(t, os.WriteFile(src, nil, files.RwRR))

		dest := filepath.Join(tmp, "template-result.txt")
		require.NoError(t, os.WriteFile(dest, []byte("some not empty file"), files.RwRR))

		tmpl, err := template.New("template.txt").
			Funcs(engine.FuncMap()).
			ParseFiles(src)
		require.NoError(t, err)

		// Act
		err = engine.ExecuteTemplate(tmpl, nil, testutils.Root(t, filepath.Dir(dest)), filepath.Base(dest), engine.PolicyRemoveEmpty, 0)

		// Assert
		require.NoError(t, err)
		assert.NoFileExists(t, dest)
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
		err := engine.ApplyTemplate(os.DirFS(srcdir), testutils.Root(b, destdir), tmpl, testconfig{Str: "value"})

		// Assert
		require.NoError(b, err)
	}
}
