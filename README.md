# go-chrootfs

`go-chrootfs` provides a chroot-like `io/fs` implementation backed by Linux `openat2` with `RESOLVE_IN_ROOT`.

## Prerequisites

- OS: Linux
- Kernel: 5.6 or later (`openat2` + `RESOLVE_IN_ROOT` support)
- Go: 1.25+

## Install

```bash
go get github.com/tsuzu/go-chrootfs
```

## Usage

```go
cfs, err := chrootfs.New("/sandbox")
if err != nil {
    // handle error
}
defer cfs.Close()

b, err := fs.ReadFile(cfs, "etc/hosts")
```
