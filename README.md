## Swarm  <!-- omit in toc -->

[https://swarm.ethereum.org](https://swarm.ethereum.org)

Swarm is a distributed storage platform and content distribution service, a native base layer service of the ethereum web3 stack. The primary objective of Swarm is to provide a decentralized and redundant store for dapp code and data as well as block chain and state data. Swarm is also set out to provide various base layer services for web3, including node-to-node messaging, media streaming, decentralised database services and scalable state-channel infrastructure for decentralised service economies.

[![Travis](https://travis-ci.org/ethersphere/swarm.svg?branch=master)](https://travis-ci.org/ethersphere/swarm)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/ethersphere/orange-lounge?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

## Table of Contents  <!-- omit in toc -->

- [Building the source](#Building-the-source)
- [Running Swarm](#Running-Swarm)
  - [Verifying that your local Swarm node is running](#Verifying-that-your-local-Swarm-node-is-running)
  - [Ethereum Name Service resolution](#Ethereum-Name-Service-resolution)
- [Documentation](#Documentation)
- [Docker](#Docker)
  - [Docker tags](#Docker-tags)
  - [Environment variables](#Environment-variables)
  - [Swarm command line arguments](#Swarm-command-line-arguments)
- [Developers Guide](#Developers-Guide)
  - [Go Environment](#Go-Environment)
  - [Vendored Dependencies](#Vendored-Dependencies)
  - [Testing](#Testing)
  - [Profiling Swarm](#Profiling-Swarm)
  - [Metrics and Instrumentation in Swarm](#Metrics-and-Instrumentation-in-Swarm)
    - [Visualizing metrics](#Visualizing-metrics)
- [Public Gateways](#Public-Gateways)
- [Swarm Dapps](#Swarm-Dapps)
- [Contributing](#Contributing)
- [License](#License)

## Building the source

Building Swarm requires Go (version 1.11 or later).

To simply compile the `swarm` binary without a `GOPATH`:

```bash
$ git clone https://github.com/ethersphere/swarm
$ cd swarm
$ make swarm
```

You will find the binary under `./build/bin/swarm`.

To build a vendored `swarm` using `go get` you must have `GOPATH` set. Then run:

```bash
$ go get -d github.com/ethersphere/swarm
$ go install github.com/ethersphere/swarm/cmd/swarm
```

## Running Swarm

```bash
$ swarm
```

If you don't have an account yet, then you will be prompted to create one and secure it with a password:

```
Your new account is locked with a password. Please give a password. Do not forget this password.
Passphrase:
Repeat passphrase:
```

If you have multiple accounts created, then you'll have to choose one of the accounts by using the `--bzzaccount` flag.

```bash
$ swarm --bzzaccount <your-account-here>

# example
$ swarm --bzzaccount 2f1cd699b0bf461dcfbf0098ad8f5587b038f0f1
```

### Verifying that your local Swarm node is running

When running, Swarm is accessible through an HTTP API on port 8500.

Confirm that it is up and running by pointing your browser to http://localhost:8500

### Ethereum Name Service resolution

The Ethereum Name Service is the Ethereum equivalent of DNS in the classic web. In order to use ENS to resolve names to Swarm content hashes (e.g. `bzz://theswarm.eth`), `swarm` has to connect to a `geth` instance, which is synced with the Ethereum mainnet. This is done using the `--ens-api` flag.

```bash
$ swarm --bzzaccount <your-account-here> \
        --ens-api '$HOME/.ethereum/geth.ipc'

# in our example
$ swarm --bzzaccount 2f1cd699b0bf461dcfbf0098ad8f5587b038f0f1 \
        --ens-api '$HOME/.ethereum/geth.ipc'
```

For more information on usage, features or command line flags, please consult the Documentation.

## Documentation

Swarm documentation can be found at [https://swarm-guide.readthedocs.io](https://swarm-guide.readthedocs.io).

## Docker

Swarm container images are available at Docker Hub: [ethersphere/swarm](https://hub.docker.com/r/ethersphere/swarm)

### Docker tags

* `latest` - latest stable release
* `edge` - latest build from `master`
* `v0.x.y` - specific stable release

### Environment variables

* `PASSWORD` - *required* - Used to setup a sample Ethereum account in the data directory. If a data directory is mounted with a volume, the first Ethereum account from it is loaded, and Swarm will try to decrypt it non-interactively with `PASSWORD`
* `DATADIR` - *optional* - Defaults to `/root/.ethereum`

### Swarm command line arguments

All Swarm command line arguments are supported and can be sent as part of the CMD field to the Docker container.

**Examples:**

Running a Swarm container from the command line

```bash
$ docker run -e PASSWORD=password123 -t ethersphere/swarm \
                            --debug \
                            --verbosity 4
```

Running a Swarm container with custom ENS endpoint

```bash
$ docker run -e PASSWORD=password123 -t ethersphere/swarm \
                            --ens-api http://1.2.3.4:8545 \
                            --debug \
                            --verbosity 4
```

Running a Swarm container with metrics enabled

```bash
$ docker run -e PASSWORD=password123 -t ethersphere/swarm \
                            --debug \
                            --metrics \
                            --metrics.influxdb.export \
                            --metrics.influxdb.endpoint "http://localhost:8086" \
                            --metrics.influxdb.username "user" \
                            --metrics.influxdb.password "pass" \
                            --metrics.influxdb.database "metrics" \
                            --metrics.influxdb.host.tag "localhost" \
                            --verbosity 4
```

Running a Swarm container with tracing and pprof server enabled

```bash
$ docker run -e PASSWORD=password123 -t ethersphere/swarm \
                            --debug \
                            --tracing \
                            --tracing.endpoint 127.0.0.1:6831 \
                            --tracing.svc myswarm \
                            --pprof \
                            --pprofaddr 0.0.0.0 \
                            --pprofport 6060
```

Running a Swarm container with custom data directory mounted from a volume

```bash
$ docker run -e DATADIR=/data -e PASSWORD=password123 -v /tmp/hostdata:/data -t ethersphere/swarm \
                            --debug \
                            --verbosity 4
```

## Developers Guide

### Go Environment

We assume that you have Go v1.11 installed, and `GOPATH` is set.

You must have your working copy under `$GOPATH/src/github.com/ethersphere/swarm`.

Most likely you will be working from your fork of `swarm`, let's say from `github.com/nirname/swarm`. Clone or move your fork into the right place:

```bash
$ git clone git@github.com:nirname/swarm.git $GOPATH/src/github.com/ethersphere/swarm
```


### Vendored Dependencies

All dependencies are tracked in the `vendor` directory. We use `govendor` to manage them.

If you want to add a new dependency, run `govendor fetch <import-path>`, then commit the result.

If you want to update all dependencies to their latest upstream version, run `govendor fetch +v`.


### Testing

This section explains how to run unit, integration, and end-to-end tests in your development sandbox.

Testing one library:

```bash
$ go test -v -cpu 4 ./api
```

Note: Using options -cpu (number of cores allowed) and -v (logging even if no error) is recommended.

Testing only some methods:

```bash
$ go test -v -cpu 4 ./api -run TestMethod
```

Note: here all tests with prefix TestMethod will be run, so if you got TestMethod, TestMethod1, then both!

Running benchmarks:

```bash
$ go test -v -cpu 4 -bench . -run BenchmarkJoin
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

```go
// incrementing a counter
metrics.GetOrRegisterCounter("network.stream.received_chunks", nil).Inc(1)

// measuring latency with a resetting timer
start := time.Now()
t := metrics.GetOrRegisterResettingTimer("http.request.GET.time"), nil)
...
t := UpdateSince(start)
```

#### Visualizing metrics

Swarm supports an InfluxDB exporter. Consult the help section to learn about the command line arguments used to configure it:

```bash
$ swarm --help | grep metrics
```

We use Grafana and InfluxDB to visualise metrics reported by Swarm. We keep our Grafana dashboards under version control at https://github.com/ethersphere/grafana-dashboards. You could use them or design your own.

We have built a tool to help with automatic start of Grafana and InfluxDB and provisioning of dashboards at https://github.com/nonsense/stateth, which requires that you have Docker installed.

Once you have `stateth` installed, and you have Docker running locally, you have to:

1. Run `stateth` and keep it running in the background

```bash
$ stateth --rm --grafana-dashboards-folder $GOPATH/src/github.com/ethersphere/grafana-dashboards --influxdb-database metrics
```

2. Run `swarm` with at least the following params:

```bash
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
 * [Code review guidelines](https://github.com/ethersphere/swarm/blob/master/docs/Code-Review-Guidelines.md).
 * Commit messages should be prefixed with the package(s) they modify.
   * E.g. "fuse: ignore default manifest entry"


## License

The swarm library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html), also
included in our repository in the `COPYING.LESSER` file.

The swarm binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also included
in our repository in the `COPYING` file.
