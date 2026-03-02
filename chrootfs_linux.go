//go:build linux

package chrootfs

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

type pathCleaner func(name string) (string, error)

// Chroot is a root-like filesystem backed by a fixed directory.
//
// It provides an os.Root-compatible API for secure file operations within
// a directory tree. Unlike os.Root, it supports absolute paths (e.g., "/etc/hosts")
// which are resolved relative to the root directory.
//
// Path resolution uses openat2(RESOLVE_IN_ROOT | RESOLVE_NO_MAGICLINKS),
// ensuring that symbolic links and ".." components cannot escape the root.
//
// Example:
//
//	root, err := chrootfs.New("/sandbox")
//	if err != nil {
//	    return err
//	}
//	defer root.Close()
//
//	// Absolute paths are allowed
//	f, err := root.Open("/etc/config.json")
//
//	// Get io/fs.FS view for standard library functions
//	fsys := root.FS()
//	data, err := fs.ReadFile(fsys, "etc/config.json")
type Chroot struct {
	mu   sync.RWMutex
	name string
	root *os.File
}

var _ RootLike = (*Chroot)(nil)

// New initializes a Chroot rooted at dir.
func New(dir string) (*Chroot, error) {
	if dir == "" {
		return nil, &fs.PathError{Op: "open", Path: dir, Err: fs.ErrInvalid}
	}

	fd, err := unix.Open(dir, unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: dir, Err: err}
	}

	return &Chroot{name: dir, root: os.NewFile(uintptr(fd), dir)}, nil
}

// Name returns the path that was used to open this Chroot.
//
// For Chroots created via FS().Sub(), this returns the joined path.
// Note: if the underlying directory is moved after opening, this name
// may no longer reflect the actual filesystem location.
func (c *Chroot) Name() string {
	if c == nil {
		return ""
	}
	return c.name
}

// Close releases the root directory handle.
func (c *Chroot) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c == nil || c.root == nil {
		return nil
	}

	err := c.root.Close()
	c.root = nil
	return err
}

// Open opens name within the configured root for reading.
func (c *Chroot) Open(name string) (*os.File, error) {
	return c.open(name, "open", uint64(unix.O_RDONLY), 0, cleanRootName)
}

// Create creates or truncates the named file in the root.
func (c *Chroot) Create(name string) (*os.File, error) {
	return c.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
}

// OpenFile opens the named file in the root.
func (c *Chroot) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	if err := validatePermMode("open", name, perm); err != nil {
		return nil, err
	}
	mode := uint64(0)
	if flag&os.O_CREATE != 0 {
		mode = uint64(perm)
	}
	return c.open(name, "open", uint64(flag), mode, cleanRootName)
}

// ReadFile reads the named file.
func (c *Chroot) ReadFile(name string) ([]byte, error) {
	return c.readFile(name, "read", cleanRootName)
}

// WriteFile writes data to the named file, creating it if needed.
func (c *Chroot) WriteFile(name string, data []byte, perm os.FileMode) error {
	f, err := c.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	_, werr := f.Write(data)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return &fs.PathError{Op: "write", Path: name, Err: werr}
	}
	return nil
}

// ReadDir reads and returns sorted directory entries.
//
// The entries are sorted by name in ascending order as required by io/fs.
// For large directories (10000+ entries), this may have performance implications.
func (c *Chroot) ReadDir(name string) ([]fs.DirEntry, error) {
	return c.readDir(name, "readdir", cleanRootName)
}

// Stat returns a FileInfo describing the named file.
func (c *Chroot) Stat(name string) (os.FileInfo, error) {
	return c.stat(name, "stat", uint64(unix.O_PATH), cleanRootName)
}

// Lstat returns a FileInfo describing the named file without following final symlink.
//
// Note: This method is not part of the standard io/fs interfaces, but is provided
// for compatibility with os package conventions and the RootLike interface.
func (c *Chroot) Lstat(name string) (os.FileInfo, error) {
	return c.stat(name, "lstat", uint64(unix.O_PATH|unix.O_NOFOLLOW), cleanRootName)
}

