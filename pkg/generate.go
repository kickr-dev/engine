package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

// ErrFailedGeneration is returned when at least one file couldn't be properly generated.
//
// Every generation error is logged during processing to avoid a big aggregated error at the end.
var ErrFailedGeneration = errors.New("some error(s) occurred during generation")

// Generate is the main function of the engine package.
// It takes a configuration, parsers and generators.
//
// It executes all given parsers, in order,
// and then runs all provided generators concurrently to apply or remove templates.
func Generate[T any](ctx context.Context, destdir string, config T, parsers []Parser[T], generators []Generator[T]) error {
	// parse repository
	errs := make([]error, 0, len(parsers))
	for _, parser := range parsers {
		errs = append(errs, parser(ctx, destdir, &config))
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}

	root, err := os.OpenRoot(destdir)
	if err != nil {
		return fmt.Errorf("open root: %w", err)
	}
	defer root.Close()

	// execute generators concurrently
	var (
		failed atomic.Bool
		group  errgroup.Group
	)
	for _, generator := range generators {
		group.Go(func() error {
			if err := generator(ctx, root, config); err != nil {
				if !errors.Is(err, ErrFailedGeneration) {
					GetLogger().Errorf("%s", err.Error())
				}
				failed.Store(true)
			}
			return nil
		})
	}
	_ = group.Wait() // generator errors are logged individually above, never returned by group.Go

	if failed.Load() {
		return ErrFailedGeneration
	}
	return nil
}
