package engine

import (
	"sync/atomic"
	"text/template"
)

// OptionFunc is the function signature for engine options to be provided in Configure.
type OptionFunc func(options) options

// WithLogger sets the engine logger when calling Configure with this option.
//
// A nil logger falls back to a no-op logger (see GetLogger).
func WithLogger(logger Logger) OptionFunc {
	return func(o options) options {
		o.logger = logger
		return o
	}
}

// WithForce sets the engine force option when calling Configure with this option.
//
// The force option can be then used to force generation inside Generator[T].
// The option is by default used in GeneratorTemplates within ShouldGenerate.
func WithForce(force bool) OptionFunc {
	return func(o options) options {
		o.force = force
		return o
	}
}

// WithFuncMap adds a custom FuncMap to all templating calls.
func WithFuncMap(funcs template.FuncMap) OptionFunc {
	return func(o options) options {
		o.funcs = funcs
		return o
	}
}

// GetLogger returns global logger if it exists or a noop logger.
func GetLogger() Logger {
	opts := o.Load()
	if opts == nil || opts.logger == nil {
		return &noopLogger{}
	}
	return opts.logger
}

// Forced returns truthy if the options' force is provided.
//
// It means that generation should be forced (applied by default in GeneratorTemplates within ShouldGenerate, but must be used manually when writing own Generator[T]).
func Forced() bool {
	opts := o.Load()
	return opts != nil && opts.force
}

// funcs returns the configured custom FuncMap, if any.
func funcs() template.FuncMap {
	opts := o.Load()
	if opts == nil {
		return nil
	}
	return opts.funcs
}

// Configure applies the options functions to the global option variable (unexported).
//
// This function should be called before calling any function within engine package in case a specific logger must be set
// or generation must be forced.
//
// Configure should be called only once per process before Generate call.
// Each call fully replaces the previously configured options rather than merging with them.
func Configure(opts ...OptionFunc) {
	next := options{}
	for _, opt := range opts {
		next = opt(next)
	}
	if next.logger == nil {
		next.logger = &noopLogger{}
	}
	o.Store(&next)
}

var o atomic.Pointer[options]

type options struct {
	force  bool
	funcs  template.FuncMap
	logger Logger
}
