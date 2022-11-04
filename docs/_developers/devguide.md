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

We assume that you have Go installed. Please use Go version 1.13 or later. We use the go
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

A stack trace provides a very detailed look into the current state of the geth node. 
It helps us to debug issues easier as it contains information about what is currently 
done by the node. Stack traces can be created by running `debug.stacks()` in the Geth
console. If the node was started without the console command or with a script in the 
background, the following command can be used to dump the stack trace into a file.

```
geth attach <path-to-geth.ipc> --exec "debug.stacks()" > stacktrace.txt
```
Geth logs the location of the IPC endpoint on startup. It is typically under 
`/home/user/.ethereum/geth.ipc` or `/tmp/geth.ipc`.

`debug.stacks()` also takes an optional `filter` argument. Passing a package name or
filepath to `filter` restricts the output to stack traces involcing only that package/file.
For example:

```sh
debug.stacks("enode")
```

returns data that looks like:

```terminal
INFO [11-04|16:15:54.486] Expanded filter expression               filter=enode   expanded="`enode` in Value"
goroutine 121 [chan receive, 3 minutes]:
github.com/ethereum/go-ethereum/p2p/enode.(*FairMix).nextFromAny(...)
	github.com/ethereum/go-ethereum/p2p/enode/iter.go:241
github.com/ethereum/go-ethereum/p2p/enode.(*FairMix).Next(0xc0008c6060)
	github.com/ethereum/go-ethereum/p2p/enode/iter.go:215 +0x2c5
github.com/ethereum/go-ethereum/p2p.(*dialScheduler).readNodes(0xc00021c2c0, {0x18149b0, 0xc0008c6060})
	github.com/ethereum/go-ethereum/p2p/dial.go:321 +0x9f
created by github.com/ethereum/go-ethereum/p2p.newDialScheduler
	github.com/ethereum/go-ethereum/p2p/dial.go:179 +0x425
```

and 
```sh
debug.stacks("consolecmd.go")
```

returns data that looks like:

```terminal
INFO [11-04|16:16:47.141] Expanded filter expression               filter=consolecmd.go expanded="`consolecmd.go` in Value"
goroutine 1 [chan receive]:
github.com/ethereum/go-ethereum/internal/jsre.(*JSRE).Do(0xc0004223c0, 0xc0003c00f0)
	github.com/ethereum/go-ethereum/internal/jsre/jsre.go:230 +0xf4
github.com/ethereum/go-ethereum/internal/jsre.(*JSRE).Evaluate(0xc00033eb60?, {0xc0013c00a0, 0x1e}, {0x180d720?, 0xc000010018})
	github.com/ethereum/go-ethereum/internal/jsre/jsre.go:289 +0xb3
github.com/ethereum/go-ethereum/console.(*Console).Evaluate(0xc0005366e0, {0xc0013c00a0?, 0x0?})
	github.com/ethereum/go-ethereum/console/console.go:353 +0x6d
github.com/ethereum/go-ethereum/console.(*Console).Interactive(0xc0005366e0)
	github.com/ethereum/go-ethereum/console/console.go:481 +0x691
main.localConsole(0xc00026d580?)
	github.com/ethereum/go-ethereum/cmd/geth/consolecmd.go:109 +0x348
github.com/ethereum/go-ethereum/internal/flags.MigrateGlobalFlags.func2.1(0x20b52c0?)
	github.com/ethereum/go-ethereum/internal/flags/helpers.go:91 +0x36
github.com/urfave/cli/v2.(*Command).Run(0x20b52c0, 0xc000313540)
	github.com/urfave/cli/v2@v2.17.2-0.20221006022127-8f469abc00aa/command.go:177 +0x719
github.com/urfave/cli/v2.(*App).RunContext(0xc0005501c0, {0x1816128?, 0xc000040110}, {0xc00003c180, 0x3, 0x3})
	github.com/urfave/cli/v2@v2.17.2-0.20221006022127-8f469abc00aa/app.go:387 +0x1035
github.com/urfave/cli/v2.(*App).Run(...)
	github.com/urfave/cli/v2@v2.17.2-0.20221006022127-8f469abc00aa/app.go:252
main.main()
	github.com/ethereum/go-ethereum/cmd/geth/main.go:266 +0x47

goroutine 159 [chan receive, 4 minutes]:
github.com/ethereum/go-ethereum/node.(*Node).Wait(...)
	github.com/ethereum/go-ethereum/node/node.go:529
main.localConsole.func1()
	github.com/ethereum/go-ethereum/cmd/geth/consolecmd.go:103 +0x2d
created by main.localConsole
	github.com/ethereum/go-ethereum/cmd/geth/consolecmd.go:102 +0x32e
```

If `geth` is started with the `--pprof` option, a debugging HTTP server is made available
on port 6060. You can bring up <http://localhost:6060/debug/pprof> to see the heap,
running routines etc. By clicking "full goroutine stack dump" you can generate a trace
that is useful for debugging.

Note that if you run multiple instances of `geth`, this port will only work for the first
instance that was launched. If you want to generate stacktraces for these other instances,
you need to start them up choosing an alternative pprof port. Make sure you are
redirecting stderr to a logfile.

```
geth -port=30300 -verbosity 5 --pprof --pprof.port 6060 2>> /tmp/00.glog
geth -port=30301 -verbosity 5 --pprof --pprof.port 6061 2>> /tmp/01.glog
geth -port=30302 -verbosity 5 --pprof --pprof.port 6062 2>> /tmp/02.glog
```

Alternatively if you want to kill the clients (in case they hang or stalled syncing, etc)
and have the stacktrace too, you can use the `-QUIT` signal with `kill`:

```
killall -QUIT geth
```

This will dump stack traces for each instance to their respective log file. Please do not
dump the stack trace into a GH issue as it is very hard for reviewers to read and intepret. 
It is much better to upload the trace to a Github Gist or Pastebin and put the link in the
issue.

[install-guide]: ../install-and-build/installing-geth
[code-review]: ../developers/code-review-guidelines
[cross-compile]: ../install-and-build/cross-compile
[go-modules]: https://github.com/golang/go/wiki/Modules
[discord]: https://discord.gg/invite/nthXNEv
[go-install]: https://golang.org/doc/install
[testflag]: https://golang.org/cmd/go/#hdr-Testing_flags
