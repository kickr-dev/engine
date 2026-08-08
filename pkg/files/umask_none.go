//go:build !(darwin || freebsd || linux || netbsd || openbsd)

package files

import "os"

// systemUmask always returns 0 on platforms with no umask concept (Windows, plan9, js/wasm, ...).
//
// Returning a fixed 0 on those platforms helps not distorting caller's requested mode (mode &^ 0 == mode).
func systemUmask() os.FileMode {
	return 0
}
