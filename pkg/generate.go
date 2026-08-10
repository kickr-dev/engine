package engine

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

// ErrFailedGeneration is returned when at least one file couldn't be properly generated.
//
// Every generation error is logged during processing to avoid a big aggregated error at the end.
var ErrFailedGeneration = errors.New("some error(s) occurred during generation")

// Generate is the main function from generate package.
// It takes a configuration and various options.
//
// It executes all parsers given in options (or default ones), in order,
// and then runs all provided generators concurrently (bounded to runtime.GOMAXPROCS(0)) to apply or remove templates.
func Generate[T any](ctx context.Context, destdir string, config T, parsers []Parser[T], generators []Generator[T]) error {
	// parse repository
	errs := make([]error, 0, len(parsers))
	for _, parser := range parsers {
		errs = append(errs, parser(ctx, destdir, &config))
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}

	// execute generators concurrently
	var group errgroup.Group
	group.SetLimit(runtime.GOMAXPROCS(0))

	var failed atomic.Bool
	for _, generator := range generators {
		group.Go(func() error {
			if err := generator(ctx, destdir, config); err != nil {
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