// Readlink returns the destination of the named symbolic link.
func (c *Chroot) Readlink(name string) (string, error) {
	return c.readlink(name, "readlink", cleanRootName)
}

// ReadLink is an alias of Readlink.
func (c *Chroot) ReadLink(name string) (string, error) {
	return c.Readlink(name)
}

// Chmod changes the mode of the named file.
func (c *Chroot) Chmod(name string, mode os.FileMode) error {
	parentFD, baseName, err := c.openParent(name, "chmod", cleanRootName)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(parentFD) }()

	if err := unix.Fchmodat(parentFD, baseName, uint32(mode), 0); err != nil {
		return &fs.PathError{Op: "chmod", Path: name, Err: err}
	}
	return nil
}

// Chown changes the uid and gid of the named file.
func (c *Chroot) Chown(name string, uid, gid int) error {
	if uid < -1 || gid < -1 {
		return &fs.PathError{Op: "chown", Path: name,
			Err: errors.New("invalid uid or gid")}
	}

	parentFD, baseName, err := c.openParent(name, "chown", cleanRootName)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(parentFD) }()

	if err := unix.Fchownat(parentFD, baseName, uid, gid, 0); err != nil {
		return &fs.PathError{Op: "chown", Path: name, Err: err}
	}
	return nil
}

// Lchown changes the uid and gid of the named file without following final symlink.
func (c *Chroot) Lchown(name string, uid, gid int) error {
	if uid < -1 || gid < -1 {
		return &fs.PathError{Op: "lchown", Path: name,
			Err: errors.New("invalid uid or gid")}
	}

	parentFD, baseName, err := c.openParent(name, "lchown", cleanRootName)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(parentFD) }()

	if err := unix.Fchownat(parentFD, baseName, uid, gid, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return &fs.PathError{Op: "lchown", Path: name, Err: err}
	}
	return nil
}

// Chtimes changes the access and modification times of the named file.
func (c *Chroot) Chtimes(name string, atime time.Time, mtime time.Time) error {
	parentFD, baseName, err := c.openParent(name, "chtimes", cleanRootName)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(parentFD) }()

	ts := []unix.Timespec{
		unix.NsecToTimespec(atime.UnixNano()),
		unix.NsecToTimespec(mtime.UnixNano()),
	}
	if err := unix.UtimesNanoAt(parentFD, baseName, ts, 0); err != nil {
		return &fs.PathError{Op: "chtimes", Path: name, Err: err}
	}
	return nil
}

// Mkdir creates a new directory in the root.
func (c *Chroot) Mkdir(name string, perm os.FileMode) error {
	if err := validatePermMode("mkdir", name, perm); err != nil {
		return err
	}

	parentFD, baseName, err := c.openParent(name, "mkdir", cleanRootName)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(parentFD) }()

	if err := unix.Mkdirat(parentFD, baseName, uint32(perm)); err != nil {
		return &fs.PathError{Op: "mkdir", Path: name, Err: err}
	}
	return nil
}

// MkdirAll creates a directory and all necessary parents in the root.
//
// In concurrent environments, there is a small window where a directory
// could be replaced with a file between existence checks.
func (c *Chroot) MkdirAll(name string, perm os.FileMode) error {
	if err := validatePermMode("mkdir", name, perm); err != nil {
		return err
	}
	cleaned, err := cleanRootName(name)
	if err != nil {
		return &fs.PathError{Op: "mkdir", Path: name, Err: err}
	}
	if cleaned == "." {
		return nil
	}

	cur := ""
	for _, part := range strings.Split(cleaned, "/") {
		if part == "" || part == "." {
			continue
		}
		if cur == "" {
			cur = part
		} else {
			cur = cur + "/" + part
		}

		err := c.Mkdir(cur, perm)
		if err == nil {
			continue
		}
		if os.IsExist(err) {
			st, statErr := c.Stat(cur)
			if statErr == nil && st.IsDir() {
				continue
			}
		}
		return err
	}
	return nil
}

