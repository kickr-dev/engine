package engine_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	engine "github.com/kickr-dev/engine/pkg"
	"github.com/kickr-dev/engine/pkg/files"
)

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
