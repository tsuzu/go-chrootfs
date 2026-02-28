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

func New(string) (*ChrootFS, error) {
	return nil, errUnsupported
}

func (c *ChrootFS) Close() error {
	return nil
}

func (c *ChrootFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: errUnsupported}
}
