# go-multistream

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-multiformats-blue.svg?style=flat-square)](https://github.com/multiformats/multiformats)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](https://webchat.freenode.net/?channels=%23ipfs)
[![](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)
[![GoDoc](https://godoc.org/github.com/multiformats/go-multistream?status.svg)](https://godoc.org/github.com/multiformats/go-multistream)
[![Travis CI](https://img.shields.io/travis/multiformats/go-multistream.svg?style=flat-square&branch=master)](https://travis-ci.org/multiformats/go-multistream)
[![codecov.io](https://img.shields.io/codecov/c/github/multiformats/go-multistream.svg?style=flat-square&branch=master)](https://codecov.io/github/multiformats/go-multistream?branch=master)

> an implementation of the multistream protocol in go

This package implements a simple stream router for the multistream-select protocol.
The protocol is defined [here](https://github.com/multiformats/multistream-select).

## Table of Contents


- [Install](#install)
- [Usage](#usage)
- [Maintainers](#maintainers)
- [Contribute](#contribute)
- [License](#license)

## Install

`go-multistream` is a standard Go module which can be installed with:

```sh
go get github.com/multiformats/go-multistream
```

Note that `go-multistream` is packaged with Gx, so it is recommended to use Gx to install and use it (see Usage section).


## Usage

### Using Gx and Gx-go

This module is packaged with [Gx](https://github.com/whyrusleeping/gx). In order to use it in your own project do:

```sh
go get -u github.com/whyrusleeping/gx
go get -u github.com/whyrusleeping/gx-go
cd <your-project-repository>
gx init
gx import github.com/multiformats/go-multistream
gx install --global
gx-go --rewrite
```

Please check [Gx](https://github.com/whyrusleeping/gx) and [Gx-go](https://github.com/whyrusleeping/gx-go) documentation for more information.

### Example


This example shows how to use a multistream muxer. A muxer uses user-added handlers to handle different "protocols". The first step when interacting with a connection handler by the muxer is to select the protocol (the example uses `SelectProtoOrFail`). This will then let the muxer use the right handler.


```go
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"

	ms "github.com/multiformats/go-multistream"
)

// This example creates a multistream muxer, adds handlers for the protocols
// "/cats" and "/docs" and exposes it on a localhost:8765. It then opens connections
// to that port, selects the protocols and tests that the handlers are working.
func main() {
	mux := ms.NewMultistreamMuxer()
	mux.AddHandler("/cats", func(proto string, rwc io.ReadWriteCloser) error {
		fmt.Fprintln(rwc, proto, ": HELLO I LIKE CATS")
		return rwc.Close()
	})
	mux.AddHandler("/dogs", func(proto string, rwc io.ReadWriteCloser) error {
		fmt.Fprintln(rwc, proto, ": HELLO I LIKE DOGS")
		return rwc.Close()
	})

	list, err := net.Listen("tcp", ":8765")
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			con, err := list.Accept()
			if err != nil {
				panic(err)
			}

			go mux.Handle(con)
		}
	}()

	// The Muxer is ready, let's test it
	conn, err := net.Dial("tcp", ":8765")
	if err != nil {
		panic(err)
	}

	// Create a new multistream to talk to the muxer
	// which will negotiate that we want to talk with /cats
	mstream := ms.NewMSSelect(conn, "/cats")
	cats, err := ioutil.ReadAll(mstream)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s", cats)
	mstream.Close()

	// A different way of talking to the muxer
	// is to manually selecting the protocol ourselves
	conn, err = net.Dial("tcp", ":8765")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	err = ms.SelectProtoOrFail("/dogs", conn)
	if err != nil {
		panic(err)
	}
	dogs, err := ioutil.ReadAll(conn)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s", dogs)
	conn.Close()
}
```


## Maintainers

Captain: [@whyrusleeping](https://github.com/whyrusleeping).

## Contribute

Contributions welcome. Please check out [the issues](https://github.com/multiformats/go-multistream/issues).

Check out our [contributing document](https://github.com/multiformats/multiformats/blob/master/contributing.md) for more information on how we work, and about contributing in general. Please be aware that all interactions related to multiformats are subject to the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md).

Small note: If editing the README, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

[MIT](LICENSE) © 2016 Jeromy Johnson
