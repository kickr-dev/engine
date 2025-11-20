package parser

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

// FileGowork represents the go.work filename.
const FileGowork = "go.work"

// ErrNoGowork is a specific case of fs.ErrNotExist when go.work file doesn't exist in ReadGowork call.
var ErrNoGowork = errors.New("no go.work file")

// Gowork represents the parsed struct for go.work file.
type Gowork struct {
	// Go is the go statement,
	// i.e. "go 1.23.4" without "go" (and space) part.
	Go string

	// Toolchain is the toolchain statement,
	// i.e. "toolchain go1.23.4" without "toolchain go" part.
	Toolchain string

	// Uses is the slice of all use in go.work file.
	Uses []GoworkUse
}

// Module returns the common prefix between all 'use' of current go.work.
//
// In case there's no 'use' at all or no common prefix can be found between them, then an empty string is returned.
func (g Gowork) Module() string {
	if len(g.Uses) == 0 {
		return ""
	}

	prefixes := strings.Split(g.Uses[0].Gomod.Module, "/")
	for _, use := range g.Uses[1:] {
		parts := strings.Split(use.Gomod.Module, "/")

		var i int
		for i < len(prefixes) && i < len(parts) && prefixes[i] == parts[i] {
			i++
		}

		prefixes = prefixes[:i]
		if len(prefixes) == 0 {
			return ""
		}
	}
	return strings.Join(prefixes, "/")
}

// GoworkUse is one use of a go.work file.
// It contains its valid parsed go.mod and it's use path.
type GoworkUse struct {
	// Gomod is the parsed go.mod of current go.work use element.
	Gomod Gomod

	// Module path in the go.work.
	ModulePath string

	// Use path of module.
	Use string
}

// ReadGowork reads the go.work file at destdir
// and returns its representation alongside all go.mod that could be defined in 'use' directive.
//
// It will return an error if the go.work file is missing the 'go' statement.
//
// It can be used as a simple function, calling it directly,
// but can also be used as its expected usage with engine.Generate:
//
//	type config struct { ... }
//
//	func Golang(ctx context.Context, destdir string, c *config) error {
//		gowork, err := parser.ReadGowork(destdir)
//		if err == nil {
//			engine.GetLogger().Infof("golang detected, file '%s' is present and valid", parser.FileGowork)
//			// do something with gowork (e.g. update config since it's a pointer)
//			return nil
//		}
//		// fs.ErrNotExist is also valid
//		// however here it won't handle in case at least one 'use' go.mod file doesn't exist
//		if !errors.Is(err, parser.ErrNoGowork) {
//			return fmt.Errorf("read go.work: %w", err)
//		}
//		// go.work doesn't exist, maybe it's a simple Go repository with a go.mod
//		// let's check that just below
//
//		gomod, err := parser.ReadGomod(destdir)
//		if err != nil {
//			if errors.Is(err, fs.ErrNotExist) {
//				return nil
//			}
//			return fmt.Errorf("read go.mod: %w", err)
//		}
//		engine.GetLogger().Infof("golang detected, file '%s' is present and valid", parser.FileGomod)
//		// do something with gomod (e.g. update config since it's a pointer)
//		return nil
//	}
func ReadGowork(destdir string) (Gowork, error) {
	workpath := filepath.Join(destdir, FileGowork)

	// read go.mod
	content, err := os.ReadFile(workpath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Gowork{}, fmt.Errorf("%w: %w", ErrNoGowork, err)
		}
		return Gowork{}, fmt.Errorf("read file: %w", err)
	}

	// parse go.work into it's modfile representation
	file, err := modfile.ParseWork(FileGowork, content, nil)
	if err != nil {
		return Gowork{}, fmt.Errorf("parse modfile: %w", err)
	}
	var gowork Gowork

	// parse go statement
	if file.Go == nil {
		return Gowork{}, ErrMissingGoStatement
	}
	gowork.Go = file.Go.Version
	if file.Toolchain != nil {
		gowork.Toolchain = file.Toolchain.Name[2:]
	}

	var (
		errs = make([]error, 0, len(file.Use))
		uses = make([]GoworkUse, 0, len(file.Use))
	)
	for _, use := range file.Use {
		gomod, err := ReadGomod(filepath.Join(destdir, use.Path))
		if err != nil {
			errs = append(errs, fmt.Errorf("read gomod in '%s': %w", use.Path, err))
			continue
		}
		uses = append(uses, GoworkUse{Gomod: gomod, ModulePath: use.ModulePath, Use: use.Path})
	}
	if err := errors.Join(errs...); err != nil {
		return Gowork{}, err // already wrapped
	}

	gowork.Uses = uses
	return gowork, nil
}
