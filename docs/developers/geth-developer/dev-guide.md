---
title: Getting Started
description: Entry point for developers working on Geth
---

This document is the entry point for developers who wish to work on Geth. Developers are people who are interested to build, develop, debug, submit
a bug report or pull request or otherwise contribute to the Geth source code.

Please see [Contributing](/docs/developers/geth-developer/contributing) for the Geth contribution guidelines.

## Building and Testing {#building-and-testing}

Developers should use a recent version of Go for building and testing. We use the go toolchain for development, which you can get from the [Go downloads page](https://golang.org/doc/install). Geth is a Go module, and uses the [Go modules system](https://github.com/golang/go/wiki/Modules) to manage dependencies. Using `GOPATH` is not required to build go-ethereum.

### Building Executables {#building-executables}

Switch to the go-ethereum repository root directory. All code can be built using the go tool, placing the resulting binary in `$GOPATH/bin`.

```sh
go install -v ./...
```

go-ethereum executables can be built individually. To build just geth, use:

```sh
go install -v ./cmd/geth
```

Cross compilation is not recommended, please build Geth for the host architecture.

### Testing {#testing}

Testing a package:

```sh
go test -v ./eth
```

Running an individual test:

```sh
go test -v ./eth -run TestMethod
```

**Note**: here all tests with prefix _TestMethod_ will be run, so if TestMethod and TestMethod1 both exist then both tests will run.

Running benchmarks, eg.:

```sh
go test -v -bench . -run BenchmarkJoin
```

For more information, see the [go test flags](https://golang.org/cmd/go/#hdr-Testing_flags) documentation.

### Getting Stack Traces {#getting-stack-traces}

A stack trace provides a very detailed look into the current state of the geth node. It helps us to debug issues easier as it contains information about what is currently done by the node. Stack traces can be created by running `debug.stacks()` in the Geth console. If the node was started without the console command or with a script in the background, the following command can be used to dump the stack trace into a file.

```sh
geth attach <path-to-geth.ipc> --exec "debug.stacks()" > stacktrace.txt
```

Geth logs the location of the IPC endpoint on startup. It is typically under `/home/user/.ethereum/geth.ipc` or `/tmp/geth.ipc`.

`debug.stacks()` also takes an optional `filter` argument. Passing a package name or filepath to `filter` restricts the output to stack traces involving only that package/file. For example:

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

If Geth is started with the `--pprof` option, a debugging HTTP server is made available on port 6060. Navigating to <http://localhost:6060/debug/pprof> displays the heap, running routines etc. By clicking "full goroutine stack dump" a trace can be generated that is useful for debugging.

Note that if multiple instances of Geth exist, port `6060` will only work for the first instance that was launched. To generate stacktraces for other instances, they should be started up with alternative pprof ports. Ensure `stderr` is being redirected to a logfile.

```sh
geth -port=30300 -verbosity 5 --pprof --pprof.port 6060 2>> /tmp/00.glog
geth -port=30301 -verbosity 5 --pprof --pprof.port 6061 2>> /tmp/01.glog
geth -port=30302 -verbosity 5 --pprof --pprof.port 6062 2>> /tmp/02.glog
```

Alternatively to kill the clients (in case they hang or stalled syncing, etc) and have the stacktrace too, use the `-QUIT` signal with `kill`:

```sh
killall -QUIT geth
```

This will dump stack traces for each instance to their respective log file.

## Where to go next {#where-next}

Read the remaining pages in the Geth developer section, and get building!
