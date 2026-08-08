/*
Package parser provides a bunch of functions to be wrapped with generate.Parser function signature.

# Example

	type config struct { ... }

	func CustomParser(ctx context.Context, destdir string, c *config) error {
		...
	}

	var _ generate.Parser[config] = CustomParser // ensure interface is implemented

	// single parser call
	func main() {
		var c config
		err := CustomParser(ctx, "path/to/dir", &c)
		// handle err
	}

	// fully used with engine.Generate
	func main() {
		destdir, _ := os.Getwd()

		var c config
		err := engine.Generate(ctx, destdir, c, []engine.Parser[config]{CustomParser}, ...)
		// handle err
	}
*/
package parser
