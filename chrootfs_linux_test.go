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

func newChroot(t *testing.T, root string) *Chroot {
	t.Helper()
	c, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestFSOpenRelative(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	fsys := newChroot(t, root).FS()
	f, err := fsys.Open("hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hello" {
		t.Fatalf("unexpected content: %q", string(b))
	}
}

func TestFSRejectsInvalidPath(t *testing.T) {
	fsys := newChroot(t, t.TempDir()).FS()

	for _, name := range []string{"", "/hello.txt", "../hello.txt", "a/../b", "./hello.txt"} {
		_, err := fsys.Open(name)
		if err == nil {
			t.Fatalf("Open(%q): expected error", name)
		}
		if !errors.Is(err, fs.ErrInvalid) {
			t.Fatalf("Open(%q): expected fs.ErrInvalid, got: %v", name, err)
		}
	}
}

func TestFSAbsoluteSymlinkIsResolvedInRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "target.txt"), []byte("in-root"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/target.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}

	fsys := newChroot(t, root).FS()
	b, err := fs.ReadFile(fsys, "link.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "in-root" {
		t.Fatalf("unexpected content: %q", string(b))
	}
}

func TestFSCannotEscapeRootViaAbsoluteSymlink(t *testing.T) {
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

	fsys := newChroot(t, root).FS()
	_, err := fsys.Open("escape.txt")
	if err == nil {
		t.Fatal("expected escape to fail")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected not-exist style error, got: %v", err)
	}
}

func TestFSReadDirStatLstatReadLink(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"b.txt", "a.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "target.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}

	fsys := newChroot(t, root).FS()

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 3 || entries[0].Name() != "a.txt" || entries[1].Name() != "b.txt" || entries[2].Name() != "c.txt" {
		t.Fatalf("unexpected order: %v", entries)
	}

	st, err := fs.Stat(fsys, "link.txt")
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode()&os.ModeSymlink != 0 {
		t.Fatal("Stat should follow symlink")
	}

	lst, err := fs.Lstat(fsys, "link.txt")
	if err != nil {
		t.Fatal(err)
	}
	if lst.Mode()&os.ModeSymlink == 0 {
		t.Fatal("Lstat should not follow final symlink")
	}

	target, err := fs.ReadLink(fsys, "link.txt")
	if err != nil {
		t.Fatal(err)
	}
	if target != "target.txt" {
		t.Fatalf("unexpected link target: %q", target)
	}
}

func TestFSGlobAndSub(t *testing.T) {
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

	fsys := newChroot(t, root).FS()
	matches, err := fs.Glob(fsys, "sub/*.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0] != "sub/a.txt" {
		t.Fatalf("unexpected glob result: %#v", matches)
	}

	sub, err := fs.Sub(fsys, "sub")
	if err != nil {
		t.Fatal(err)
	}
	b, err := fs.ReadFile(sub, "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "a" {
		t.Fatalf("unexpected content: %q", string(b))
	}
}

func TestRootLikeOpenSupportsAbsolutePath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := newChroot(t, root)
	f, err := c.Open("/hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hello" {
		t.Fatalf("unexpected content: %q", string(b))
	}
}

func TestRootLikeAbsoluteSymlinkIsResolvedInRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "target.txt"), []byte("in-root"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/target.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}

	c := newChroot(t, root)
	f, err := c.Open("/link.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "in-root" {
		t.Fatalf("unexpected content: %q", string(b))
	}
}

func TestRootLikeAbsoluteSymlinkCannotEscape(t *testing.T) {
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

	c := newChroot(t, root)
	_, err := c.Open("/escape.txt")
	if err == nil {
		t.Fatal("expected escape to fail")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected not-exist style error, got: %v", err)
	}
}

func TestRootLikeWriteMkdirRenameLinkSymlinkRemove(t *testing.T) {
	root := t.TempDir()
	c := newChroot(t, root)

	if err := c.MkdirAll("dir/sub", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := c.WriteFile("dir/sub/a.txt", []byte("A"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := c.Rename("dir/sub/a.txt", "dir/sub/b.txt"); err != nil {
		t.Fatal(err)
	}
	if err := c.Link("dir/sub/b.txt", "dir/sub/c.txt"); err != nil {
		t.Fatal(err)
	}
	if err := c.Symlink("b.txt", "dir/sub/link.txt"); err != nil {
		t.Fatal(err)
	}

	target, err := c.Readlink("dir/sub/link.txt")
	if err != nil {
		t.Fatal(err)
	}
	if target != "b.txt" {
		t.Fatalf("unexpected symlink target: %q", target)
	}

	if err := c.Chmod("dir/sub/b.txt", 0o600); err != nil {
		t.Fatal(err)
	}

	if err := c.Remove("dir/sub/link.txt"); err != nil {
		t.Fatal(err)
	}
	if err := c.RemoveAll("dir"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Stat("dir"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected removed dir to be not-exist, got: %v", err)
	}
}

func TestRootLikeStatDoesNotRequireReadPermission(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "x")
	if err := os.WriteFile(p, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o000); err != nil {
		t.Fatal(err)
	}

	c := newChroot(t, root)
	st, err := c.Stat("x")
	if err != nil {
		t.Fatal(err)
	}
	if st.Name() != "x" {
		t.Fatalf("unexpected name: %q", st.Name())
	}
}
