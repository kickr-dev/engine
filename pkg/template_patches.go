package engine

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"gitlab.com/kickr-dev/engine/pkg/files"
)

// ApplyPatches applies patches defined in input tmpl, if any.
//
// Each patch is templatized using Go template and then patched on provided tmpl file.
func ApplyPatches[T any](fsys fs.FS, root *os.Root, tmpl Template[T], data any) error {
	if len(tmpl.Patches) == 0 {
		return nil
	}
	GetLogger().Infof("applying patches on '%s'", path.Base(tmpl.Out))

	apply := func(diff *gitdiff.File) error {
		file, err := root.OpenFile(tmpl.Out, os.O_RDWR|os.O_CREATE, files.RwRR)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer file.Close()

		var output bytes.Buffer
		if err := gitdiff.Apply(&output, file, diff); err != nil {
			return fmt.Errorf("apply diff: %w", err)
		}

		// truncate manually (instead of os.O_TRUNC) and after apply because patching needs the initial content
		if err := file.Truncate(int64(output.Len())); err != nil {
			return fmt.Errorf("truncate file: %w", err)
		}
		if _, err := file.WriteAt(output.Bytes(), 0); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
		return nil
	}

	errs := make([]error, 0, len(tmpl.Patches))
	for _, patch := range tmpl.Patches {
		patchname := path.Base(patch)
		GetLogger().Debugf("applying patch file '%s'", patchname)

		tt, err := newTemplate(patchname, tmpl.Delimiters)
		if err != nil {
			errs = append(errs, fmt.Errorf("new template patch '%s': %w", patchname, err))
			continue
		}
		if tt, err = tt.ParseFS(fsys, patch); err != nil {
			errs = append(errs, fmt.Errorf("parse template patch '%s': %w", patchname, err))
			continue
		}

		var buffer bytes.Buffer
		if err := tt.Execute(&buffer, data); err != nil {
			errs = append(errs, fmt.Errorf("template patch execution '%s': %w", patchname, err))
			continue
		}

		diffs, _, err := gitdiff.Parse(&buffer)
		if err != nil {
			errs = append(errs, fmt.Errorf("parse git patch '%s': %w", patchname, err))
			continue
		}

		for index, diff := range diffs {
			GetLogger().Debugf("applying diff number '%d' of '%s'", index, patchname)
			if err := apply(diff); err != nil {
				errs = append(errs, fmt.Errorf("apply diff number '%d' of '%s': %w", index, patchname, err))
			}
		}
	}
	return errors.Join(errs...)
}
