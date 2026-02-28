//go:build linux

package chrootfs

import (
	"io"
	"io/fs"
	"os"
	"path"
	"slices"
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
var _ fs.ReadFileFS = (*ChrootFS)(nil)
var _ fs.ReadDirFS = (*ChrootFS)(nil)
var _ fs.StatFS = (*ChrootFS)(nil)
var _ fs.ReadLinkFS = (*ChrootFS)(nil)
var _ fs.GlobFS = (*ChrootFS)(nil)
var _ fs.SubFS = (*ChrootFS)(nil)

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
	file, err := c.open(name, "open", uint64(unix.O_RDONLY))
	if err != nil {
		return nil, err
	}
	return file, nil
}

// ReadFile reads the named file.
func (c *ChrootFS) ReadFile(name string) ([]byte, error) {
	file, err := c.open(name, "read", uint64(unix.O_RDONLY))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, &fs.PathError{Op: "read", Path: name, Err: err}
	}
	return data, nil
}

// ReadDir reads and returns sorted directory entries.
func (c *ChrootFS) ReadDir(name string) ([]fs.DirEntry, error) {
	file, err := c.open(name, "readdir", uint64(unix.O_RDONLY|unix.O_DIRECTORY))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	entries, err := file.ReadDir(-1)
	if err != nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: err}
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})
	return entries, nil
}

// Stat returns a FileInfo describing the named file.
func (c *ChrootFS) Stat(name string) (fs.FileInfo, error) {
	file, err := c.open(name, "stat", uint64(unix.O_PATH))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: err}
	}
	return info, nil
}

// Lstat returns a FileInfo describing the named file without following final symlink.
func (c *ChrootFS) Lstat(name string) (fs.FileInfo, error) {
	file, err := c.open(name, "lstat", uint64(unix.O_PATH|unix.O_NOFOLLOW))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, &fs.PathError{Op: "lstat", Path: name, Err: err}
	}
	return info, nil
}

// ReadLink returns the destination of the named symbolic link.
func (c *ChrootFS) ReadLink(name string) (string, error) {
	cleaned, err := cleanOpenName(name)
	if err != nil {
		return "", &fs.PathError{Op: "readlink", Path: name, Err: err}
	}

	dirName, baseName := path.Split(cleaned)
	if dirName == "" {
		dirName = "."
	} else {
		dirName = strings.TrimSuffix(dirName, "/")
	}

	var parentFD int

	c.mu.RLock()
	if c == nil || c.root == nil {
		c.mu.RUnlock()
		return "", &fs.PathError{Op: "readlink", Path: name, Err: os.ErrClosed}
	}
	parentFD, err = openat2(int(c.root.Fd()), dirName, uint64(unix.O_PATH|unix.O_DIRECTORY))
	c.mu.RUnlock()
	if err != nil {
		return "", &fs.PathError{Op: "readlink", Path: name, Err: err}
	}
	defer func() { _ = unix.Close(parentFD) }()

	target, err := readlinkat(parentFD, baseName)
	if err != nil {
		return "", &fs.PathError{Op: "readlink", Path: name, Err: err}
	}
	return target, nil
}

// Glob returns matching file names for pattern.
func (c *ChrootFS) Glob(pattern string) ([]string, error) {
	return fs.Glob(openOnlyFS{open: c.Open}, pattern)
}

// Sub returns a filesystem rooted at dir under this root.
func (c *ChrootFS) Sub(dir string) (fs.FS, error) {
	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "sub", Path: dir, Err: fs.ErrInvalid}
	}
	if dir == "." {
		return c, nil
	}

	subRoot, err := c.open(dir, "sub", uint64(unix.O_PATH|unix.O_DIRECTORY))
	if err != nil {
		return nil, err
	}
	return &ChrootFS{root: subRoot}, nil
}

func (c *ChrootFS) open(name string, op string, flags uint64) (*os.File, error) {
	cleaned, err := cleanOpenName(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c == nil || c.root == nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: os.ErrClosed}
	}

	fd, err := openat2(int(c.root.Fd()), cleaned, flags)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	return os.NewFile(uintptr(fd), cleaned), nil
}

func openat2(rootFD int, name string, flags uint64) (int, error) {
	how := &unix.OpenHow{
		Flags:   flags | uint64(unix.O_CLOEXEC),
		Resolve: uint64(unix.RESOLVE_IN_ROOT | unix.RESOLVE_NO_MAGICLINKS),
	}
	return unix.Openat2(rootFD, name, how)
}

func readlinkat(dirFD int, name string) (string, error) {
	size := 128
	for {
		buf := make([]byte, size)
		n, err := unix.Readlinkat(dirFD, name, buf)
		if err != nil {
			return "", err
		}
		if n < len(buf) {
			return string(buf[:n]), nil
		}
		size *= 2
		if size > 1024*1024 {
			return "", unix.ENAMETOOLONG
		}
	}
}

type openOnlyFS struct {
	open func(string) (fs.File, error)
}

func (o openOnlyFS) Open(name string) (fs.File, error) {
	return o.open(name)
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
