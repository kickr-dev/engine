package engine

import (
	"context"
	"io/fs"
	"os"
	"path"
)

// GeneratorTemplates is a simple generator taking as input a filesystem and all templates to apply.
//
// Errors encountered during templates generation are logged, in that case a final error being ErrFailedGeneration is returned.
func GeneratorTemplates[T any](fsys fs.FS, templates []Template[T]) Generator[T] {
	return func(_ context.Context, root *os.Root, config T) error {
		var errcount int
		for _, tmpl := range templates {
			if err := ApplyTemplate(fsys, root, tmpl, config); err != nil {
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

	return func(ctx context.Context, root *os.Root, config T) error {
		var failed bool
		for _, module := range modules(config) {
			sub, err := root.OpenRoot(module.Dir())
			if err != nil {
				failed = true
				GetLogger().Errorf("failed to open module '%s': %v", module.Dir(), err)
				continue
			}
			if err := generator(ctx, sub, module); err != nil {
				failed = true
				GetLogger().Errorf("failed to generate '%s': %v", module.Dir(), err)
			}
			if err := sub.Close(); err != nil {
				GetLogger().Warnf("failed to close module '%s': %v", module.Dir(), err)
			}
		}
		if failed {
			return ErrFailedGeneration
		}
		return nil
	}
}
