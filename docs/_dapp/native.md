---
title: Go API
sort_key: C
---

Ethereum was originally conceptualized to be the base layer for [Web3][web3-link], providing 
the backbone for a new generation of decentralized, permissionless and censorship resistant 
applications called [dapps][dapp-link]. The first step towards this vision was the development 
of clients providing an RPC interface into the peer-to-peer protocols. This allowed users to 
transact between accounts and interact with smart contracts using command line tools. 
Geth was one of the original clients to provide this type of gateway to the Ethereum network.

Before long, web-browser-like graphical interfaces (e.g. Mist) were created to extend clients.
However, as dapps became more complex their requirements exceeded what a simple browser environment
could support. To support the next generation of dapps, built using time-tested HTML/CSS/JS 
technologies, programmatic access to client functions was required

Programmatic access to client functions is best delivered in the form of an open API. 
This not only levels-up the complexity of dapps that can be built on top of 
Ethereum but also opens up client technologies as re-usable, composable units that can be 
applied in creative ways by a global community of developers.

To support this vision, Geth ships official Go packages that can be embedded into third party 
desktop and server applications. There is also a [mobile API](/docs/dapp/mobile) that can be 
used to embed Geth into mobile applications.

This page provides a high-level overview of the Go API.

*Note, this guide will assume some familiarity with Go development. It does not cover general topics 
about Go project layouts, import paths or any other standard methodologies. If you are new to Go, 
consider reading [Getting Started with Go][go-guide] first.*

## Overview

Geth's reusable Go libraries focus on three main usage areas:

- Simplified client side account management
- Remote node interfacing via different transports
- Contract interactions through auto-generated bindings

The libraries are updated synchronously with the Geth Github repository. 
The Go libraries can be viewed in full at [Go Packages][go-pkg-link]

The benefits of using the Go libraries include easy client-side [account management](/docs/dapp/native-accounts)
that allows users to keep their private keys safely encrypted locally and make their own
security decisions. The Go libraries allow dapp developers to spend more of their time
building their dapps rather than implementing complex protocols such as encoding and
decoding function calls to smart contracts - this can all be abstracted away using
Geth's Go libraries.

Péter Szilágyi (@karalabe) gave a high level overview of the Go libraries in 
a talk at DevCon2 in Shanghai in 2016. The slides are still a useful resource
([available here][peter-slides]) and the talk itself can be viewed by clicking
the image below (it is also archived on [IPFS][ipfs-link]).

[![Peter's Devcon2 talk](/static/images/devcon2_labelled.webp)](https://www.youtube.com/watch?v=R0Ia1U9Gxjg)

## Go packages

The Geth library is distributed as a collection of standard Go packages straight from Geth's GitHub
repository. The packages can be used directly via the official Go toolkit, without needing any 
third party tools. External dependencies are vendored locally into `vendor`, ensuring both 
self-containment and code stability. When Geth is used in downstream projects these best
practices (packing dependencies into a local `vendor`) should be followed there too to 
avoid any accidental API breakages.

The canonical import path for Geth is `github.com/ethereum/go-ethereum`, with all packages residing
underneath. Although there are [lots of them][go-ethereum-dir] most developers will only care about 
a limited subset.

All the Geth packages can be downloaded using:

```
$ go get -d github.com/ethereum/go-ethereum/...
```

More Go API support for dapp developers can be found on the [Go Contract Bindings](/docs/dapp/native-bindings)
and [Go Account Management](/docs/dapp/native-accounts) pages.

## Summary

There are a wide variety of Go APIs available for dapp developers that abstract away the complexity of interacting with Ethereum
using a set of composable, reusable functions provided by Geth.

[go-guide]: https://github.com/golang/go/wiki#getting-started-with-go
[peter-slides]: https://ethereum.karalabe.com/talks/2016-devcon.html
[go-ethereum-dir]: https://pkg.go.dev/github.com/ethereum/go-ethereum/#section-directories
[go-pkg-link]: https://pkg.go.dev/github.com/ethereum/go-ethereum#section-directories
[ipfs-link]: https://ipfs.io/ipfs/QmQRuKPKWWJAamrMqAp9rytX6Q4NvcXUKkhvu3kuREKqXR
[dapp-link]: https://ethereum.org/en/glossary/#dapp
[web3-link]: https://ethereum.org/en/web3/