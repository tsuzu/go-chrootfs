//go:build !linux

package chrootfs

import (
	"errors"
	"io/fs"
	"os"
	"time"
)

var errUnsupported = errors.New("chrootfs requires Linux openat2 support")

// Chroot is unavailable on non-Linux platforms.
type Chroot struct {
	name string
}

var _ RootLike = (*Chroot)(nil)

func New(string) (*Chroot, error) {
	return nil, errUnsupported
}

func (c *Chroot) Name() string { return c.name }

func (c *Chroot) Close() error { return nil }

func (c *Chroot) Open(name string) (*os.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: errUnsupported}
}

func (c *Chroot) Create(name string) (*os.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: errUnsupported}
}

func (c *Chroot) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: errUnsupported}
}

func (c *Chroot) Chmod(name string, mode os.FileMode) error {
	return &fs.PathError{Op: "chmod", Path: name, Err: errUnsupported}
}

func (c *Chroot) Mkdir(name string, perm os.FileMode) error {
	return &fs.PathError{Op: "mkdir", Path: name, Err: errUnsupported}
}

func (c *Chroot) MkdirAll(name string, perm os.FileMode) error {
	return &fs.PathError{Op: "mkdir", Path: name, Err: errUnsupported}
}

func (c *Chroot) Chown(name string, uid, gid int) error {
	return &fs.PathError{Op: "chown", Path: name, Err: errUnsupported}
}

func (c *Chroot) Lchown(name string, uid, gid int) error {
	return &fs.PathError{Op: "lchown", Path: name, Err: errUnsupported}
}

func (c *Chroot) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return &fs.PathError{Op: "chtimes", Path: name, Err: errUnsupported}
}

func (c *Chroot) Remove(name string) error {
	return &fs.PathError{Op: "remove", Path: name, Err: errUnsupported}
}

func (c *Chroot) RemoveAll(name string) error {
	return &fs.PathError{Op: "removeall", Path: name, Err: errUnsupported}
}

func (c *Chroot) Stat(name string) (os.FileInfo, error) {
	return nil, &fs.PathError{Op: "stat", Path: name, Err: errUnsupported}
}

func (c *Chroot) Lstat(name string) (os.FileInfo, error) {
	return nil, &fs.PathError{Op: "lstat", Path: name, Err: errUnsupported}
}

func (c *Chroot) Readlink(name string) (string, error) {
	return "", &fs.PathError{Op: "readlink", Path: name, Err: errUnsupported}
}

func (c *Chroot) ReadLink(name string) (string, error) {
	return c.Readlink(name)
}

func (c *Chroot) Rename(oldname, newname string) error {
	return &os.LinkError{Op: "rename", Old: oldname, New: newname, Err: errUnsupported}
}

func (c *Chroot) Link(oldname, newname string) error {
	return &os.LinkError{Op: "link", Old: oldname, New: newname, Err: errUnsupported}
}

func (c *Chroot) Symlink(oldname, newname string) error {
	return &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: errUnsupported}
}

func (c *Chroot) ReadFile(name string) ([]byte, error) {
	return nil, &fs.PathError{Op: "read", Path: name, Err: errUnsupported}
}

func (c *Chroot) WriteFile(name string, data []byte, perm os.FileMode) error {
	return &fs.PathError{Op: "write", Path: name, Err: errUnsupported}
}

func (c *Chroot) ReadDir(name string) ([]fs.DirEntry, error) {
	return nil, &fs.PathError{Op: "readdir", Path: name, Err: errUnsupported}
}

func (c *Chroot) FS() fs.FS {
	return &chrootFS{}
}

type chrootFS struct{}

var _ fs.FS = (*chrootFS)(nil)
var _ fs.ReadFileFS = (*chrootFS)(nil)
var _ fs.ReadDirFS = (*chrootFS)(nil)
var _ fs.StatFS = (*chrootFS)(nil)
var _ fs.ReadLinkFS = (*chrootFS)(nil)
var _ fs.GlobFS = (*chrootFS)(nil)
var _ fs.SubFS = (*chrootFS)(nil)

func (c *chrootFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: errUnsupported}
}

func (c *chrootFS) ReadFile(name string) ([]byte, error) {
	return nil, &fs.PathError{Op: "read", Path: name, Err: errUnsupported}
}

func (c *chrootFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return nil, &fs.PathError{Op: "readdir", Path: name, Err: errUnsupported}
}

func (c *chrootFS) Stat(name string) (fs.FileInfo, error) {
	return nil, &fs.PathError{Op: "stat", Path: name, Err: errUnsupported}
}

func (c *chrootFS) Lstat(name string) (fs.FileInfo, error) {
	return nil, &fs.PathError{Op: "lstat", Path: name, Err: errUnsupported}
}

func (c *chrootFS) ReadLink(name string) (string, error) {
	return "", &fs.PathError{Op: "readlink", Path: name, Err: errUnsupported}
}

func (c *chrootFS) Glob(pattern string) ([]string, error) {
	return nil, &fs.PathError{Op: "glob", Path: pattern, Err: errUnsupported}
}

func (c *chrootFS) Sub(dir string) (fs.FS, error) {
	return nil, &fs.PathError{Op: "sub", Path: dir, Err: errUnsupported}
}
