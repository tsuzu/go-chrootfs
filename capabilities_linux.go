//go:build linux

package chrootfs

import (
	"os"

	"golang.org/x/sys/unix"
)

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
