//go:build linux

package chrootfs

import (
	"io/fs"
	"os"
	"testing"
)

func TestGetCapabilitiesWithChroot(t *testing.T) {
	root := t.TempDir()
	c := newChroot(t, root)
	defer c.Close()

	caps := GetCapabilities(c)

	if !caps.SupportsOpenRoot {
		t.Error("expected Chroot to support OpenRoot")
	}
	if !caps.IsChrootFS {
		t.Error("expected IsChrootFS to be true for Chroot")
	}
	if caps.IsOSRoot {
		t.Error("expected IsOSRoot to be false for Chroot")
	}
}

func TestGetCapabilitiesWithOSRoot(t *testing.T) {
	root := t.TempDir()
	osRoot, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer osRoot.Close()

	caps := GetCapabilities(osRoot)

	if !caps.SupportsOpenRoot {
		t.Error("expected os.Root to support OpenRoot")
	}
	if caps.IsChrootFS {
		t.Error("expected IsChrootFS to be false for os.Root")
	}
	if !caps.IsOSRoot {
		t.Error("expected IsOSRoot to be true for os.Root")
	}
}

func TestSupportsOpenRootWithChroot(t *testing.T) {
	root := t.TempDir()
	c := newChroot(t, root)
	defer c.Close()

	if !SupportsOpenRoot(c) {
		t.Error("expected Chroot to support OpenRoot")
	}
}

func TestSupportsOpenRootWithOSRoot(t *testing.T) {
	root := t.TempDir()
	osRoot, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer osRoot.Close()

	if !SupportsOpenRoot(osRoot) {
		t.Error("expected os.Root to support OpenRoot")
	}
}

// MockRootLike is a wrapped RootLike implementation that embeds os.Root
// Since it embeds *os.Root, it inherits the OpenRoot method and will be detected as os.Root
type MockRootLike struct {
	*os.Root
}

func (m *MockRootLike) FS() fs.FS {
	return m.Root.FS()
}

func TestGetCapabilitiesWithMockRootLike(t *testing.T) {
	root := t.TempDir()
	osRoot, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer osRoot.Close()

	mock := &MockRootLike{Root: osRoot}
	caps := GetCapabilities(mock)

	// MockRootLike embeds *os.Root, so it inherits OpenRoot method
	// and will be detected as supporting OpenRoot and being os.Root-like
	if !caps.SupportsOpenRoot {
		t.Error("expected MockRootLike to support OpenRoot (inherited from os.Root)")
	}
	if caps.IsChrootFS {
		t.Error("expected IsChrootFS to be false for MockRootLike")
	}
	if !caps.IsOSRoot {
		t.Error("expected IsOSRoot to be true for MockRootLike (inherited from os.Root)")
	}
}

func TestCapabilitiesDrivenUsage(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(root+"/subdir", 0o755); err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name string
		root RootLike
	}{
		{
			name: "Chroot",
			root: func() RootLike {
				c := newChroot(t, root)
				t.Cleanup(func() { c.Close() })
				return c
			}(),
		},
		{
			name: "os.Root",
			root: func() RootLike {
				r, err := os.OpenRoot(root)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { r.Close() })
				return r
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Check capabilities and use them to determine behavior
			if SupportsOpenRoot(tc.root) {
				subRoot, err := OpenRoot(tc.root, "subdir")
				if err != nil {
					t.Fatalf("OpenRoot failed: %v", err)
				}
				defer subRoot.Close()

				// Verify the sub-root works
				if subRoot.Name() == "" {
					t.Error("expected non-empty Name() from sub-root")
				}
			} else {
				t.Error("expected root to support OpenRoot")
			}
		})
	}
}
