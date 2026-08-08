package files

import (
	"os"
	"sync"
)

// umask memorizes systemUmask for the process' lifetime.
//
// Use Umask function to access the system umask, saving this global variable from replacement.
var umask = sync.OnceValue(systemUmask)

// Umask returns the running process' umask,
// computed once for the process' lifetime on first call.
func Umask() os.FileMode {
	return umask()
}
