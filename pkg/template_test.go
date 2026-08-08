package engine_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	engine "github.com/kickr-dev/engine/pkg"
	"github.com/kickr-dev/engine/pkg/files"
)

func TestApplyTemplate(t *testing.T) {
	configure := func(t *testing.T, l engine.Logger) {
		t.Helper()

		funcs := engine.WithFuncMap(template.FuncMap{"ping": func() string { return "pong" }})

		initial := engine.GetLogger()
		engine.Configure(engine.WithLogger(l), funcs)
		t.Cleanup(func() { engine.Configure(engine.WithLogger(initial), funcs) })
	}

	t.Run("error_missing_out", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		// Act
		err := engine.ApplyTemplate(os.DirFS(destdir), destdir, engine.Template[testconfig]{}, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "localize path")
	})

	t.Run("error_read_template_out", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{Out: "dir"}
		require.NoError(t, os.Mkdir(filepath.Join(destdir, template.Out), files.RwxRxRxRx))

		// Act
		err := engine.ApplyTemplate(os.DirFS(destdir), destdir, template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "should generate")
	})

	t.Run("error_template_invalid_globs", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{Out: "file.txt"}

		buf := strings.Builder{}
		logger := engine.NewTestLogger(&buf)
		configure(t, logger)

		// Act
		err := engine.ApplyTemplate(os.DirFS(destdir), destdir, template, testconfig{})

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
		err := engine.ApplyTemplate(os.DirFS(destdir), destdir, template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "parse template file(s)")
	})

	t.Run("success_template_already_exists", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{Out: "file.txt"}
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out), []byte("some not empty file"), files.RwRR))

		buf := strings.Builder{}
		logger := engine.NewTestLogger(&buf)
		configure(t, logger)

		// Act
		err := engine.ApplyTemplate(os.DirFS(destdir), destdir, template, testconfig{})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, buf.String(), fmt.Sprintf("not generating '%s' since it already exists (or was modified manually)", template.Out))
	})

	t.Run("error_remove", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("running as root bypasses permission checks")
		}

		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Out:    "file.txt",
			Remove: func(testconfig) bool { return true },
		}
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out), []byte("content"), files.RwRR))
		require.NoError(t, os.Chmod(destdir, 0o500))
		t.Cleanup(func() { _ = os.Chmod(destdir, files.RwxRxRxRx) })

		// Act
		err := engine.ApplyTemplate(os.DirFS(destdir), destdir, template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "remove")
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
		err := engine.ApplyTemplate(os.DirFS(srcdir), destdir, template, testconfig{})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "pong", string(content))
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
			[]engine.Template[testmodule]{
				{Globs: []string{"invalid.txt"}, Out: "file.txt"},
			})

		// Act
		err := generator(ctx, destdir, []testmodule{{directory: "."}})

		// Assert
		assert.ErrorIs(t, err, engine.ErrFailedGeneration)
	})

	t.Run("success_no_module", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		generator := engine.GeneratorModules(os.DirFS(destdir), modules,
			[]engine.Template[testmodule]{
				{Globs: []string{"invalid.txt"}, Out: "file.txt"},
			})

		// Act
		err := generator(ctx, destdir, nil)

		// Assert
		require.NoError(t, err)
		assert.NoFileExists(t, filepath.Join(destdir, "file.txt"))
	})

	t.Run("success_root_module", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, "file.txt"+engine.TmplExtension), []byte("{{ .Dir }}"), files.RwRR))

		generator := engine.GeneratorModules(os.DirFS(srcdir), modules,
			[]engine.Template[testmodule]{
				{Globs: []string{"file.txt" + engine.TmplExtension}, Out: "file.txt"},
			})

		// Act
		err := generator(ctx, destdir, []testmodule{{directory: "."}})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, "file.txt"))
		require.NoError(t, err)
		assert.Equal(t, ".", string(content))
	})

	t.Run("success_multiple_modules", func(t *testing.T) {
		// Arrange
		srcdir := t.TempDir()
		destdir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(srcdir, "file.txt"+engine.TmplExtension), []byte("{{ .Dir }}"), files.RwRR))

		generator := engine.GeneratorModules(os.DirFS(srcdir), modules,
			[]engine.Template[testmodule]{
				{Globs: []string{"file.txt" + engine.TmplExtension}, Out: "file.txt"},
			})

		// Act
		err := generator(ctx, destdir, []testmodule{{directory: "."}, {directory: "apps/api"}})

		// Assert
		require.NoError(t, err)
		for _, dir := range []string{".", "apps/api"} {
			content, err := os.ReadFile(filepath.Join(destdir, dir, "file.txt"))
			require.NoError(t, err)
			assert.Equal(t, dir, string(content))
		}
	})

	t.Run("success_remove_in_module", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		out := filepath.Join(destdir, "apps", "api", "file.txt")
		require.NoError(t, os.MkdirAll(filepath.Dir(out), files.RwxRxRxRx))
		require.NoError(t, os.WriteFile(out, []byte("some not empty file"), files.RwRR))

		generator := engine.GeneratorModules(os.DirFS(destdir), modules,
			[]engine.Template[testmodule]{
				{Out: "file.txt", Remove: func(testmodule) bool { return true }},
			})

		// Act
		err := generator(ctx, destdir, []testmodule{{directory: "apps/api"}})

		// Assert
		require.NoError(t, err)
		assert.NoFileExists(t, out)
	})
}

