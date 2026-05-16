package files

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

const (
	// Rw represents a file permission of read/write for current user
	// and no access for user's group and other groups.
	Rw fs.FileMode = 0o600

	// RwRR represents a file permission of read/write for current user
	// and read-only access for user's group and other groups.
	RwRR fs.FileMode = 0o644

	// RwRwRw represents a file permission of read/write for current user
	// and read/write too for user's group and other groups.
	RwRwRw fs.FileMode = 0o666

	// RwxRxRxRx represents a file permission of read/write/execute for current user
	// and read/execute for user's group and other groups.
	RwxRxRxRx fs.FileMode = 0o755
)

// Exists returns a boolean indicating whether the provided input src can be stat'ed or not.
func Exists(src string) bool {
	_, err := os.Stat(src)
	return err == nil
}

// GlobOption represents a function that can be giving when calling Glob to add specific behaviors.
type GlobOption func(o globOptions) globOptions

// GlobExcludedDirectories returns a GlobOption which adds excluded directories from Glob checking.
//
// Excluded directories are apply to all directories levels (root and subdirectories) during checking.
func GlobExcludedDirectories(dirs ...string) GlobOption {
	return func(o globOptions) globOptions {
		o.ExcludedDirectories = dirs
		return o
	}
}

// GlobExcludedFiles returns a GlobOption which adds excluded files from Glob checking.
//
// Excluded files are applied to all directories levels (root and subdirectories) during checking.
func GlobExcludedFiles(files ...string) GlobOption {
	return func(o globOptions) globOptions {
		o.ExcludedFiles = files
		return o
	}
}

type globOptions struct {
	ExcludedDirectories []string
	ExcludedFiles       []string
}

func newGlobOptions(opts ...GlobOption) globOptions {
	var gopts globOptions
	for _, opt := range opts {
		gopts = opt(gopts)
	}
	return gopts
}

// Glob returns all matching files for the input glob and root (and its subdirectories).
//
// In case root directory doesn't exist, no matches are returned (error is silenced).
func Glob(root, glob string, opts ...GlobOption) (matches []string) {
	gopts := newGlobOptions(opts...)
	if slices.Contains(gopts.ExcludedDirectories, filepath.Base(root)) {
		return nil
	}

	globs, _ := filepath.Glob(filepath.Join(root, glob))
	for _, match := range globs {
		if !slices.Contains(gopts.ExcludedFiles, filepath.Base(match)) {
			matches = append(matches, match)
		}
	}

	entries, _ := os.ReadDir(root)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		matches = append(matches, Glob(filepath.Join(root, entry.Name()), glob, opts...)...)
	}
	return matches
}
