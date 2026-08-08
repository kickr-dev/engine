//go:build darwin || freebsd || linux || netbsd || openbsd

package files

import (
	"os"
	"syscall"
)

// systemUmask reads the running process' umask without permanently changing it.
//
// Since syscall.Umask has no side-effect-free "read" mode,
// set-then-restore is the standard Go idiom, same as used by cmd/go/internal/toolchain/umask_unix.go.
func systemUmask() os.FileMode {
	old := syscall.Umask(0)
	syscall.Umask(old)
	return os.FileMode(old) //nolint:gosec // umask is always within [0,0o777], no overflow risk
}
