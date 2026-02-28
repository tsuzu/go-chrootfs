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

func TestOpenRelativeAndAbsolute(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfs, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cfs.Close() })

	for _, name := range []string{"hello.txt", "/hello.txt"} {
		f, err := cfs.Open(name)
		if err != nil {
			t.Fatalf("Open(%q): %v", name, err)
		}

		b, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			t.Fatalf("ReadAll(%q): %v", name, err)
		}
		if string(b) != "hello" {
			t.Fatalf("unexpected content for %q: %q", name, string(b))
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
