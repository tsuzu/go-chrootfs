package chrootfs

// IsSupported checks if the current OS and kernel support go-chrootfs functionality.
//
// This function tests whether openat2 with RESOLVE_IN_ROOT is available by
// performing an actual syscall. This is necessary because some distributions
// (e.g., RHEL) may backport features to older kernel version numbers.
//
// Returns true if go-chrootfs can be used on the current system, false otherwise.
//
// Example:
//
//	if chrootfs.IsSupported() {
//	    root, err := chrootfs.New("/sandbox")
//	    // ...
//	} else {
//	    // Fallback to os.Root or other mechanism
//	    root, err := os.OpenRoot("/sandbox")
//	    // ...
//	}
func IsSupported() bool {
	return CheckSupport() == nil
}

// MustBeSupported panics if go-chrootfs is not supported on the current system.
//
// This is useful for applications that absolutely require RESOLVE_IN_ROOT semantics
// and cannot fall back to alternative implementations.
//
// Example:
//
//	func init() {
//	    chrootfs.MustBeSupported()
//	}
func MustBeSupported() {
	if !IsSupported() {
		panic("go-chrootfs requires Linux kernel 5.6+ with openat2 and RESOLVE_IN_ROOT support")
	}
}
