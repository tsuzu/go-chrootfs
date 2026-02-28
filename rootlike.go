package chrootfs

import (
	"io/fs"
	"os"
	"time"
)

// RootLike is a Root-style filesystem API.
//
// It is intentionally similar to os.Root, but does not include OpenRoot.
type RootLike interface {
	Name() string
	Close() error

	Open(name string) (*os.File, error)
	Create(name string) (*os.File, error)
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)

	Chmod(name string, mode os.FileMode) error
	Mkdir(name string, perm os.FileMode) error
	MkdirAll(name string, perm os.FileMode) error
	Chown(name string, uid, gid int) error
	Lchown(name string, uid, gid int) error
	Chtimes(name string, atime time.Time, mtime time.Time) error

	Remove(name string) error
	RemoveAll(name string) error

	Stat(name string) (os.FileInfo, error)
	Lstat(name string) (os.FileInfo, error)
	Readlink(name string) (string, error)

	Rename(oldname, newname string) error
	Link(oldname, newname string) error
	Symlink(oldname, newname string) error

	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error

	FS() fs.FS
}
