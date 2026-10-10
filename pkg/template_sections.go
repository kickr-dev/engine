package engine

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/kickr-dev/engine/pkg/files"
)

// ApplySections regenerates each section defined in input tmpl, the rest of the already existing tmpl file is kept as is.
//
// The whole template is rendered and each rendered section (markers included) replaces its counterpart in the destination,
// user-added content outside sections is never templatized.
//
// The destination must already exist.
func ApplySections[T any](fsys fs.FS, destdir string, tmpl Template[T], data any) error {
	// force out localization since generation is always done on current fs
	out, err := filepath.Localize(tmpl.Out)
	if err != nil {
		return fmt.Errorf("localize path: %w", err)
	}
	out = filepath.Join(destdir, out)

	actual, err := os.ReadFile(out)
	if err != nil { // output file should exist since ApplySections is expected to be called from ApplyTemplate or at least after
		return fmt.Errorf("read file: %w", err)
	}

	// template the output file as if it was the first generation to get one full rendered content from which to pick all sections
	tt, err := newTemplate(path.Base(tmpl.Globs[0]), tmpl.Delimiters)
	if err != nil {
		return fmt.Errorf("new template: %w", err)
	}
	if tt, err = tt.ParseFS(fsys, tmpl.Globs...); err != nil {
		return fmt.Errorf("parse template file(s): %w", err)
	}
	var rendered bytes.Buffer
	if err := tt.Execute(&rendered, data); err != nil {
		return fmt.Errorf("template execution: %w", err)
	}

	expected := actual
	for _, section := range tmpl.Sections {
		begin, end := []byte(section.Begin), []byte(section.End)

		_, rest, found := bytes.Cut(rendered.Bytes(), begin)
		inner, _, closed := bytes.Cut(rest, end)
		if !found {
			GetLogger().Debugf("skipping section '%s' of '%s' since it's not rendered", section.Begin, filepath.Base(out))
			continue
		}
		if !closed {
			GetLogger().Warnf("skipping section '%s' of '%s', begin marker is rendered but not its end marker", section.Begin, filepath.Base(out))
			continue
		}

		before, rest, found := bytes.Cut(expected, begin)
		_, after, closed := bytes.Cut(rest, end)
		if !found || !closed {
			GetLogger().Infof("skipping section '%s' of '%s' since its markers are missing", section.Begin, filepath.Base(out))
			continue
		}
		expected = slices.Concat(before, begin, inner, end, after)
	}

	// leave an unchanged out untouched (no disk write, mtime kept), a read failure falls writes the file as if there was changes
	if bytes.Equal(actual, expected) {
		return nil
	}
	if err := os.WriteFile(out, expected, files.RwRR); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}
