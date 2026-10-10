package engine

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/kickr-dev/engine/pkg/files"
)

// ApplyTemplate writes or deletes an input Template with associated data.
func ApplyTemplate[T any](fsys fs.FS, destdir string, tmpl Template[T], config T) error {
	// force out localization since generation is always done on current fs
	out, err := filepath.Localize(tmpl.Out)
	if err != nil {
		return fmt.Errorf("localize path: %w", err)
	}
	out = filepath.Join(destdir, out)

	// remove file when tmpl.Remove asks for it
	if tmpl.Remove != nil && tmpl.Remove(config) {
		if !files.Exists(out) {
			return nil
		}

		GetLogger().Debugf("removing '%s'", tmpl.Out)
		if err := os.RemoveAll(out); err != nil {
			return fmt.Errorf("remove '%s': %w", tmpl.Out, err)
		}
		return nil
	}

	// avoid generating file if it already exists or something else
	ok, err := ShouldGenerate(out, tmpl.GeneratePolicy)
	if err != nil {
		return fmt.Errorf("should generate: %w", err)
	}
	switch {
	case !ok:
		GetLogger().Infof("not generating '%s' since it already exists (or was modified manually)", tmpl.Out)
	case len(tmpl.Globs) == 0:
		GetLogger().Warnf("empty template 'globs', skipping '%s' generation", tmpl.Out)
	default:
		GetLogger().Debugf("generating '%s'", tmpl.Out)
		tt, err := newTemplate(path.Base(tmpl.Globs[0]), tmpl.Delimiters).ParseFS(fsys, tmpl.Globs...)
		if err != nil {
			return fmt.Errorf("parse template file(s): %w", err)
		}
		if err := ExecuteTemplate(tt, config, out, tmpl.EmptyPolicy, tmpl.Mode); err != nil {
			return fmt.Errorf("template execute: %w", err)
		}
	}

	if !ok && len(tmpl.Globs) > 0 && len(tmpl.Sections) > 0 {
		GetLogger().Infof("applying sections on '%s'", tmpl.Out)
		if err := ApplySections(fsys, destdir, tmpl, config); err != nil {
			return fmt.Errorf("apply sections: %w", err)
		}
	}

	if len(tmpl.Patches) > 0 {
		GetLogger().Infof("applying patches on '%s'", path.Base(out))
		if err := ApplyPatches(fsys, destdir, tmpl, config); err != nil {
			return fmt.Errorf("apply patches: %w", err)
		}
	}
	return nil
}

// ExecuteTemplate runs tmpl.Execute with input data and writes the result into given out.
//
// When ExecuteTemplate is called, it truncates out in case it already exists and reevaluates its permissions.
//
// The input mode sets the requested file mode for the generated file (e.g. files.RwRR, files.RwxRxRxRx),
// defaulting to files.RwRR when not provided.
//
// The system umask (see files.Umask) is always applied on top (mode &^ umask, no-op on non-compatible platforms).
func ExecuteTemplate(tmpl *template.Template, data any, out string, policy EmptyPolicy, mode os.FileMode) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("template execution: %w", err)
	}

	if ok := IsEmpty(buf.Bytes(), policy); ok {
		base := filepath.Base(out)
		if !files.Exists(out) {
			GetLogger().Debugf("not generating '%s' since it would be empty", base)
			return nil
		}
		GetLogger().Debugf("removing '%s' since it's empty", base)
		if err := os.RemoveAll(out); err != nil {
			return fmt.Errorf("remove '%s': %w", base, err)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(out), files.RwxRxRxRx); err != nil && !errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("mkdir: %w", err)
	}

	// apply the requested permissions to out file, honoring the system umask
	requested := mode
	if requested == 0 {
		requested = files.RwRR
	}
	requested &^= files.Umask()

	file, err := os.OpenFile(out, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, requested)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	// force refresh permissions
	if err := file.Chmod(requested); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	return nil
}

// newTemplate creates a new template.Template with the provided delimiters
// and all funcs coming from the following places:
//   - Sprig
//   - Engine default ones
//   - Engine provided ones with Configure
func newTemplate(name string, delims Delimiters) *template.Template {
	return template.New(name).
		Funcs(sprig.FuncMap()).
		Funcs(FuncMap()).
		Funcs(funcs()).
		Delims(delims.StartDelim, delims.EndDelim)
}