// Remove removes the named file or (empty) directory in the root.
func (c *Chroot) Remove(name string) error {
	parentFD, baseName, err := c.openParent(name, "remove", cleanRootName)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(parentFD) }()

	err = unix.Unlinkat(parentFD, baseName, 0)
	if err == nil {
		return nil
	}
	err2 := unix.Unlinkat(parentFD, baseName, unix.AT_REMOVEDIR)
	if err2 == nil {
		return nil
	}
	if err2 != unix.ENOTDIR {
		return &fs.PathError{Op: "remove", Path: name, Err: err2}
	}
	return &fs.PathError{Op: "remove", Path: name, Err: err}
}

// RemoveAll removes the named path and any children it contains.
func (c *Chroot) RemoveAll(name string) error {
	cleaned, err := cleanRootName(name)
	if err != nil {
		return &fs.PathError{Op: "removeall", Path: name, Err: err}
	}
	if cleaned == "." {
		return &fs.PathError{Op: "removeall", Path: name, Err: unix.EINVAL}
	}

	if err := c.removeAllDepth(cleaned, 0); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

const maxRemoveDepth = 1000

func (c *Chroot) removeAllDepth(cleaned string, depth int) error {
	if depth > maxRemoveDepth {
		return &fs.PathError{Op: "removeall", Path: cleaned,
			Err: errors.New("directory tree too deep")}
	}

	st, err := c.Lstat(cleaned)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if st.Mode()&os.ModeSymlink != 0 || !st.IsDir() {
		return c.Remove(cleaned)
	}

	entries, err := c.ReadDir(cleaned)
	if err != nil {
		return err
	}
	for _, ent := range entries {
		child := ent.Name()
		if cleaned != "." {
			child = cleaned + "/" + child
		}
		if err := c.removeAllDepth(child, depth+1); err != nil {
			return err
		}
	}
	return c.Remove(cleaned)
}

// Rename renames oldname to newname.
func (c *Chroot) Rename(oldname, newname string) error {
	oldParentFD, oldBaseName, err := c.openParent(oldname, "rename", cleanRootName)
	if err != nil {
		return &os.LinkError{Op: "rename", Old: oldname, New: newname, Err: err}
	}
	defer func() { _ = unix.Close(oldParentFD) }()

	newParentFD, newBaseName, err := c.openParent(newname, "rename", cleanRootName)
	if err != nil {
		return &os.LinkError{Op: "rename", Old: oldname, New: newname, Err: err}
	}
	defer func() { _ = unix.Close(newParentFD) }()

	if err := unix.Renameat(oldParentFD, oldBaseName, newParentFD, newBaseName); err != nil {
		return &os.LinkError{Op: "rename", Old: oldname, New: newname, Err: err}
	}
	return nil
}

// Link creates newname as a hard link to oldname.
func (c *Chroot) Link(oldname, newname string) error {
	oldParentFD, oldBaseName, err := c.openParent(oldname, "link", cleanRootName)
	if err != nil {
		return &os.LinkError{Op: "link", Old: oldname, New: newname, Err: err}
	}
	defer func() { _ = unix.Close(oldParentFD) }()

	newParentFD, newBaseName, err := c.openParent(newname, "link", cleanRootName)
	if err != nil {
		return &os.LinkError{Op: "link", Old: oldname, New: newname, Err: err}
	}
	defer func() { _ = unix.Close(newParentFD) }()

	if err := unix.Linkat(oldParentFD, oldBaseName, newParentFD, newBaseName, 0); err != nil {
		return &os.LinkError{Op: "link", Old: oldname, New: newname, Err: err}
	}
	return nil
}

// Symlink creates newname as a symbolic link to oldname.
func (c *Chroot) Symlink(oldname, newname string) error {
	newParentFD, newBaseName, err := c.openParent(newname, "symlink", cleanRootName)
	if err != nil {
		return &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: err}
	}
	defer func() { _ = unix.Close(newParentFD) }()

	if err := unix.Symlinkat(oldname, newParentFD, newBaseName); err != nil {
		return &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: err}
	}
	return nil
}

