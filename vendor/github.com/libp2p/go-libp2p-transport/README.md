# go-libp2p-transport

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](https://protocol.ai)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](https://webchat.freenode.net/?channels=%23ipfs)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](https://libp2p.io/)
[![standard-readme compliant](https://img.shields.io/badge/standard--readme-OK-green.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)
[![GoDoc](https://godoc.org/github.com/libp2p/go-libp2p-transport?status.svg)](https://godoc.org/github.com/libp2p/go-libp2p-transport)
[![Coverage Status](https://img.shields.io/codecov/c/github/libp2p/go-libp2p-transport.svg?style=flat-square&branch=master)](https://codecov.io/github/libp2p/go-libp2p-transport?branch=master)
[![Build Status](https://travis-ci.org/libp2p/go-libp2p-transport.svg?branch=master)](https://travis-ci.org/libp2p/go-libp2p-transport)

> libp2p transport code

A common interface for network transports.

This is the 'base' layer for any transport that wants to be used by libp2p and ipfs. If you want to make 'ipfs work over X', the first thing you'll want to do is to implement the `Transport` interface for 'X'.

## Install

```sh
> gx install --global
> gx-go rewrite
```

## Usage

```go
var t Transport

t = NewTCPTransport()

list, err := t.Listen(listener_maddr)
if err != nil {
	log.Fatal(err)
}

con, err := list.Accept()
if err != nil {
	log.Fatal(err)
}

fmt.Fprintln(con, "Hello World!")
```

## Contribute

Feel free to join in. All welcome. Open an [issue](https://github.com/libp2p/go-libp2p-transport/issues)!

This repository falls under the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md).

### Want to hack on IPFS?

[![](https://cdn.rawgit.com/jbenet/contribute-ipfs-gif/master/img/contribute.gif)](https://github.com/ipfs/community/blob/master/contributing.md)

## License

MIT
