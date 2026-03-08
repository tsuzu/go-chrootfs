//go:build linux

package chrootfs

import (
	"testing"
)

func TestCheckSupport(t *testing.T) {
	err := CheckSupport()

	// On modern Linux (5.6+), this should return nil
	if err != nil {
		t.Skipf("openat2 with RESOLVE_IN_ROOT not supported: %v", err)
	}

	t.Log("go-chrootfs support check passed")
}
