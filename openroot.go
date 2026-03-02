package chrootfs

import (
	"io/fs"
	"os"
)

// OpenRoot opens a subdirectory within the given root and returns it as a new root.
//
// This function provides a uniform interface for creating sub-roots that works with
// both os.Root (Go 1.24+) and Chroot implementations.
//
// For Chroot, it calls the OpenRoot method and returns a RootLike.
// For os.Root, it calls the OpenRoot method and returns the *os.Root directly.
//
// Example with os.Root:
//
//	osRoot, _ := os.OpenRoot("/data")
//	defer osRoot.Close()
//
//	subRoot, _ := chrootfs.OpenRoot(osRoot, "configs")
//	defer subRoot.Close()
//
// Example with Chroot:
//
//	chroot, _ := chrootfs.New("/data")
//	defer chroot.Close()
//
//	subRoot, _ := chrootfs.OpenRoot(chroot, "configs")
//	defer subRoot.Close()
//
// The returned RootLike is independent of the parent. Closing the parent does not
// affect the child.
func OpenRoot(root RootLike, name string) (RootLike, error) {
	// Try RootLikeWithOpenRoot first (Chroot)
	if rootWithOpen, ok := root.(RootLikeWithOpenRoot); ok {
		return rootWithOpen.OpenRoot(name)
	}

	// Check if it's os.Root (has OpenRoot method that returns *os.Root)
	// We use type assertion with a local interface to avoid direct dependency
	type osRootLike interface {
		RootLike
		OpenRoot(string) (*os.Root, error)
	}

	if osRoot, ok := root.(osRootLike); ok {
		// os.Root already implements all RootLike methods, so we can return it directly
		return osRoot.OpenRoot(name)
	}

	// Unsupported root type
	return nil, &fs.PathError{
		Op:   "openroot",
		Path: name,
		Err:  fs.ErrInvalid,
	}
}
