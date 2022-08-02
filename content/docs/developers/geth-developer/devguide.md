---
title: Getting Started
sort_key: A
---

This document is the entry point for developers who wish to work on Geth.
Developers are people who are interested to build, develop, debug, submit
a bug report or pull request or otherwise contribute to the Geth source code.

Please see [Contributing](/content/docs/developers/contributing.md) for the
Geth contribution guidelines.

## Building and Testing

Developers should use a recent version of Go for building and testing. We use the go
toolchain for development, which you can get from the [Go downloads page][go-install].

Geth is a Go module, and uses the [Go modules system][go-modules] to manage
dependencies. Using `GOPATH` is not required to build go-ethereum.

### Building Executables

Switch to the go-ethereum repository root directory.
All code can be built using the go tool, placing the resulting binary in `$GOPATH/bin`.

```text
go install -v ./...
```

go-ethereum exectuables can be built individually. To build just geth, use:

```text
go install -v ./cmd/geth
```

Cross compilation is not recommended, please build Geth for the host architecture.

### Testing

Testing a package:

```
go test -v ./eth
```

Running an individual test:

```
go test -v ./eth -run TestMethod
```

**Note**: here all tests with prefix _TestMethod_ will be run, so if TestMethod and
TestMethod1 both exist then both tests will run.

Running benchmarks, eg.:

```
go test -v -bench . -run BenchmarkJoin
```

For more information, see the [go test flags][testflag] documentation.

### Stack Traces

If Geth is started with the `--pprof` option, a debugging HTTP server is made available
on port 6060. Navigating to <http://localhost:6060/debug/pprof> displays the heap,
running routines etc. By clicking "full goroutine stack dump" a trace can be generated
that is useful for debugging.

Note that if multiple instances of Geth exist, port `6060` will only work for the first
instance that was launched. To generate stacktraces for other instances,
they should be started up with alternative pprof ports. Ensure `stderr` is being
redirected to a logfile.

```
geth -port=30300 -verbosity 5 --pprof --pprof.port 6060 2>> /tmp/00.glog
geth -port=30301 -verbosity 5 --pprof --pprof.port 6061 2>> /tmp/01.glog
geth -port=30302 -verbosity 5 --pprof --pprof.port 6062 2>> /tmp/02.glog
```

Alternatively to kill the clients (in case they hang or stalled syncing, etc)
and have the stacktrace too, use the `-QUIT` signal with `kill`:

```
killall -QUIT geth
```

This will dump stack traces for each instance to their respective log file.

[install-guide]: ../install-and-build/installing-geth
[code-review]: ../developers/code-review-guidelines
[cross-compile]: ../install-and-build/cross-compile
[go-modules]: https://github.com/golang/go/wiki/Modules
[discord]: https://discord.gg/invite/nthXNEv
[go-install]: https://golang.org/doc/install
[testflag]: https://golang.org/cmd/go/#hdr-Testing_flags
