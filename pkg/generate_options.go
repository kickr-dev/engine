package engine

import (
	"context"
	"sync/atomic"
	"text/template"
)

// Options holds the settings of a generation run.
//
// They travel with the context given to Generate (see ContextWithOptions),
// and a context carrying none generates with the zero value, the defaults.
type Options struct {
	// Force regenerates the files under PolicyDefault even when they were edited by hand,
	// while removed and skipped files still follow their policy.
	Force bool

	// Funcs is added to every template, on top of the Sprig and engine funcs (see FuncMap).
	Funcs template.FuncMap

	// StripMarkers drops the generated notice lines from every generated file,
	// along with its template's section marker lines (see Template.Sections).
	//
	// Such a file isn't owned by generation anymore (see IsGenerated),
	// so later runs only regenerate it when Force is set, and its sections can't be regenerated alone.
	StripMarkers bool
}

type optionsKey struct{}

// ContextWithOptions returns a copy of ctx carrying opts, for Generate and the template functions to read them.
func ContextWithOptions(ctx context.Context, opts Options) context.Context {
	return context.WithValue(ctx, optionsKey{}, opts)
}

// OptionsFromContext returns the Options carried by ctx, or the zero Options when it carries none.
func OptionsFromContext(ctx context.Context) Options {
	if opts, ok := ctx.Value(optionsKey{}).(Options); ok {
		return opts
	}
	return Options{}
}

// SetLogger sets the logger used across the engine, a nil one falling back to a no-op logger.
func SetLogger(logger Logger) {
	next := logger
	if next == nil {
		next = &noopLogger{}
	}
	l.Store(&next)
}

// GetLogger returns the logger given to SetLogger, or a no-op logger when none was set.
func GetLogger() Logger {
	logger := l.Load()
	if logger == nil {
		return &noopLogger{}
	}
	return *logger
}

var l atomic.Pointer[Logger]