func TestApplyPatches(t *testing.T) {
	t.Run("error_missing_out", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()

		// Act
		err := engine.ApplyPatches(os.DirFS(destdir), destdir, engine.Template[testconfig]{}, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "localize path")
	})

	t.Run("error_missing_template_patch", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Out:     "file.txt",
			Patches: []string{"file.patch"},
		}

		// Act
		err := engine.ApplyPatches(os.DirFS(destdir), destdir, template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "parse template patch")
	})

	t.Run("error_template_patch", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Out:     "file.txt",
			Patches: []string{"file.patch"},
		}
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Patches[0]), []byte("{{ .invalid }}"), files.RwRR))

		// Act
		err := engine.ApplyPatches(os.DirFS(destdir), destdir, template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "template patch execution")
	})

	t.Run("error_invalid_patch_file", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Out:     "file.txt",
			Patches: []string{"file.patch"},
		}
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Patches[0]), []byte(`
diff --git a/file.txt b/file.txt
index 332d5ce..39af8aa 100644
--- a/file.txt
+++ b/file.txt
@@ -1,0 +1,2 @@
+value`), files.RwRR))

		// Act
		err := engine.ApplyPatches(os.DirFS(destdir), destdir, template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "parse git patch")
	})

	t.Run("error_apply_patch", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Out:     "file.txt",
			Patches: []string{"file.patch"},
		}
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Patches[0]), []byte(`
diff --git a/file.txt b/file.txt
index 332d5ce..39af8aa 100644
--- a/file.txt
+++ b/file.txt
@@ -2,0 +2,1 @@
+value`), files.RwRR))

		// Act
		err := engine.ApplyPatches(os.DirFS(destdir), destdir, template, testconfig{})

		// Assert
		assert.ErrorContains(t, err, "apply diff number")
	})

	t.Run("success_create", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Out:     "file.txt",
			Patches: []string{"file.patch"},
		}
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Patches[0]), []byte(`
diff --git a/file.txt b/file.txt
index 332d5ce..39af8aa 100644
--- a/file.txt
+++ b/file.txt
@@ -1,0 +1,1 @@
+value`), files.RwRR))

		// Act
		err := engine.ApplyPatches(os.DirFS(destdir), destdir, template, testconfig{})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "value", string(content))
	})

	t.Run("success_update_shorter", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Out:     "file.txt",
			Patches: []string{"file.patch"},
		}
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out), []byte("some replaced value in non empty file"), files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Patches[0]), []byte(`
diff --git a/file.txt b/file.txt
index 332d5ce..39af8aa 100644
--- a/file.txt
+++ b/file.txt
@@ -1 +1 @@
-some replaced value in non empty file
\ No newline at end of file
+some not empty file
\ No newline at end of file`), files.RwRR))

		// Act
		err := engine.ApplyPatches(os.DirFS(destdir), destdir, template, testconfig{})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "some not empty file", string(content))
	})

	t.Run("success_update", func(t *testing.T) {
		// Arrange
		destdir := t.TempDir()
		template := engine.Template[testconfig]{
			Out:     "file.txt",
			Patches: []string{"file.patch"},
		}
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Out), []byte("some not empty file"), files.RwRR))
		require.NoError(t, os.WriteFile(filepath.Join(destdir, template.Patches[0]), []byte(`
diff --git a/file.txt b/file.txt
index 332d5ce..39af8aa 100644
--- a/file.txt
+++ b/file.txt
@@ -1 +1 @@
-some not empty file
\ No newline at end of file
+some replaced value in non empty file
\ No newline at end of file`), files.RwRR))

		// Act
		err := engine.ApplyPatches(os.DirFS(destdir), destdir, template, testconfig{})

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(filepath.Join(destdir, template.Out))
		require.NoError(t, err)
		assert.Equal(t, "some replaced value in non empty file", string(content))
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
		err = engine.ExecuteTemplate(tmpl, nil, dest, engine.PolicyRemove, 0)

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
		err := engine.ExecuteTemplate(tmpl, nil, dest, engine.PolicyRemove, 0)

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
		err = engine.ExecuteTemplate(tmpl, data, filepath.Dir(dest), engine.PolicyRemove, 0)

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
		err = engine.ExecuteTemplate(tmpl, data, dest, engine.PolicyRemove, 0)

		// Assert
		require.NoError(t, err)
		content, err := os.ReadFile(dest)
		require.NoError(t, err)
		assert.Equal(t, "hey ! A name", string(content))
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
		err = engine.ExecuteTemplate(tmpl, nil, dest, engine.PolicyRemove, 0)

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
		err = engine.ExecuteTemplate(tmpl, map[string]any{}, dest, engine.PolicyKeep, files.RwxRxRxRx)

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
		err = engine.ExecuteTemplate(tmpl, nil, dest, engine.PolicyRemove, 0)

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
		err = engine.ExecuteTemplate(tmpl, nil, dest, engine.PolicyKeep, 0)

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
		err = engine.ExecuteTemplate(tmpl, nil, dest, engine.PolicyRemove, 0)

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
		err = engine.ExecuteTemplate(tmpl, nil, dest, engine.PolicyRemove, 0)

		// Assert
		require.NoError(t, err)
		assert.NoFileExists(t, dest)
	})
}
