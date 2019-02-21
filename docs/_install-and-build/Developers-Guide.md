---
title: Developers' guide
---
**NOTE: These instructions are for people who want to contribute Go source code changes.
If you just want to run ethereum, use the normal [Installation Instructions](Building-Ethereum)**

This document is the entry point for developers of the Go implementation of Ethereum. Developers here refer to the hands-on: who are interested in build, develop, debug, submit a bug report or pull request or contribute code to go-ethereum.

## Building and Testing

### Go Environment

We assume that you have [`go` v1.8 installed](../doc/Installing-Go), and `GOPATH` is set.

**Note**:You must have your working copy under `$GOPATH/src/github.com/ethereum/go-ethereum`.

Since `go` does not use relative path for import, working in any other directory will have no effect, since the import paths will be appended to `$GOPATH/src`, and if the lib does not exist, the version at master HEAD will be downloaded.

Most likely you will be working from your fork of `go-ethereum`, let's say from `github.com/nirname/go-ethereum`. Clone or move your fork into the right place:

```
git clone git@github.com:nirname/go-ethereum.git $GOPATH/src/github.com/ethereum/go-ethereum
```

### Managing Vendored Dependencies

All other dependencies are tracked in the vendor/ directory. We use [govendor](https://github.com/kardianos/govendor) to manage them.

If you want to add a new dependency, run `govendor fetch <import-path>`, then commit the result.

If you want to update all dependencies to their latest upstream version, run `govendor fetch +v`.

You can also use govendor to run certain commands on all go-ethereum packages, excluding vendored
code. Example: to recreate all generated code, run `govendor generate +l`. 

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

Read about cross compilation of go-ethereum [here](../doc/Cross-compiling-Ethereum).

### Git flow

To make life easier try [git flow](http://nvie.com/posts/a-successful-git-branching-model/) it sets this all up and streamlines your work flow.

### Testing

Testing one library:

```
go test -v -cpu 4 ./eth  
```

Using options `-cpu` (number of cores allowed) and `-v` (logging even if no error) is recommended.

Testing only some methods:

```
go test -v -cpu 4 ./eth -run TestMethod
```

**Note**: here all tests with prefix _TestMethod_ will be run, so if you got TestMethod, TestMethod1, then both!

Running benchmarks, eg.:

```
go test -v -cpu 4 -bench . -run BenchmarkJoin
```

for more see [go test flags](http://golang.org/cmd/go/#hdr-Description_of_testing_flags)

### Metrics and monitoring

`geth` can do node behaviour monitoring, aggregation and show performance metric charts. 
Read about [metrics and monitoring](../doc/Metrics-and-Monitoring)

### Getting Stack Traces

If `geth` is started with the `--pprof` option, a debugging HTTP server is made available on port 6060. You can bring up http://localhost:6060/debug/pprof to see the heap, running routines etc. By clicking full goroutine stack dump (clicking http://localhost:6060/debug/pprof/goroutine?debug=2) you can generate trace that is useful for debugging.

Note that if you run multiple instances of `geth`, this port will only work for the first instance that was launched. If you want to generate stacktraces for these other instances, you need to start them up choosing an alternative pprof port. Make sure you are redirecting stderr to a logfile. 

```
geth -port=30300 -verbosity 5 --pprof --pprofport 6060 2>> /tmp/00.glog
geth -port=30301 -verbosity 5 --pprof --pprofport 6061 2>> /tmp/01.glog
geth -port=30302 -verbosity 5 --pprof --pprofport 6062 2>> /tmp/02.glog
```

Alternatively if you want to kill the clients (in case they hang or stalled syncing, etc) but have the stacktrace too, you can use the `-QUIT` signal with `kill`:

```
killall -QUIT geth 
```

This will dump stack traces for each instance to their respective log file.

## Contributing

Thank you for considering to help out with the source code! We welcome contributions from
anyone on the internet, and are grateful for even the smallest of fixes!

GitHub is used to track issues and contribute code, suggestions, feature requests or
documentation.

If you'd like to contribute to go-ethereum, please fork, fix, commit and send a pull
request (PR) for the maintainers to review and merge into the main code base. If you wish
to submit more complex changes though, please check up with the core devs first on [our
gitter channel](https://gitter.im/ethereum/go-ethereum) to ensure those changes are in
line with the general philosophy of the project and/or get some early feedback which can
make both your efforts much lighter as well as our review and merge procedures quick and
simple.

PRs need to be based on and opened against the `master` branch (unless by explicit
agreement, you contribute to a complex feature branch).

Your PR will be reviewed according to the [Code Review
Guidelines](../doc/Code-Review-Guidelines).

We encourage a PR early approach, meaning you create the PR the earliest even without the
fix/feature. This will let core devs and other volunteers know you picked up an issue.
These early PRs should indicate 'in progress' status.

## Dev Tutorials (mostly outdated)

* [Private networks, local clusters and monitoring](../doc/Setting-up-private-network-or-local-cluster)

* [P2P 101](../doc/Peer-to-Peer): a tutorial about setting up and creating a p2p server and p2p sub protocol.

* [How to Whisper](../doc/How-to-Whisper): an introduction to whisper.