// FS returns an io/fs.FS view of this chroot.
//
// The returned filesystem follows io/fs conventions:
//   - Absolute paths are rejected (use relative paths only)
//   - Open() returns read-only files
//   - Implements fs.ReadFileFS, fs.ReadDirFS, fs.StatFS, fs.GlobFS, fs.SubFS
//
// For write operations, use the Chroot methods directly (Create, WriteFile, etc).
func (c *Chroot) FS() fs.FS {
	return &chrootFS{root: c}
}

// OpenRoot opens a subdirectory and returns it as a new RootLike.
//
// This method is similar to os.Root.OpenRoot, creating a new root filesystem
// at the specified subdirectory within the current root.
//
// The returned RootLike is independent of the parent Chroot. Closing the parent
// does not affect the child, and vice versa.
//
// Example:
//
//	root, _ := chrootfs.New("/data")
//	defer root.Close()
//
//	subRoot, _ := root.OpenRoot("configs")
//	defer subRoot.Close()
//
//	// subRoot operates within /data/configs
//	data, _ := subRoot.ReadFile("app.json")
func (c *Chroot) OpenRoot(name string) (RootLike, error) {
	if !fs.ValidPath(name) && name != "" && name != "." {
		// Try cleaning as root name (allows absolute paths)
		cleaned, err := cleanRootName(name)
		if err != nil {
			return nil, &fs.PathError{Op: "openroot", Path: name, Err: err}
		}
		name = cleaned
	}

	if name == "" || name == "." {
		// Return a new handle to the same root
		c.mu.RLock()
		defer c.mu.RUnlock()

		if c == nil || c.root == nil {
			return nil, &fs.PathError{Op: "openroot", Path: name, Err: os.ErrClosed}
		}

		fd, err := unix.Dup(int(c.root.Fd()))
		if err != nil {
			return nil, &fs.PathError{Op: "openroot", Path: name, Err: err}
		}

		return &Chroot{
			name: c.name,
			root: os.NewFile(uintptr(fd), c.name),
		}, nil
	}

	// Open subdirectory as new root
	subRoot, err := c.open(name, "openroot", uint64(unix.O_PATH|unix.O_DIRECTORY), 0, cleanRootName)
	if err != nil {
		return nil, err
	}

	return &Chroot{
		name: path.Join(c.name, name),
		root: subRoot,
	}, nil
}

func (c *Chroot) open(name string, op string, flags uint64, mode uint64, cleaner pathCleaner) (*os.File, error) {
	cleaned, err := cleaner(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c == nil || c.root == nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: os.ErrClosed}
	}

	fd, err := openat2(int(c.root.Fd()), cleaned, flags, mode)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}
	return os.NewFile(uintptr(fd), cleaned), nil
}

func (c *Chroot) readFile(name string, op string, cleaner pathCleaner) ([]byte, error) {
	f, err := c.open(name, op, uint64(unix.O_RDONLY), 0, cleaner)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}
	return data, nil
}

func (c *Chroot) readDir(name string, op string, cleaner pathCleaner) ([]fs.DirEntry, error) {
	f, err := c.open(name, op, uint64(unix.O_RDONLY|unix.O_DIRECTORY), 0, cleaner)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	entries, err := f.ReadDir(-1)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})
	return entries, nil
}

func (c *Chroot) stat(name string, op string, flags uint64, cleaner pathCleaner) (os.FileInfo, error) {
	f, err := c.open(name, op, flags, 0, cleaner)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}
	return st, nil
}

func (c *Chroot) readlink(name string, op string, cleaner pathCleaner) (string, error) {
	parentFD, baseName, err := c.openParent(name, op, cleaner)
	if err != nil {
		return "", err
	}
	defer func() { _ = unix.Close(parentFD) }()

	target, err := readlinkat(parentFD, baseName)
	if err != nil {
		return "", &fs.PathError{Op: op, Path: name, Err: err}
	}
	return target, nil
}

