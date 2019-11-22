---
title: Developer Guide
sort_key: A
---

**NOTE: These instructions are for people who want to contribute Go source code changes.
If you just want to run ethereum, use the regular [Installation Instructions][install-guide].**

This document is the entry point for developers of the Go implementation of Ethereum.
Developers here refer to the hands-on: who are interested in build, develop, debug, submit
a bug report or pull request or contribute code to go-ethereum.

## Contributing

Thank you for considering to help out with the source code! We welcome contributions from
anyone on the internet, and are grateful for even the smallest of fixes!

GitHub is used to track issues and contribute code, suggestions, feature requests or
documentation.

If you'd like to contribute to go-ethereum, please fork, fix, commit and send a pull
request (PR) for the maintainers to review and merge into the main code base. If you wish
to submit more complex changes though, please check up with the core devs in the
go-ethereum [Discord Server][discord]. to ensure those changes are in line with the
general philosophy of the project and/or get some early feedback. This can reduce your
effort as well as speeding up our review and merge procedures.

PRs need to be based on and opened against the `master` branch (unless by explicit
agreement, you contribute to a complex feature branch).

Your PR will be reviewed according to the [Code Review guidelines][code-review].

We encourage a PR early approach, meaning you create the PR the earliest even without the
fix/feature. This will let core devs and other volunteers know you picked up an issue.
These early PRs should indicate 'in progress' status.

## Building and Testing

We assume that you have Go installed. Please use Go version 1.13 or later. We use the gc
toolchain for development, which you can get from the [Go downloads page][go-install].

go-ethereum is a Go module, and uses the [Go modules system][go-modules] to manage
dependencies. Using `GOPATH` is not required to build go-ethereum.

### Building Executables

Switch to the go-ethereum repository root directory.

You can build all code using the go tool, placing the resulting binary in `$GOPATH/bin`.

```text
go install -v ./...
```

go-ethereum exectuables can be built individually. To build just geth, use:

```text
go install -v ./cmd/geth
```

If you want to compile geth for an architecture that differs from your host, please
consult our [cross compilation guide][cross-compile].

### Testing

Testing a package:

```
go test -v ./eth
```

Running an individual test:

```
go test -v ./eth -run TestMethod
```

**Note**: here all tests with prefix _TestMethod_ will be run, so if you got TestMethod,
TestMethod1, then both tests will run.

Running benchmarks, eg.:

```
go test -v -bench . -run BenchmarkJoin
```

For more information, see the [go test flags][testflag] documentation.

### Getting Stack Traces

If `geth` is started with the `--pprof` option, a debugging HTTP server is made available
on port 6060. You can bring up <http://localhost:6060/debug/pprof> to see the heap,
running routines etc. By clicking "full goroutine stack dump" you can generate a trace
that is useful for debugging.

Note that if you run multiple instances of `geth`, this port will only work for the first
instance that was launched. If you want to generate stacktraces for these other instances,
you need to start them up choosing an alternative pprof port. Make sure you are
redirecting stderr to a logfile.

```
geth -port=30300 -verbosity 5 --pprof --pprofport 6060 2>> /tmp/00.glog
geth -port=30301 -verbosity 5 --pprof --pprofport 6061 2>> /tmp/01.glog
geth -port=30302 -verbosity 5 --pprof --pprofport 6062 2>> /tmp/02.glog
```

Alternatively if you want to kill the clients (in case they hang or stalled syncing, etc)
and have the stacktrace too, you can use the `-QUIT` signal with `kill`:

```
killall -QUIT geth
```

This will dump stack traces for each instance to their respective log file.

[install-guide]: ../install-and-build/installing-geth
[code-review]: ../developers/code-review-guidelines
[cross-compile]: ../install-and-build/cross-compile
[go-modules]: https://github.com/golang/go/wiki/Modules
[discord]: https://discord.gg/nthXNEv
[go-install]: https://golang.org/doc/install
[testflag]: https://golang.org/cmd/go/#hdr-Description_of_testing_flags
