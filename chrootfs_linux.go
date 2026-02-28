//go:build linux

package chrootfs

import (
	"io/fs"
	"os"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

// ChrootFS is an fs.FS backed by a fixed root directory.
//
// Paths are resolved via openat2(RESOLVE_IN_ROOT), so symlink traversals are
// constrained to the root directory passed to New.
type ChrootFS struct {
	mu   sync.RWMutex
	root *os.File
}

var _ fs.FS = (*ChrootFS)(nil)

// New initializes a ChrootFS rooted at dir.
func New(dir string) (*ChrootFS, error) {
	if dir == "" {
		return nil, &fs.PathError{Op: "open", Path: dir, Err: fs.ErrInvalid}
	}

	fd, err := unix.Open(dir, unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: dir, Err: err}
	}

	return &ChrootFS{root: os.NewFile(uintptr(fd), dir)}, nil
}

// Close releases the root directory handle.
func (c *ChrootFS) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c == nil || c.root == nil {
		return nil
	}

	err := c.root.Close()
	c.root = nil
	return err
}

// Open opens name within the configured root using RESOLVE_IN_ROOT.
func (c *ChrootFS) Open(name string) (fs.File, error) {
	cleaned, err := cleanOpenName(name)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c == nil || c.root == nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: os.ErrClosed}
	}

	how := &unix.OpenHow{
		Flags:   uint64(unix.O_RDONLY | unix.O_CLOEXEC),
		Resolve: uint64(unix.RESOLVE_IN_ROOT | unix.RESOLVE_NO_MAGICLINKS),
	}

	fd, err := unix.Openat2(int(c.root.Fd()), cleaned, how)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}

	return os.NewFile(uintptr(fd), cleaned), nil
}

func cleanOpenName(name string) (string, error) {
	if strings.ContainsRune(name, 0) {
		return "", fs.ErrInvalid
	}
	if !fs.ValidPath(name) {
		return "", fs.ErrInvalid
	}

	return name, nil
}
