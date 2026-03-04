package chrootfs

import "os"

// Capabilities describes the features supported by a RootLike implementation.
type Capabilities struct {
	// SupportsOpenRoot indicates whether the implementation supports creating sub-roots.
	// This is true for Chroot and os.Root (Go 1.24+).
	SupportsOpenRoot bool

	// IsOSRoot indicates whether the implementation is os.Root from the standard library.
	IsOSRoot bool

	// IsChrootFS indicates whether the implementation is from this package.
	IsChrootFS bool
}

// GetCapabilities returns the capabilities of the given RootLike implementation.
//
// This function uses type assertions to determine which features are supported
// by the implementation, without requiring direct dependencies on concrete types.
//
// Example:
//
//	caps := chrootfs.GetCapabilities(root)
//	if caps.SupportsOpenRoot {
//	    subRoot, err := chrootfs.OpenRoot(root, "subdir")
//	    // ...
//	}
func GetCapabilities(root RootLike) Capabilities {
	caps := Capabilities{}

	// Check if it supports OpenRoot
	if _, ok := root.(RootLikeWithOpenRoot); ok {
		caps.SupportsOpenRoot = true
		caps.IsChrootFS = true
		return caps
	}

	// Check if it's os.Root (has OpenRoot method that returns *os.Root)
	type osRootLike interface {
		RootLike
		OpenRoot(string) (*os.Root, error)
	}

	if _, ok := root.(osRootLike); ok {
		caps.SupportsOpenRoot = true
		caps.IsOSRoot = true
		return caps
	}

	return caps
}

// SupportsOpenRoot checks if the given RootLike implementation supports creating sub-roots.
//
// This is a convenience function that returns true if the implementation has an OpenRoot method,
// which is present in both Chroot and os.Root (Go 1.24+).
//
// Example:
//
//	if chrootfs.SupportsOpenRoot(root) {
//	    subRoot, err := chrootfs.OpenRoot(root, "configs")
//	    // ...
//	}
func SupportsOpenRoot(root RootLike) bool {
	return GetCapabilities(root).SupportsOpenRoot
}
