package chrootfs

import (
	"os"

	"golang.org/x/sys/unix"
)

// IsSupported checks if the current OS and kernel support go-chrootfs functionality.
//
// This function checks whether the Linux kernel version is 5.6 or later,
// which is required for openat2 with RESOLVE_IN_ROOT support.
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
	var uname unix.Utsname
	if err := unix.Uname(&uname); err != nil {
		return false
	}

	// Parse kernel version from uname
	// Release format is typically "5.15.0-91-generic" or similar
	release := string(uname.Release[:])
	// Find the null terminator
	for i, c := range uname.Release {
		if c == 0 {
			release = string(uname.Release[:i])
			break
		}
	}

	major, minor, err := parseKernelVersion(release)
	if err != nil {
		// If parsing fails, fall back to actual syscall test
		return isSupportedBySyscall()
	}

	// Check if kernel is >= 5.6
	// openat2 was introduced in Linux 5.6
	if major > 5 || (major == 5 && minor >= 6) {
		return true
	}

	return false
}

// parseKernelVersion parses a kernel version string like "5.15.0-91-generic"
// and returns the major and minor version numbers.
func parseKernelVersion(version string) (major, minor int, err error) {
	// Find the first two numbers separated by dots
	var n int
	for i := 0; i < len(version); i++ {
		if version[i] >= '0' && version[i] <= '9' {
			n = n*10 + int(version[i]-'0')
		} else if version[i] == '.' {
			if major == 0 {
				major = n
				n = 0
			} else {
				minor = n
				return major, minor, nil
			}
		} else {
			// Stop at first non-digit, non-dot character
			if major > 0 {
				minor = n
				return major, minor, nil
			}
		}
	}
	// Handle case where we reached end of string
	if major > 0 {
		minor = n
		return major, minor, nil
	}
	return 0, 0, &os.PathError{Op: "parse_version", Path: version, Err: os.ErrInvalid}
}

// isSupportedBySyscall performs an actual syscall to test support.
// This is used as a fallback when kernel version parsing fails.
func isSupportedBySyscall() bool {
	fd, err := unix.Open(".", unix.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		return false
	}
	defer unix.Close(fd)

	var how unix.OpenHow
	how.Flags = unix.O_RDONLY | unix.O_PATH
	how.Resolve = unix.RESOLVE_IN_ROOT | unix.RESOLVE_NO_MAGICLINKS

	testFd, err := unix.Openat2(fd, ".", &how)
	if err != nil {
		return false
	}
	unix.Close(testFd)

	return true
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

// CheckSupport returns a detailed error if go-chrootfs is not supported.
//
// This function is similar to IsSupported() but provides more context about
// why the feature is not available.
//
// Returns nil if supported, otherwise returns an error describing the issue.
//
// Example:
//
//	if err := chrootfs.CheckSupport(); err != nil {
//	    log.Printf("go-chrootfs not available: %v", err)
//	    // Use fallback
//	}
func CheckSupport() error {
	fd, err := unix.Open(".", unix.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		return &os.PathError{
			Op:   "check_support",
			Path: ".",
			Err:  err,
		}
	}
	defer unix.Close(fd)

	var how unix.OpenHow
	how.Flags = unix.O_RDONLY | unix.O_PATH
	how.Resolve = unix.RESOLVE_IN_ROOT | unix.RESOLVE_NO_MAGICLINKS

	testFd, err := unix.Openat2(fd, ".", &how)
	if err != nil {
		return &os.PathError{
			Op:   "openat2",
			Path: ".",
			Err:  err,
		}
	}
	unix.Close(testFd)

	return nil
}
