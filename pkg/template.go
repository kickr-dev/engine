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
	"sync"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"gitlab.com/kickr-dev/engine/pkg/files"
)

// ApplyTemplate generates an input Template with associated data,
// or only regenerates its sections when the file is kept (see ApplySections),
// and then applies its patches (see ApplyPatches).
//
// Whether the file is generated, kept, removed or skipped depends on its GeneratePolicy and on Options.Force from ctx,
// and a removed or skipped file gets no patches.
func ApplyTemplate[T any](ctx context.Context, fsys fs.FS, root *os.Root, tmpl Template[T], config T) error {
	opts := OptionsFromContext(ctx)
	out := filepath.Join(root.Name(), tmpl.Out)

	var policy GeneratePolicy
	if tmpl.GeneratePolicy != nil {
		policy = tmpl.GeneratePolicy(out, config)
	}

	var generate bool
	switch policy {
	case PolicyAlways:
		generate = true
	case PolicyRemove:
		if _, err := root.Lstat(tmpl.Out); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return fmt.Errorf("lstat '%s': %w", tmpl.Out, err)
		}

		GetLogger().Debugf("removing '%s'", tmpl.Out)
		if err := root.RemoveAll(tmpl.Out); err != nil {
			return fmt.Errorf("remove '%s': %w", tmpl.Out, err)
		}
		return nil
	case PolicySkip:
		return nil
	case 0, PolicyDefault:
		generate = opts.Force
		if !generate {
			ok, err := IsGenerated(out)
			if err != nil {
				return fmt.Errorf("is generated: %w", err)
			}
			generate = ok
		}
	default:
		return fmt.Errorf("invalid generate policy %d", policy)
	}

	switch {
	case !generate:
		GetLogger().Infof("not generating '%s' since it already exists (or was modified manually)", tmpl.Out)
		if err := ApplySections(ctx, fsys, root, tmpl, config); err != nil {
			return fmt.Errorf("apply sections: %w", err)
		}
	case len(tmpl.Globs) == 0:
		GetLogger().Warnf("empty template 'globs', skipping '%s' generation", tmpl.Out)
	default:
		GetLogger().Debugf("generating '%s'", tmpl.Out)
		tt, err := newTemplate(path.Base(tmpl.Globs[0]), tmpl.Delimiters, opts.Funcs)
		if err != nil {
			return fmt.Errorf("new template: %w", err)
		}
		if tt, err = tt.ParseFS(fsys, tmpl.Globs...); err != nil {
			return fmt.Errorf("parse template file(s): %w", err)
		}
		if err := executeTemplate(ctx, tt, root, tmpl, config); err != nil {
			return fmt.Errorf("template execute: %w", err)
		}
	}

	if err := ApplyPatches(ctx, fsys, root, tmpl, config); err != nil {
		return fmt.Errorf("apply patches: %w", err)
	}
	return nil
}

// executeTemplate executes tt with config and writes the result to tmpl.Out inside root,
// first dropping the generated notice lines when ctx asks for it (see Options.StripNotice).
//
// An unchanged file is left as is, though its permissions are always reapplied from tmpl.Mode
// (files.RwRR when zero) minus the system umask (see files.Umask).
func executeTemplate[T any](ctx context.Context, tt *template.Template, root *os.Root, tmpl Template[T], config T) error {
	opts := OptionsFromContext(ctx)

	var buf bytes.Buffer
	if err := tt.Execute(&buf, config); err != nil {
		return fmt.Errorf("template execution: %w", err)
	}

	content := buf.Bytes()
	if opts.StripNotice {
		content = noticeBlock.ReplaceAll(content, nil)
	}

	if ok := IsEmpty(content, tmpl.EmptyPolicy); ok {
		base := filepath.Base(tmpl.Out)
		if _, err := root.Lstat(tmpl.Out); err != nil {
			GetLogger().Debugf("not generating '%s' since it would be empty", base)
			return nil
		}
		GetLogger().Debugf("removing '%s' since it's empty", base)
		if err := root.RemoveAll(tmpl.Out); err != nil {
			return fmt.Errorf("remove '%s': %w", base, err)
		}
		return nil
	}

	if err := root.MkdirAll(filepath.Dir(tmpl.Out), files.RwxRxRxRx); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// apply the requested permissions to tmpl.Out file, honoring the system umask
	requested := tmpl.Mode
	if requested == 0 {
		requested = files.RwRR
	}
	requested &^= files.Umask()

	// leave an unchanged tmpl.Out untouched (no disk write, mtime kept), a read failure falls writes the file as if there was changes
	if existing, err := root.ReadFile(tmpl.Out); err == nil && bytes.Equal(existing, content) {
		if err := root.Chmod(tmpl.Out, requested); err != nil {
			return fmt.Errorf("chmod: %w", err)
		}
		return nil
	}

	file, err := root.OpenFile(tmpl.Out, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, requested)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(content); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	// force refresh permissions
	if err := file.Chmod(requested); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	return nil
}

// baseTemplate holds Sprig and engine default funcs, validated once for the process' lifetime and cloned by newTemplate.
var baseTemplate = sync.OnceValue(func() *template.Template {
	return template.New("").Funcs(sprig.HermeticTxtFuncMap()).Funcs(FuncMap())
})

// newTemplate creates a new template.Template with the provided delimiters
// and all funcs coming from the following places:
//   - Sprig hermetic ones (no environment, network, date or random funcs)
//   - Engine default ones
//   - The given funcs, usually Options.Funcs
func newTemplate(name string, delims Delimiters, funcs template.FuncMap) (*template.Template, error) {
	base, err := baseTemplate().Clone()
	if err != nil {
		return nil, fmt.Errorf("clone base template: %w", err)
	}
	return base.New(name).
		Funcs(funcs).
		Delims(delims.StartDelim, delims.EndDelim), nil
}
