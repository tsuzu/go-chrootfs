// Package chrootfs provides secure filesystem access within a directory tree.
//
// # RootLike API
//
// The Chroot type implements a RootLike interface similar to os.Root (Go 1.24+),
// providing operations like Open, Create, Mkdir, Remove, etc. Unlike os.Root,
// which uses platform-specific implementations, Chroot uses Linux openat2 for
// kernel-level path resolution security.
//
// Example:
//
//	root, err := chrootfs.New("/sandbox")
//	if err != nil {
//	    return err
//	}
//	defer root.Close()
//
//	// Absolute paths are allowed and resolved within the root
//	f, err := root.Open("/etc/config.json")
//
//	// Create and write files
//	err = root.WriteFile("/data/output.txt", []byte("hello"), 0o644)
//
// # io/fs.FS View
//
// Call FS() to get an io/fs.FS view for use with standard library functions:
//
//	root, _ := chrootfs.New("/data")
//	fsys := root.FS()
//
//	// Use with standard library
//	data, _ := fs.ReadFile(fsys, "config.json")
//	http.Handle("/files/", http.FileServer(http.FS(fsys)))
//
// The fs.FS view follows io/fs conventions (no absolute paths, read-only Open).
//
// # Security
//
// All operations use openat2(RESOLVE_IN_ROOT | RESOLVE_NO_MAGICLINKS),
// preventing escape via:
//   - Symbolic links (resolved within root)
//   - ".." components (cannot escape root)
//   - Magic links like /proc/self/fd/*
//   - Absolute symlink targets (treated as relative to root)
//
// Note: This does not prevent escape via bind mounts, which require root privileges.
//
// # Platform Support
//
// Requires Linux kernel 5.6+ with openat2 support. On other platforms,
// the package compiles but all operations return an error.
package chrootfs
