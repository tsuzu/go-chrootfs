package chrootfs

import (
	"io/fs"
	"os"
	"time"
)

// OpenRoot opens a subdirectory within the given root and returns it as a new root.
//
// This function provides a uniform interface for creating sub-roots that works with
// both os.Root (Go 1.24+) and Chroot implementations.
//
// For os.Root, it calls the OpenRoot method and wraps the result.
// For Chroot or other RootLike implementations that support OpenRoot, it calls that method.
// For other RootLike implementations, it returns an error.
//
// Example with os.Root:
//
//	osRoot, _ := os.OpenRoot("/data")
//	defer osRoot.Close()
//
//	subRoot, _ := chrootfs.OpenRoot(osRoot, "configs")
//	defer subRoot.(io.Closer).Close()
//
// Example with Chroot:
//
//	chroot, _ := chrootfs.New("/data")
//	defer chroot.Close()
//
//	subRoot, _ := chrootfs.OpenRoot(chroot, "configs")
//	defer subRoot.Close()
//
// The returned RootLike is independent of the parent. Closing the parent does not
// affect the child.
func OpenRoot(root RootLike, name string) (RootLike, error) {
	// Try RootLikeWithOpenRoot first (Chroot)
	if rootWithOpen, ok := root.(RootLikeWithOpenRoot); ok {
		return rootWithOpen.OpenRoot(name)
	}

	// Check if it's os.Root (has OpenRoot method that returns *os.Root)
	// We use type assertion with a local interface to avoid direct dependency
	type osRootLike interface {
		OpenRoot(string) (*os.Root, error)
	}

	if osRoot, ok := root.(osRootLike); ok {
		osSubRoot, err := osRoot.OpenRoot(name)
		if err != nil {
			return nil, err
		}
		return &osRootWrapper{root: osSubRoot}, nil
	}

	// Unsupported root type
	return nil, &fs.PathError{
		Op:   "openroot",
		Path: name,
		Err:  fs.ErrInvalid,
	}
}

// osRootWrapper wraps os.Root to implement RootLike interface.
//
// This allows os.Root to be used through the RootLike interface,
// providing a uniform API for both os.Root and Chroot.
type osRootWrapper struct {
	root *os.Root
}

var _ RootLike = (*osRootWrapper)(nil)

func (w *osRootWrapper) Name() string {
	// os.Root doesn't have a Name() method, return empty string
	return ""
}

func (w *osRootWrapper) Close() error {
	return w.root.Close()
}

func (w *osRootWrapper) Open(name string) (*os.File, error) {
	return w.root.Open(name)
}

func (w *osRootWrapper) Create(name string) (*os.File, error) {
	return w.root.Create(name)
}

func (w *osRootWrapper) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return w.root.OpenFile(name, flag, perm)
}

func (w *osRootWrapper) Chmod(name string, mode os.FileMode) error {
	return w.root.Chmod(name, mode)
}

func (w *osRootWrapper) Mkdir(name string, perm os.FileMode) error {
	return w.root.Mkdir(name, perm)
}

func (w *osRootWrapper) MkdirAll(name string, perm os.FileMode) error {
	return w.root.MkdirAll(name, perm)
}

func (w *osRootWrapper) Chown(name string, uid, gid int) error {
	return w.root.Chown(name, uid, gid)
}

func (w *osRootWrapper) Lchown(name string, uid, gid int) error {
	return w.root.Lchown(name, uid, gid)
}

func (w *osRootWrapper) Chtimes(name string, atime, mtime time.Time) error {
	return w.root.Chtimes(name, atime, mtime)
}

func (w *osRootWrapper) Remove(name string) error {
	return w.root.Remove(name)
}

func (w *osRootWrapper) RemoveAll(name string) error {
	return w.root.RemoveAll(name)
}

func (w *osRootWrapper) Stat(name string) (os.FileInfo, error) {
	return w.root.Stat(name)
}

func (w *osRootWrapper) Lstat(name string) (os.FileInfo, error) {
	return w.root.Lstat(name)
}

func (w *osRootWrapper) Readlink(name string) (string, error) {
	return w.root.Readlink(name)
}

func (w *osRootWrapper) Rename(oldname, newname string) error {
	return w.root.Rename(oldname, newname)
}

func (w *osRootWrapper) Link(oldname, newname string) error {
	return w.root.Link(oldname, newname)
}

func (w *osRootWrapper) Symlink(oldname, newname string) error {
	return w.root.Symlink(oldname, newname)
}

func (w *osRootWrapper) ReadFile(name string) ([]byte, error) {
	return w.root.ReadFile(name)
}

func (w *osRootWrapper) WriteFile(name string, data []byte, perm os.FileMode) error {
	return w.root.WriteFile(name, data, perm)
}

func (w *osRootWrapper) FS() fs.FS {
	// os.Root itself implements fs.FS (as of Go 1.24)
	// But since we don't know the exact version, we return nil
	// Users should use the root directly if they need fs.FS
	return nil
}
