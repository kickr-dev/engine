package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/kickr-dev/engine/pkg/files"
)

// GeneratorTemplates is a simple generator taking as input a filesystem and all templates to apply.
//
// Errors encountered during templates generation are logged, in that case a final error being ErrFailedGeneration is returned.
func GeneratorTemplates[T any](fsys fs.FS, templates []Template[T]) Generator[T] {
	return func(_ context.Context, destdir string, config T) error {
		var errcount int
		for _, tmpl := range templates {
			if err := ApplyTemplate(fsys, destdir, tmpl, config); err != nil {
				errcount++
				GetLogger().Errorf("failed to generate '%s': %v", path.Base(tmpl.Out), err)
			}
		}
		if errcount > 0 {
			return ErrFailedGeneration
		}
		return nil
	}
}

// GeneratorModules is a generator applying all input templates inside each module directory.
//
// The modules function extracts the modules slice from the parsed configuration.
//
// Each Template.Out is relative to each module where it will be generated
// and Template.Remove is up to the characteristics of a given module.
//
// Errors encountered during templates generation are logged, in that case a final error being ErrFailedGeneration is returned.
func GeneratorModules[T any, M Module](fsys fs.FS, modules func(config T) []M, templates []Template[M]) Generator[T] {
	generator := GeneratorTemplates(fsys, templates)

	return func(ctx context.Context, destdir string, config T) error {
		var errcount int
		for _, module := range modules(config) {
			if err := generator(ctx, filepath.Join(destdir, module.Dir()), module); err != nil {
				errcount++
				GetLogger().Errorf("failed to generate '%s': %v", module.Dir(), err)
			}
		}
		if errcount > 0 {
			return ErrFailedGeneration
		}
		return nil
	}
}

// ApplyTemplate writes or deletes an input Template with associated data.
func ApplyTemplate[T any](fsys fs.FS, destdir string, tmpl Template[T], config T) error {
	// force out localization since generation is always done on current fs
	out, err := filepath.Localize(tmpl.Out)
	if err != nil {
		return fmt.Errorf("localize path: %w", err)
	}
	out = filepath.Join(destdir, out)

	// remove file in case result is asking it
	if tmpl.Remove != nil && tmpl.Remove(config) {
		if !files.Exists(out) {
			return nil
		}

		GetLogger().Debugf("removing '%s'", tmpl.Out)
		if err := os.RemoveAll(out); err != nil {
			GetLogger().Warnf("failed to delete '%s': %v", tmpl.Out, err)
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
		tt, err := template.New(path.Base(tmpl.Globs[0])).
			Funcs(sprig.FuncMap()).
			Funcs(FuncMap()).
			Funcs(o.funcs).
			Delims(tmpl.StartDelim, tmpl.EndDelim).
			ParseFS(fsys, tmpl.Globs...)
		if err != nil {
			return fmt.Errorf("parse template file(s): %w", err)
		}
		if err := ExecuteTemplate(tt, config, out, tmpl.EmptyPolicy); err != nil {
			return fmt.Errorf("template execute: %w", err)
		}
	}

	if len(tmpl.Patches) > 0 {
		GetLogger().Infof("applying patches on '%s'", path.Base(out))
		return ApplyPatches(fsys, destdir, tmpl, config)
	}
	return nil
}

// ApplyPatches apply patches defined in input tmpl.
// Each patch is templatized using Go template and then patched on provided tmpl file.
//
// It's the continuance function of ApplyTemplate (which only generates - if necessary - the initial template).
func ApplyPatches[T any](fsys fs.FS, destdir string, tmpl Template[T], data any) error {
	// force out localization since generation is always done on current fs
	out, err := filepath.Localize(tmpl.Out)
	if err != nil {
		return fmt.Errorf("localize path: %w", err)
	}
	out = filepath.Join(destdir, out)

	apply := func(diff *gitdiff.File) error {
		file, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE, files.RwRR)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer file.Close()

		var output bytes.Buffer
		if err := gitdiff.Apply(&output, file, diff); err != nil {
			return fmt.Errorf("apply diff: %w", err)
		}

		if _, err := file.Write(output.Bytes()); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
		return nil
	}

	errs := make([]error, 0, len(tmpl.Patches))
	for _, patch := range tmpl.Patches {
		patchname := path.Base(patch)
		GetLogger().Debugf("applying patch file '%s'", patchname)

		tt, err := template.New(patchname).
			Funcs(sprig.FuncMap()).
			Funcs(FuncMap()).
			Funcs(o.funcs).
			Delims(tmpl.StartDelim, tmpl.EndDelim).
			ParseFS(fsys, patch)
		if err != nil {
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

// exeRegex matches file extensions requiring the executable bit: any *sh extension (.sh, .bash, .zsh, ...) and .ps1.
var exeRegex = regexp.MustCompile(`^\.(\w*sh|ps1)$`)

// ExecuteTemplate runs tmpl.ExecuteTemplate with input data and write result into given out.
//
// When ExecuteTemplate is called, it truncates out in case it already exists and reevaluate its rights (specific to linux).
func ExecuteTemplate(tmpl *template.Template, data any, out string, policy EmptyPolicy) error {
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
			GetLogger().Warnf("failed to delete '%s': %v", base, err)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(out), files.RwxRxRxRx); err != nil && !errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("mkdir: %w", err)
	}

	// affect the right rights to out file
	mode := files.RwRR
	if exeRegex.MatchString(filepath.Ext(out)) {
		mode = files.RwxRxRxRx
	}

	file, err := os.OpenFile(out, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, mode)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	// force refresh rights
	if err := file.Chmod(mode); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	return nil
}
