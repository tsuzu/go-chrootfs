//go:build !linux

package chrootfs

// CheckSupport always returns an error on non-Linux platforms: openat2 with
// RESOLVE_IN_ROOT is Linux-only.
func CheckSupport() error {
	return errUnsupported
}
