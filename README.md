# go-chrootfs

`go-chrootfs` provides a chroot-like filesystem implementation backed by Linux `openat2` with `RESOLVE_IN_ROOT`.

## Features

- **os.Root-compatible API**: Implements the same `RootLike` interface as Go 1.24+ `os.Root`
- **io/fs.FS view**: Access files through the standard `io/fs` interface
- **Path traversal prevention**: Uses `openat2` with `RESOLVE_IN_ROOT` to prevent escaping the root
- **Absolute symlink handling**: Unlike `os.Root` which rejects absolute symlinks, go-chrootfs resolves them relative to the root (e.g., a symlink to `/etc/passwd` points to `/sandbox/etc/passwd`)
- **Sub-root creation**: Create isolated subdirectory roots with `OpenRoot`

## Prerequisites

- OS: Linux
- Kernel: 5.6 or later (`openat2` + `RESOLVE_IN_ROOT` support)
- Go: 1.25+

## Install

```bash
go get github.com/tsuzu/go-chrootfs
```

## Usage

### Basic Usage with RootLike API

```go
package main

import (
    "fmt"
    "github.com/tsuzu/go-chrootfs"
)

func main() {
    // Create a chroot at /sandbox
    root, err := chrootfs.New("/sandbox")
    if err != nil {
        panic(err)
    }
    defer root.Close()

    // Use RootLike API (similar to os.Root)
    data, err := root.ReadFile("config.json")
    if err != nil {
        panic(err)
    }
    fmt.Println(string(data))

    // Write files
    err = root.WriteFile("output.txt", []byte("hello"), 0644)
    if err != nil {
        panic(err)
    }

    // Create directories
    err = root.MkdirAll("logs/app", 0755)
    if err != nil {
        panic(err)
    }

    // Open files for reading/writing
    f, err := root.OpenFile("data.txt", os.O_RDWR|os.O_CREATE, 0644)
    if err != nil {
        panic(err)
    }
    defer f.Close()
}
```

### Using io/fs.FS Interface

```go
root, err := chrootfs.New("/sandbox")
if err != nil {
    panic(err)
}
defer root.Close()

// Get io/fs.FS view
fsys := root.FS()

// Use with standard library functions
data, err := fs.ReadFile(fsys, "etc/hosts")
if err != nil {
    panic(err)
}

// Walk directory tree
fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
    if err != nil {
        return err
    }
    fmt.Println(path)
    return nil
})
```

### Creating Sub-roots

```go
root, err := chrootfs.New("/data")
if err != nil {
    panic(err)
}
defer root.Close()

// Create an isolated sub-root for configs
configRoot, err := root.OpenRoot("configs")
if err != nil {
    panic(err)
}
defer configRoot.Close()

// configRoot can only access /data/configs and below
data, err := configRoot.ReadFile("app.json")

// Works with both Chroot and os.Root
subRoot, err := chrootfs.OpenRoot(root, "subdir")
if err != nil {
    panic(err)
}
defer subRoot.Close()
```

## Comparison with os.Root

`go-chrootfs` provides similar functionality to Go 1.24+ `os.Root`, but with some differences:

| Feature | go-chrootfs | os.Root (Go 1.24+) |
|---------|-------------|-------------------|
| **Platform** | Linux only (openat2) | Cross-platform |
| **Kernel requirement** | Linux 5.6+ | N/A |
| **Path traversal prevention** | `openat2` with `RESOLVE_IN_ROOT` | Platform-specific mechanisms |
| **Absolute symlink handling** | Resolved relative to root | Rejected (error) |
| **API compatibility** | `RootLike` interface | `os.Root` type |
| **io/fs.FS view** | ✅ `FS()` method | ✅ `FS()` method |
| **Sub-root creation** | ✅ `OpenRoot()` method | ✅ `OpenRoot()` method |
| **Absolute path handling** | Converted to relative | Converted to relative |

### When to use go-chrootfs vs os.Root

**Use go-chrootfs when:**
- You need guaranteed `openat2` with `RESOLVE_IN_ROOT` semantics
- You're running on Linux 5.6+ and want the strongest path traversal prevention
- You want to ensure consistent behavior across your codebase on Linux
- You need to follow absolute symlinks that point outside the root (they will be resolved within the root)

**Use os.Root when:**
- You need cross-platform support (Windows, macOS, BSD, etc.)
- You're building an application that should work on any Go-supported platform
- You want to use the standard library implementation
- You want absolute symlinks to be rejected with an error (stricter symlink handling)

Both provide similar security guarantees and can be used interchangeably through the `RootLike` interface:

```go
// Function that works with both
func processFiles(root chrootfs.RootLike) error {
    data, err := root.ReadFile("config.json")
    // ...
    return nil
}

// Use with go-chrootfs
chroot, _ := chrootfs.New("/data")
defer chroot.Close()
processFiles(chroot)

// Use with os.Root
osRoot, _ := os.OpenRoot("/data")
defer osRoot.Close()
processFiles(osRoot)
```

## API Reference

See the [package documentation](https://pkg.go.dev/github.com/tsuzu/go-chrootfs) for detailed API reference.

## License

MIT License. See [LICENSE](LICENSE) file for details.
