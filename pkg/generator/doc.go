/*
Package generator exposes a bunch of functions to be wrapped with generate.Generator function signature.

# Example

	type config struct { ... }

	func CustomGenerator(ctx context.Context, destdir string, c config) error {
		...
	}

	var _ generate.Generator[config] = CustomGenerator // ensure interface is implemented

	// single generator call
	func main() {
		var c config
		err := CustomGenerator(ctx, "path/to/dir", c)
		// handle err
	}

	// fully used with engine.Generate
	func main() {
		destdir, _ := os.Getwd()

		var c config
		err := engine.Generate(ctx, destdir, c, ..., []engine.Generator[config]{CustomGenerator})
		// handle err
	}
*/
package generator