func (c *Chroot) openParent(name string, op string, cleaner pathCleaner) (int, string, error) {
	cleaned, err := cleaner(name)
	if err != nil {
		return 0, "", &fs.PathError{Op: op, Path: name, Err: err}
	}

	dirName, baseName := path.Split(cleaned)
	if dirName == "" {
		dirName = "."
	} else {
		dirName = strings.TrimSuffix(dirName, "/")
	}
	if baseName == "" {
		baseName = "."
	}

	c.mu.RLock()
	if c == nil || c.root == nil {
		c.mu.RUnlock()
		return 0, "", &fs.PathError{Op: op, Path: name, Err: os.ErrClosed}
	}
	parentFD, err := openat2(int(c.root.Fd()), dirName, uint64(unix.O_PATH|unix.O_DIRECTORY), 0)
	c.mu.RUnlock()
	if err != nil {
		return 0, "", &fs.PathError{Op: op, Path: name, Err: err}
	}
	return parentFD, baseName, nil
}

func openat2(rootFD int, name string, flags uint64, mode uint64) (int, error) {
	how := &unix.OpenHow{
		Flags:   flags | uint64(unix.O_CLOEXEC),
		Mode:    mode,
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

func validatePermMode(op string, path string, perm os.FileMode) error {
	if perm&0o777 != perm {
		return &fs.PathError{Op: op, Path: path,
			Err: errors.New("permission mode must be 0-0777, special bits not allowed")}
	}
	return nil
}

type chrootFS struct {
	root *Chroot
}

var _ fs.FS = (*chrootFS)(nil)
var _ fs.ReadFileFS = (*chrootFS)(nil)
var _ fs.ReadDirFS = (*chrootFS)(nil)
var _ fs.StatFS = (*chrootFS)(nil)
var _ fs.ReadLinkFS = (*chrootFS)(nil)
var _ fs.GlobFS = (*chrootFS)(nil)
var _ fs.SubFS = (*chrootFS)(nil)

func (f *chrootFS) Open(name string) (fs.File, error) {
	return f.root.open(name, "open", uint64(unix.O_RDONLY), 0, cleanOpenName)
}

func (f *chrootFS) ReadFile(name string) ([]byte, error) {
	return f.root.readFile(name, "read", cleanOpenName)
}

func (f *chrootFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return f.root.readDir(name, "readdir", cleanOpenName)
}

func (f *chrootFS) Stat(name string) (fs.FileInfo, error) {
	return f.root.stat(name, "stat", uint64(unix.O_PATH), cleanOpenName)
}

func (f *chrootFS) Lstat(name string) (fs.FileInfo, error) {
	return f.root.stat(name, "lstat", uint64(unix.O_PATH|unix.O_NOFOLLOW), cleanOpenName)
}

func (f *chrootFS) ReadLink(name string) (string, error) {
	return f.root.readlink(name, "readlink", cleanOpenName)
}

func (f *chrootFS) Glob(pattern string) ([]string, error) {
	return fs.Glob(openOnlyFS{open: f.Open}, pattern)
}

func (f *chrootFS) Sub(dir string) (fs.FS, error) {
	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "sub", Path: dir, Err: fs.ErrInvalid}
	}
	if dir == "." {
		return f, nil
	}

	subRoot, err := f.root.open(dir, "sub", uint64(unix.O_PATH|unix.O_DIRECTORY), 0, cleanOpenName)
	if err != nil {
		return nil, err
	}
	sub := &Chroot{
		name: path.Join(f.root.Name(), dir),
		root: subRoot,
	}
	return &chrootFS{root: sub}, nil
}

// openOnlyFS is a minimal fs.FS implementation for fs.Glob.
//
// The fs.Glob function requires an fs.FS with only an Open method,
// so we wrap the Open function to satisfy the interface.
type openOnlyFS struct {
	open func(string) (fs.File, error)
}

func (o openOnlyFS) Open(name string) (fs.File, error) {
	return o.open(name)
}

func cleanRootName(name string) (string, error) {
	if strings.ContainsRune(name, 0) {
		return "", fs.ErrInvalid
	}
	if name == "" {
		return "", fs.ErrInvalid
	}

	for strings.HasPrefix(name, "/") {
		name = strings.TrimPrefix(name, "/")
	}
	if name == "" {
		return ".", nil
	}

	if !fs.ValidPath(name) {
		return "", fs.ErrInvalid
	}
	return name, nil
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
