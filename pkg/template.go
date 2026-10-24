package engine

import (
	"bytes"
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
// The GeneratePolicy and the force option (see WithForce) decide whether the file is generated, kept, removed or skipped.
// A removed or skipped file gets no patches.
func ApplyTemplate[T any](fsys fs.FS, root *os.Root, tmpl Template[T], config T) error {
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
		generate = Forced()
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
		if err := ApplySections(fsys, root, tmpl, config); err != nil {
			return fmt.Errorf("apply sections: %w", err)
		}
	case len(tmpl.Globs) == 0:
		GetLogger().Warnf("empty template 'globs', skipping '%s' generation", tmpl.Out)
	default:
		GetLogger().Debugf("generating '%s'", tmpl.Out)
		tt, err := newTemplate(path.Base(tmpl.Globs[0]), tmpl.Delimiters)
		if err != nil {
			return fmt.Errorf("new template: %w", err)
		}
		if tt, err = tt.ParseFS(fsys, tmpl.Globs...); err != nil {
			return fmt.Errorf("parse template file(s): %w", err)
		}
		if err := ExecuteTemplate(tt, config, root, tmpl.Out, tmpl.EmptyPolicy, tmpl.Mode); err != nil {
			return fmt.Errorf("template execute: %w", err)
		}
	}

	if err := ApplyPatches(fsys, root, tmpl, config); err != nil {
		return fmt.Errorf("apply patches: %w", err)
	}
	return nil
}

// ExecuteTemplate runs tmpl.Execute with input data and writes the result into given name inside root.
//
// When ExecuteTemplate is called, it rewrites name only when its content changes and always reevaluates its permissions.
//
// The input mode sets the requested file mode for the generated file (e.g. files.RwRR, files.RwxRxRxRx),
// defaulting to files.RwRR when not provided.
//
// The system umask (see files.Umask) is always applied on top (mode &^ umask, no-op on non-compatible platforms).
func ExecuteTemplate(tmpl *template.Template, data any, root *os.Root, name string, policy EmptyPolicy, mode os.FileMode) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("template execution: %w", err)
	}

	if ok := IsEmpty(buf.Bytes(), policy); ok {
		base := filepath.Base(name)
		if _, err := root.Lstat(name); err != nil {
			GetLogger().Debugf("not generating '%s' since it would be empty", base)
			return nil
		}
		GetLogger().Debugf("removing '%s' since it's empty", base)
		if err := root.RemoveAll(name); err != nil {
			return fmt.Errorf("remove '%s': %w", base, err)
		}
		return nil
	}

	if err := root.MkdirAll(filepath.Dir(name), files.RwxRxRxRx); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// apply the requested permissions to name file, honoring the system umask
	requested := mode
	if requested == 0 {
		requested = files.RwRR
	}
	requested &^= files.Umask()

	// leave an unchanged name untouched (no disk write, mtime kept), a read failure falls writes the file as if there was changes
	if existing, err := root.ReadFile(name); err == nil && bytes.Equal(existing, buf.Bytes()) {
		if err := root.Chmod(name, requested); err != nil {
			return fmt.Errorf("chmod: %w", err)
		}
		return nil
	}

	file, err := root.OpenFile(name, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, requested)
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

// baseTemplate holds Sprig and engine default funcs, validated once for the process' lifetime and cloned by newTemplate.
var baseTemplate = sync.OnceValue(func() *template.Template {
	return template.New("").Funcs(sprig.HermeticTxtFuncMap()).Funcs(FuncMap())
})

// newTemplate creates a new template.Template with the provided delimiters
// and all funcs coming from the following places:
//   - Sprig hermetic ones (no environment, network, date or random funcs)
//   - Engine default ones
//   - Engine provided ones with Configure
func newTemplate(name string, delims Delimiters) (*template.Template, error) {
	base, err := baseTemplate().Clone()
	if err != nil {
		return nil, fmt.Errorf("clone base template: %w", err)
	}
	return base.New(name).
		Funcs(funcs()).
		Delims(delims.StartDelim, delims.EndDelim), nil
}
