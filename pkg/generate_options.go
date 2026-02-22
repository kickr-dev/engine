package engine

// OptionFunc is the function signature for engine options to be provided in Configure.
type OptionFunc func(*options)

// WithLogger sets the engine logger when calling Configure with this option.
//
// In case the logger is nil, the logger won't be set / updated.
func WithLogger(logger Logger) OptionFunc {
	return func(o *options) {
		o.logger = logger
	}
}

// WithForce sets the engine force option when calling Configure with this option.
//
// The force option can be then used to force generation inside Generator[T].
// The option is by default used in GeneratorTemplates within ShouldGenerate.
func WithForce(force bool) OptionFunc {
	return func(o *options) {
		o.force = force
	}
}

// GetLogger returns global logger if it exists or a noop logger.
func GetLogger() Logger {
	if o.logger == nil {
		return &noopLogger{}
	}
	return o.logger
}

// Forced returns truthy if the options' force is provided.
//
// It means that generation should be forced (applied by default in GeneratorTemplates within ShouldGenerate, but must be used manually when writing own Generator[T]).
func Forced() bool {
	return o.force
}

// Configure applies the options functions to the global option variable (unexported).
//
// This function should be called before calling any function within engine package in case a specific logger must be set
// or generation must be forced.
func Configure(opts ...OptionFunc) {
	for _, opt := range opts {
		opt(&o)
	}
	if o.logger == nil {
		o.logger = &noopLogger{}
	}
}

var o options

type options struct {
	force  bool
	logger Logger
}
