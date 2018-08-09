## Swarm

[https://swarm.ethereum.org](https://swarm.ethereum.org)

Swarm is a distributed storage platform and content distribution service, a native base layer service of the ethereum web3 stack. The primary objective of Swarm is to provide a decentralized and redundant store for dapp code and data as well as block chain and state data. Swarm is also set out to provide various base layer services for web3, including node-to-node messaging, media streaming, decentralised database services and scalable state-channel infrastructure for decentralised service economies.

[![Travis](https://travis-ci.org/ethereum/go-ethereum.svg?branch=master)](https://travis-ci.org/ethereum/go-ethereum)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/ethersphere/orange-lounge?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

## Table of Contents

*  [Building the source](#building-the-source)
*  [Running Swarm](#running-swarm)
*  [Documentation](#documentation)
*  [Developers Guide](#developers-guide)
   *  [Go Environment](#development-environment)
   *  [Vendored Dependencies](#vendored-dependencies)
   *  [Testing](#testing)
   *  [Profiling Swarm](#profiling-swarm)
   *  [Metrics and Instrumentation in Swarm](#metrics-and-instrumentation-in-swarm)
*  [Public Gateways](#public-gateways)
*  [Swarm Dapps](#swarm-dapps)
*  [Contributing](#contributing)
*  [License](#license)

## Building the source

Building Swarm requires Go (version 1.10 or later).

    go get -d github.com/ethereum/go-ethereum

    go install github.com/ethereum/go-ethereum/cmd/swarm

## Running Swarm

Going through all the possible command line flags is out of scope here, but we've enumerated a few common parameter combos to get you up to speed quickly on how you can run your own Swarm node.

To run Swarm you need an Ethereum account. You can create a new account by running the following command:

    geth account new

You will be prompted for a password:

    Your new account is locked with a password. Please give a password. Do not forget this password.
    Passphrase:
    Repeat passphrase:

Once you have specified the password, the output will be the Ethereum address representing that account. For example:

    Address: {2f1cd699b0bf461dcfbf0098ad8f5587b038f0f1}

Using this account, connect to Swarm with

    swarm --bzzaccount <your-account-here>

    # in our example

    swarm --bzzaccount 2f1cd699b0bf461dcfbf0098ad8f5587b038f0f1


### Verifying that your local Swarm node is running

When running, Swarm is accessible through an HTTP API on port 8500.

Confirm that it is up and running by pointing your browser to http://localhost:8500

### Ethereum Name Service resolution

The Ethereum Name Service is the Ethereum equivalent of DNS in the classic web. In order to use ENS to resolve names to Swarm content hashes (e.g. `bzz://theswarm.eth`), `swarm` has to connect to a `geth` instance, which is synced with the Ethereum mainnet. This is done using the `--ens-api` flag.

    swarm --bzzaccount <your-account-here> \
          --ens-api '$HOME/.ethereum/geth.ipc'

    # in our example

    swarm --bzzaccount 2f1cd699b0bf461dcfbf0098ad8f5587b038f0f1 \
          --ens-api '$HOME/.ethereum/geth.ipc'

For more information on usage, features or command line flags, please consult the Documentation.


## Documentation

Swarm documentation can be found at [https://swarm-guide.readthedocs.io](https://swarm-guide.readthedocs.io).


## Developers Guide

### Go Environment

We assume that you have Go v1.10 installed, and `GOPATH` is set.

You must have your working copy under `$GOPATH/src/github.com/ethereum/go-ethereum`.

Most likely you will be working from your fork of `go-ethereum`, let's say from `github.com/nirname/go-ethereum`. Clone or move your fork into the right place:

```
git clone git@github.com:nirname/go-ethereum.git $GOPATH/src/github.com/ethereum/go-ethereum
```


### Vendored Dependencies

All dependencies are tracked in the `vendor` directory. We use `govendor` to manage them.

If you want to add a new dependency, run `govendor fetch <import-path>`, then commit the result.

If you want to update all dependencies to their latest upstream version, run `govendor fetch +v`.


### Testing

This section explains how to run unit, integration, and end-to-end tests in your development sandbox.

Testing one library:

```
go test -v -cpu 4 ./swarm/api
```

Note: Using options -cpu (number of cores allowed) and -v (logging even if no error) is recommended.

Testing only some methods:

```
go test -v -cpu 4 ./eth -run TestMethod
```

Note: here all tests with prefix TestMethod will be run, so if you got TestMethod, TestMethod1, then both!

Running benchmarks:

```
go test -v -cpu 4 -bench . -run BenchmarkJoin
```


### Profiling Swarm

This section explains how to add Go `pprof` profiler to Swarm

If `swarm` is started with the `--pprof` option, a debugging HTTP server is made available on port 6060.

You can bring up http://localhost:6060/debug/pprof to see the heap, running routines etc.

By clicking full goroutine stack dump (clicking http://localhost:6060/debug/pprof/goroutine?debug=2) you can generate trace that is useful for debugging.


### Metrics and Instrumentation in Swarm

This section explains how to visualize and use existing Swarm metrics and how to instrument Swarm with a new metric.

Swarm metrics system is based on the `go-metrics` library.

The most common types of measurements we use in Swarm are `counters` and `resetting timers`. Consult the `go-metrics` documentation for full reference of available types.

```
# incrementing a counter
metrics.GetOrRegisterCounter("network.stream.received_chunks", nil).Inc(1)

# measuring latency with a resetting timer
start := time.Now()
t := metrics.GetOrRegisterResettingTimer("http.request.GET.time"), nil)
...
t := UpdateSince(start)
```

#### Visualizing metrics

Swarm supports an InfluxDB exporter. Consult the help section to learn about the command line arguments used to configure it:

```
swarm --help | grep metrics
```

We use Grafana and InfluxDB to visualise metrics reported by Swarm. We keep our Grafana dashboards under version control at `./swarm/grafana_dashboards`. You could use them or design your own.

We have built a tool to help with automatic start of Grafana and InfluxDB and provisioning of dashboards at https://github.com/nonsense/stateth , which requires that you have Docker installed.

Once you have `stateth` installed, and you have Docker running locally, you have to:

1. Run `stateth` and keep it running in the background
```
stateth --rm --grafana-dashboards-folder $GOPATH/src/github.com/ethereum/go-ethereum/swarm/grafana_dashboards --influxdb-database metrics
```

2. Run `swarm` with at least the following params:
```
--metrics \
--metrics.influxdb.export \
--metrics.influxdb.endpoint "http://localhost:8086" \
--metrics.influxdb.username "admin" \
--metrics.influxdb.password "admin" \
--metrics.influxdb.database "metrics"
```

3. Open Grafana at http://localhost:3000 and view the dashboards to gain insight into Swarm.


## Public Gateways

Swarm offers a local HTTP proxy API that Dapps can use to interact with Swarm. The Ethereum Foundation is hosting a public gateway, which allows free access so that people can try Swarm without running their own node.

The Swarm public gateways are temporary and users should not rely on their existence for production services.

The Swarm public gateway can be found at https://swarm-gateways.net and is always running the latest `stable` Swarm release.

## Swarm Dapps

You can find a few reference Swarm decentralised applications at: https://swarm-gateways.net/bzz:/swarmapps.eth

Their source code can be found at: https://github.com/ethersphere/swarm-dapps

## Contributing

Thank you for considering to help out with the source code! We welcome contributions from
anyone on the internet, and are grateful for even the smallest of fixes!

If you'd like to contribute to Swarm, please fork, fix, commit and send a pull request
for the maintainers to review and merge into the main code base. If you wish to submit more
complex changes though, please check up with the core devs first on [our Swarm gitter channel](https://gitter.im/ethersphere/orange-lounge)
to ensure those changes are in line with the general philosophy of the project and/or get some
early feedback which can make both your efforts much lighter as well as our review and merge
procedures quick and simple.

Please make sure your contributions adhere to our coding guidelines:

 * Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
 * Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
 * Pull requests need to be based on and opened against the `master` branch.
 * [Code review guidelines](https://github.com/ethereum/go-ethereum/wiki/Code-Review-Guidelines).
 * Commit messages should be prefixed with the package(s) they modify.
   * E.g. "swarm/fuse: ignore default manifest entry"


## License

The go-ethereum library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html), also
included in our repository in the `COPYING.LESSER` file.

The go-ethereum binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also included
in our repository in the `COPYING` file.
