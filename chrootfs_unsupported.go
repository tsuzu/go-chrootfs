//go:build !linux

package chrootfs

import (
	"errors"
	"io/fs"
)

var errUnsupported = errors.New("chrootfs requires Linux openat2 support")

// ChrootFS is unavailable on non-Linux platforms.
type ChrootFS struct{}

var _ fs.FS = (*ChrootFS)(nil)
var _ fs.ReadFileFS = (*ChrootFS)(nil)
var _ fs.ReadDirFS = (*ChrootFS)(nil)
var _ fs.StatFS = (*ChrootFS)(nil)
var _ fs.ReadLinkFS = (*ChrootFS)(nil)
var _ fs.GlobFS = (*ChrootFS)(nil)
var _ fs.SubFS = (*ChrootFS)(nil)

func New(string) (*ChrootFS, error) {
	return nil, errUnsupported
}

func (c *ChrootFS) Close() error {
	return nil
}

func (c *ChrootFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: errUnsupported}
}

func (c *ChrootFS) ReadFile(name string) ([]byte, error) {
	return nil, &fs.PathError{Op: "read", Path: name, Err: errUnsupported}
}

func (c *ChrootFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return nil, &fs.PathError{Op: "readdir", Path: name, Err: errUnsupported}
}

func (c *ChrootFS) Stat(name string) (fs.FileInfo, error) {
	return nil, &fs.PathError{Op: "stat", Path: name, Err: errUnsupported}
}

func (c *ChrootFS) ReadLink(name string) (string, error) {
	return "", &fs.PathError{Op: "readlink", Path: name, Err: errUnsupported}
}

func (c *ChrootFS) Lstat(name string) (fs.FileInfo, error) {
	return nil, &fs.PathError{Op: "lstat", Path: name, Err: errUnsupported}
}

func (c *ChrootFS) Glob(pattern string) ([]string, error) {
	return nil, &fs.PathError{Op: "glob", Path: pattern, Err: errUnsupported}
}

func (c *ChrootFS) Sub(dir string) (fs.FS, error) {
	return nil, &fs.PathError{Op: "sub", Path: dir, Err: errUnsupported}
}
