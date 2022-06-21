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

Before long, web-browser-like graphical interfaces (e.g. Mist) were created to extend clients, and
client functions were built into websites built using the time-tested HTML/CSS/JS stack. 
However, to support the most diverse, complex dapps, developers require programmatic access to client
functions through an API. This opens up client technologies as re-usable, composable units that 
can be applied in creative ways by a global community of developers.

To support this, Geth ships official Go packages that can be embedded into third party 
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

Péter Szilágyi (@karalabe) gave a high level overview of the Go libraries in 
a talk at DevCon2 in Shanghai in 2016. The slides are still a useful resource
([available here][peter-slides]) and the talk itself can be viewed by clicking
the image below (it is also archived on [IPFS][ipfs-link]).

[![Peter's Devcon2 talk](/static/images/devcon2_labelled.webp)](https://www.youtube.com/watch?v=R0Ia1U9Gxjg)

## Go packages

The `go-ethereum` library is distributed as a collection of standard Go packages straight from Geth's GitHub
repository. The packages can be used directly via the official Go toolkit, without needing any 
third party tools.

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
