# `ghafs`

[![CI Status](https://img.shields.io/github/workflow/status/guangie88/ghafs/ci/master?label=ci&logo=github&style=for-the-badge)](https://github.com/guangie88/ghafs/actions)

Experimental GitHub Release Assets as FUSE filesystem mount.

This set-up only works for Linux and MacOS, since Windows does not support the
concept of FUSE.

## How to Use

Assuming that `ghafs` executable has already been built, the run command looks
like this:

```bash
./ghafs <mountpoint> <repo_owner> <repo_name>
```

Note that you should not need `root` privileges for the above command as long as
the mountpoint is a user mountable directory.

A proper example looks something like this:

```bash
mkdir -p /tmp/tera-cli
./ghafs /tmp/tera-cli guangie88 tera-cli
```

You should then be able to traverse into `/tmp/tera-cli` to look at the various
release tags and assets with another terminal / file browser.

To unmount, currently one has to do the following:

1. Press `CTRL-C` on the running `ghafs` to terminate the application
2. Make sure no other terminal / application is running at the mountpoint
3. If you are on Linux, you can run this to umount:
   `fusermount -u /tmp/tera-cli`.
   For MacOS, run `umount /tmp/tera-cli` instead,
   which may or may not require `sudo` (not tested).

## How to Build

You will need `go` of version 1.12 or higher for Go module support.

Simply run

```bash
go build -v ./...
```

This will generate the `ghafs` executable.

If you prefer a fully statically linked executable that can be deployed
anywhere, run with `CGO_ENABLED=0` env var disabled instead, like the following:

```bash
CGO_ENABLED=0 go build -v ./...
```

To check that it is indeed fully statically linked, run

```bash
ldd ghafs
```

And make sure it reads along the line of "not a dynamic executable".
