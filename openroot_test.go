//go:build linux

package chrootfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChrootOpenRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub/nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "root.txt"), []byte("root"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub/sub.txt"), []byte("sub"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub/nested/nested.txt"), []byte("nested"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := newChroot(t, root)

	// Test opening subdirectory
	subRoot, err := c.OpenRoot("sub")
	if err != nil {
		t.Fatal(err)
	}
	defer subRoot.Close()

	// Should see sub.txt
	data, err := subRoot.ReadFile("sub.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "sub" {
		t.Fatalf("unexpected content: %q", string(data))
	}

	// Should NOT see root.txt (it's outside the sub root)
	_, err = subRoot.ReadFile("root.txt")
	if err == nil {
		t.Fatal("expected error accessing parent file")
	}

	// Should see nested directory
	_, err = subRoot.Stat("nested")
	if err != nil {
		t.Fatalf("expected to see nested dir: %v", err)
	}

	// Test opening nested directory
	nestedRoot, err := subRoot.(RootLikeWithOpenRoot).OpenRoot("nested")
	if err != nil {
		t.Fatal(err)
	}
	defer nestedRoot.Close()

	data, err = nestedRoot.ReadFile("nested.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "nested" {
		t.Fatalf("unexpected content: %q", string(data))
	}
}

func TestChrootOpenRootWithAbsolutePath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs/app.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := newChroot(t, root)

	// Test with absolute path (should be converted to relative)
	subRoot, err := c.OpenRoot("/configs")
	if err != nil {
		t.Fatal(err)
	}
	defer subRoot.Close()

	data, err := subRoot.ReadFile("app.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{}" {
		t.Fatalf("unexpected content: %q", string(data))
	}
}

func TestChrootOpenRootDot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "test.txt"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := newChroot(t, root)

	// Test opening "." (same root)
	sameRoot, err := c.OpenRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	defer sameRoot.Close()

	data, err := sameRoot.ReadFile("test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "test" {
		t.Fatalf("unexpected content: %q", string(data))
	}

	// Test that they're independent
	c.Close()

	// sameRoot should still work
	data, err = sameRoot.ReadFile("test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "test" {
		t.Fatalf("unexpected content after parent close: %q", string(data))
	}
}

func TestChrootOpenRootNonExistent(t *testing.T) {
	root := t.TempDir()
	c := newChroot(t, root)

	_, err := c.OpenRoot("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected not-exist error, got: %v", err)
	}
}

func TestChrootOpenRootOnFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := newChroot(t, root)

	_, err := c.OpenRoot("file.txt")
	if err == nil {
		t.Fatal("expected error opening file as root")
	}
}

func TestOpenRootHelperWithChroot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "data/file.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := newChroot(t, root)

	// Use helper function
	subRoot, err := OpenRoot(c, "data")
	if err != nil {
		t.Fatal(err)
	}
	defer subRoot.Close()

	data, err := subRoot.ReadFile("file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "data" {
		t.Fatalf("unexpected content: %q", string(data))
	}
}

func TestOpenRootHelperWithOsRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "subdir/test.txt"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	osRoot, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer osRoot.Close()

	// Use helper function with os.Root
	subRoot, err := OpenRoot(osRoot, "subdir")
	if err != nil {
		t.Fatal(err)
	}
	defer subRoot.Close()

	data, err := subRoot.ReadFile("test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "test" {
		t.Fatalf("unexpected content: %q", string(data))
	}

	// Verify it implements RootLike
	var _ RootLike = subRoot
}

func TestRootLikeWithOpenRootInterface(t *testing.T) {
	root := t.TempDir()
	c := newChroot(t, root)

	// Verify Chroot implements RootLikeWithOpenRoot
	var _ RootLikeWithOpenRoot = c

	// Verify we can use it through the interface
	var rootWithOpen RootLikeWithOpenRoot = c
	if err := os.Mkdir(filepath.Join(root, "test"), 0o755); err != nil {
		t.Fatal(err)
	}

	sub, err := rootWithOpen.OpenRoot("test")
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()

	// Verify returned value is also RootLike
	var _ RootLike = sub
}

func TestOpenRootNamePropagation(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a/b/c"), 0o755); err != nil {
		t.Fatal(err)
	}

	c := newChroot(t, root)
	if c.Name() != root {
		t.Fatalf("expected name %q, got %q", root, c.Name())
	}

	a, _ := c.OpenRoot("a")
	defer a.Close()
	expectedA := filepath.Join(root, "a")
	if a.Name() != expectedA {
		t.Fatalf("expected name %q, got %q", expectedA, a.Name())
	}

	b, _ := a.(RootLikeWithOpenRoot).OpenRoot("b")
	defer b.Close()
	expectedB := filepath.Join(root, "a/b")
	if b.Name() != expectedB {
		t.Fatalf("expected name %q, got %q", expectedB, b.Name())
	}

	c2, _ := b.(RootLikeWithOpenRoot).OpenRoot("c")
	defer c2.Close()
	expectedC := filepath.Join(root, "a/b/c")
	if c2.Name() != expectedC {
		t.Fatalf("expected name %q, got %q", expectedC, c2.Name())
	}
}
