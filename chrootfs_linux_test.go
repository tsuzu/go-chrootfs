//go:build linux

package chrootfs

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRelative(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfs, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cfs.Close() })

	f, err := cfs.Open("hello.txt")
	if err != nil {
		t.Fatalf("Open(%q): %v", "hello.txt", err)
	}

	b, err := io.ReadAll(f)
	_ = f.Close()
	if err != nil {
		t.Fatalf("ReadAll(%q): %v", "hello.txt", err)
	}
	if string(b) != "hello" {
		t.Fatalf("unexpected content for %q: %q", "hello.txt", string(b))
	}
}

func TestOpenRejectsInvalidFSPath(t *testing.T) {
	root := t.TempDir()
	cfs, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cfs.Close() })

	for _, name := range []string{"", "/hello.txt", "../hello.txt", "a/../b", "./hello.txt"} {
		_, err := cfs.Open(name)
		if err == nil {
			t.Fatalf("Open(%q): expected error", name)
		}
		if !errors.Is(err, fs.ErrInvalid) {
			t.Fatalf("Open(%q): expected fs.ErrInvalid, got: %v", name, err)
		}
	}
}

func TestOpenRejectsDotDotEscapePath(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}

	outside := filepath.Join(base, "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfs, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cfs.Close() })

	for _, name := range []string{"../../outside.txt", "dir/../../outside.txt"} {
		_, err := cfs.Open(name)
		if err == nil {
			t.Fatalf("Open(%q): expected error", name)
		}
		if !errors.Is(err, fs.ErrInvalid) {
			t.Fatalf("Open(%q): expected fs.ErrInvalid, got: %v", name, err)
		}
	}
}

func TestAbsoluteSymlinkIsResolvedInRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "target.txt"), []byte("in-root"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/target.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}

	cfs, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cfs.Close() })

	f, err := cfs.Open("link.txt")
	if err != nil {
		t.Fatal(err)
	}

	b, err := io.ReadAll(f)
	_ = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "in-root" {
		t.Fatalf("unexpected content: %q", string(b))
	}
}

func TestCannotEscapeRootViaAbsoluteSymlink(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}

	outside := filepath.Join(base, "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(outside, filepath.Join(root, "escape.txt")); err != nil {
		t.Fatal(err)
	}

	cfs, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cfs.Close() })

	_, err = cfs.Open("escape.txt")
	if err == nil {
		t.Fatal("expected escape to fail")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected not-exist style error, got: %v", err)
	}
}

func TestOpenDotReturnsRootDir(t *testing.T) {
	root := t.TempDir()

	cfs, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cfs.Close() })

	f, err := cfs.Open(".")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if !st.IsDir() {
		t.Fatal("expected root to be a directory")
	}
}

func TestReadFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfs, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cfs.Close() })

	data, err := cfs.ReadFile("hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected content: %q", string(data))
	}
}

func TestReadDirSorted(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"b.txt", "a.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cfs, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cfs.Close() })

	entries, err := cfs.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("unexpected entry count: %d", len(entries))
	}
	if entries[0].Name() != "a.txt" || entries[1].Name() != "b.txt" || entries[2].Name() != "c.txt" {
		t.Fatalf("unexpected order: %q, %q, %q", entries[0].Name(), entries[1].Name(), entries[2].Name())
	}
}

func TestStatLstatReadLink(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "target.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}

	cfs, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cfs.Close() })

	st, err := cfs.Stat("link.txt")
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode()&os.ModeSymlink != 0 {
		t.Fatal("Stat should follow symlink")
	}

	lst, err := cfs.Lstat("link.txt")
	if err != nil {
		t.Fatal(err)
	}
	if lst.Mode()&os.ModeSymlink == 0 {
		t.Fatal("Lstat should not follow final symlink")
	}

	target, err := cfs.ReadLink("link.txt")
	if err != nil {
		t.Fatal(err)
	}
	if target != "target.txt" {
		t.Fatalf("unexpected link target: %q", target)
	}
}

func TestReadLinkCannotEscapeViaSymlinkedParent(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	outside := filepath.Join(base, "outside")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("secret.txt", filepath.Join(outside, "link.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}

	cfs, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cfs.Close() })

	_, err = cfs.ReadLink("escape/link.txt")
	if err == nil {
		t.Fatal("expected readlink escape to fail")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected not-exist style error, got: %v", err)
	}
}

func TestGlobAndSub(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "b.log"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfs, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cfs.Close() })

	matches, err := cfs.Glob("sub/*.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0] != "sub/a.txt" {
		t.Fatalf("unexpected glob result: %#v", matches)
	}

	subFS, err := cfs.Sub("sub")
	if err != nil {
		t.Fatal(err)
	}

	rfs, ok := subFS.(fs.ReadFileFS)
	if !ok {
		t.Fatal("sub fs should support ReadFileFS")
	}
	data, err := rfs.ReadFile("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "a" {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}
