/*
Package generator exposes a bunch of functions to be wrapped with generate.Generator function signature.

Examples:

	type config struct { ... }

	func GeneratorGitignore(ctx context.Context, destdir string, c config) error {
		return generator.DownloadGitignore(ctx, cleanhttp.DefaultClient(), filepath.Join(destdir, generator.FileGitignore), "java", "linux")
	}

	var _ generate.Generator[config] = GeneratorGitignore // ensure interface is implemented

	// single generator call
	func main() {
		var c config
		err := generator.GeneratorGitignore(ctx, "path/to/dir", c)
		// handle err
	}

	// fully used with engine.Generate
	func main() {
		destdir, _ := os.Getwd()

		var c config
		err := engine.Generate(ctx, destdir, c, ..., []engine.Generator[config]{GeneratorGitignore})
		// handle err
	}
*/
package generator
