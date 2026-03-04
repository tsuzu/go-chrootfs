//go:build linux

package chrootfs

import (
	"os"
	"testing"
)

func TestIsSupported(t *testing.T) {
	// On Linux with kernel 5.6+, this should return true
	// On older kernels or non-Linux, it should return false
	supported := IsSupported()

	// We're running tests on Linux, so we expect it to be supported
	// (CI uses ubuntu-latest which has kernel 5.6+)
	if !supported {
		t.Skip("openat2 with RESOLVE_IN_ROOT not supported on this system")
	}

	t.Log("go-chrootfs is supported on this system")
}

func TestCheckSupport(t *testing.T) {
	err := CheckSupport()

	// On modern Linux (5.6+), this should return nil
	if err != nil {
		t.Skipf("openat2 with RESOLVE_IN_ROOT not supported: %v", err)
	}

	t.Log("go-chrootfs support check passed")
}

func TestMustBeSupportedDoesNotPanic(t *testing.T) {
	// This test verifies that MustBeSupported doesn't panic on supported systems
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("MustBeSupported panicked (system not supported): %v", r)
		}
	}()

	MustBeSupported()
	t.Log("MustBeSupported() did not panic")
}

func TestIsSupportedConsistency(t *testing.T) {
	// IsSupported and CheckSupport should be consistent
	isSupported := IsSupported()
	checkErr := CheckSupport()

	if isSupported && checkErr != nil {
		t.Errorf("inconsistent results: IsSupported()=true but CheckSupport()=%v", checkErr)
	}

	if !isSupported && checkErr == nil {
		t.Error("inconsistent results: IsSupported()=false but CheckSupport()=nil")
	}
}

func TestNewRequiresSupport(t *testing.T) {
	if !IsSupported() {
		t.Skip("openat2 not supported on this system")
	}

	// If IsSupported() returns true, New() should work
	root := t.TempDir()
	c, err := New(root)
	if err != nil {
		t.Fatalf("New() failed despite IsSupported()=true: %v", err)
	}
	defer c.Close()

	// Verify basic functionality
	_, err = c.Stat(".")
	if err != nil {
		t.Fatalf("Stat() failed: %v", err)
	}
}

func TestParseKernelVersion(t *testing.T) {
	tests := []struct {
		version string
		major   int
		minor   int
		wantErr bool
	}{
		{"5.15.0-91-generic", 5, 15, false},
		{"6.8.0-90-generic", 6, 8, false},
		{"5.6.0", 5, 6, false},
		{"5.6", 5, 6, false},
		{"4.19.0", 4, 19, false},
		{"6.10.1-amd64", 6, 10, false},
		{"invalid", 0, 0, true},
		{"", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			major, minor, err := parseKernelVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseKernelVersion(%q) error = %v, wantErr %v", tt.version, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if major != tt.major || minor != tt.minor {
					t.Errorf("parseKernelVersion(%q) = (%d, %d), want (%d, %d)", tt.version, major, minor, tt.major, tt.minor)
				}
			}
		})
	}
}

func TestFallbackPattern(t *testing.T) {
	// This test demonstrates the recommended fallback pattern
	root := t.TempDir()

	var fsRoot interface {
		ReadFile(string) ([]byte, error)
		Close() error
	}

	if IsSupported() {
		c, err := New(root)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		fsRoot = c
		t.Log("Using go-chrootfs")
	} else {
		osRoot, err := os.OpenRoot(root)
		if err != nil {
			t.Fatalf("os.OpenRoot() failed: %v", err)
		}
		fsRoot = osRoot
		t.Log("Falling back to os.Root")
	}
	defer fsRoot.Close()

	// Write test file
	testFile := root + "/test.txt"
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Both should work
	data, err := fsRoot.ReadFile("test.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != "test" {
		t.Fatalf("unexpected content: %q", string(data))
	}
}
