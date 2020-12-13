---
title: Go API
---

The Ethereum blockchain along with its two extension protocols Whisper and Swarm was
originally conceptualized to become the supporting pillar of web3, providing the
consensus, messaging and storage backbone for a new generation of distributed (actually,
decentralized) applications called DApps.

The first incarnation towards this dream of web3 was a command line client providing an
RPC interface into the peer-to-peer protocols. The client was soon enough extended with a
web-browser-like graphical user interface, permitting developers to write DApps based on
the tried and proven HTML/CSS/JS technologies.

As many DApps have more complex requirements than what a browser environment can handle,
it became apparent that providing programmatic access to the web3 pillars would open the
door towards a new class of applications. As such, the second incarnation of the web3
dream is to open up all our technologies for other projects as reusable components.

Starting with the 1.5 release family of `go-ethereum`, we transitioned away from providing
only a full blown Ethereum client and started shipping official Go packages that could be
embedded into third party desktop and server applications.

*Note, this guide will assume you are familiar with Go development. It will make no
attempts to cover general topics about Go project layouts, import paths or any other
standard methodologies. If you are new to Go, consider reading its [getting started
guides](https://github.com/golang/go/wiki#getting-started-with-go) first.*

## Quick overview

Our reusable Go libraries focus on four main usage areas:

- Simplified client side account management
- Remote node interfacing via different transports
- Contract interactions through auto-generated bindings
- In-process Ethereum, Whisper and Swarm peer-to-peer node

You can watch a quick overview about these in Peter's (@karalabe) talk titled "Import
Geth: Ethereum from Go and beyond", presented at the Ethereum Devcon2 developer conference
in September, 2016 (Shanghai). Slides are [available
here](https://ethereum.karalabe.com/talks/2016-devcon.html).

[![Peter's Devcon2 talk](https://img.youtube.com/vi/R0Ia1U9Gxjg/0.jpg)](https://www.youtube.com/watch?v=R0Ia1U9Gxjg)

## Go packages

The `go-ethereum` library is distributed as a collection of standard Go packages straight
from our GitHub repository. The packages can be used directly via the official Go toolkit,
without needing any third party tools. External dependencies are vendored locally into
`vendor`, ensuring both self-containment as well as code stability. If you reuse
`go-ethereum` in your own project, please follow these best practices and vendor it
yourself too to avoid any accidental API breakages!

The canonical import path for `go-ethereum` is `github.com/ethereum/go-ethereum`, with all
packages residing underneath. Although there are [quite a
number](https://godoc.org/github.com/ethereum/go-ethereum#pkg-subdirectories) of them,
you'll only need to care about a limited subset, each of which will be properly introduced
in their relevant section.

You can download all our packages via:

```
$ go get -d github.com/ethereum/go-ethereum/...
```

You may also need Go's original context package. Although this was moved into the official
Go SDK in Go 1.7, `go-ethereum` will depend on the original `golang.org/x/net/context`
package until we officially drop support for Go 1.5 and Go 1.6.

```
$ go get -u golang.org/x/net/context
```
