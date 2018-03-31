# go-peerstream

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-libp2p-blue.svg?style=flat-square)](http://github.com/libp2p/libp2p)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)
[![standard-readme compliant](https://img.shields.io/badge/standard--readme-OK-green.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)
[![GoDoc](https://godoc.org/github.com/libp2p/go-peerstream?status.svg)](https://godoc.org/github.com/libp2p/go-peerstream)
[![Build Status](https://travis-ci.org/libp2p/go-peerstream.svg?branch=master)](https://travis-ci.org/libp2p/go-peerstream)
[![Coverage Status](https://coveralls.io/repos/github/libp2p/go-peerstream/badge.svg?branch=master)](https://coveralls.io/github/libp2p/go-peerstream?branch=master)

> P2P stream multi-multiplexing in Go

Package peerstream is a peer-to-peer networking library that multiplexes connections to many hosts. It attempts to simplify the complexity of:

* accepting incoming connections over **multiple** listeners
* dialing outgoing connections over **multiple** transports
* multiplexing **multiple** connections per-peer
* multiplexing **multiple** different servers or protocols
* handling backpressure correctly
* handling stream multiplexing
* providing a **simple** interface to the user

## Table of Contents

- [Install](#install)
- [Usage](#usage)
- [Maintainers](#maintainers)
- [Contribute](#contribute)
- [License](#license)

## Install

`go-peerstream` is a standard Go module which can be installed with:

```sh
go get github.com/libp2p/go-peerstream
```

Note that `go-peerstream` is packaged with Gx, so it is recommended to use Gx to install and use it (see Usage section).


## Usage

### Using Gx and Gx-go

This module is packaged with [Gx](https://github.com/whyrusleeping/gx). In order to use it in your own project it is recommended that you:

```sh
go get -u github.com/whyrusleeping/gx
go get -u github.com/whyrusleeping/gx-go
cd <your-project-repository>
gx init
gx import github.com/libp2p/go-peerstream
gx install --global
gx-go --rewrite
```

Please check [Gx](https://github.com/whyrusleeping/gx) and [Gx-go](https://github.com/whyrusleeping/gx-go) documentation for more information.

### Example

See [example/example.go](example/example.go) and [example/blockhandler/blockhandler.go](example/blockhandler/blockhandler.go) for examples covering the functionality of `go-peerstream`.

To build the examples, please make sure to run `make` in the `examples/` folder.

## Maintainers

This project is maintained by **@hsanjuan**.

## Contribute

PRs accepted.

Small note: If editing the README, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

MIT © Protocol Labs, Inc
