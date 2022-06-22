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

The `go-ethereum` library is distributed as a collection of standard Go packages straight from Geth's 
GitHub repository. The packages can be used directly via the official Go toolkit, without needing any 
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


## Tutorial

This section includes some basic usage examples for the `ethclient` and `gethclient` packages available as 
part of the Go API. The `ethclient` package provides a client that implements the full Ethereum JSON-RPC API, 
whereas `gethclient` offers the Geth-specific API. 

### Instantiating a client

The client is an instance of the `Client` struct which has associated functions that wrap requests to the Ethereum 
or Geth RPC API endpoints.

A client is instantiated by passing a raw url or path to an ipc file to the client's `Dial` function. In the following 
code snippet the path to the ipc file for a local Geth node is provided to `ethclient.Dial()`.

```go
// create instance of ethclient and assign to cl
cl, err := ethclient.Dial("/tmp/geth.ipc")
if err != nil {
	panic(err)
}
_ = cl
```

### Interacting with the client

The client can now be used to handle requests to the Geth node using the full JSON-RPC API. For example, the function
`blockNumer()` wraps a call to the `eth_blockNumber` endpoint. The function `sendTransaction` wraps a call to 
`eth_sendTransaction`. The syntax for calling these functions in a Go application is as follows:

```go
cl.a_function(arg1, arg2...)
```

Frequently, the functions take an instance of the `Context` type as their leading argument. This defines some low-level
rules about requests sent from the application such as deadlines, cancellation signals etc. More information on this can
be found in the [Go documentation](https://pkg.go.dev/golang.org/x/net/context). An empty context instance can be 
created using `Context.Background()`. 

### Simple functions

Some of the client functions simpy request data from the client and do not require any arguments other than the
`context.Context` instance that defines the request, for example `chainID()` and `PeerCount()`. Others
take additional arguments such as an Ethereum address or block number, for example `BalanceAt()`.

To request the network ID, the `ChainID()` function can be used. It wraps a call to the `eth_chainID` endpoint. 
The `ChainID()` function takes an instance of `context.Context` as its only argument. The network ID can therefore
be obtained as a `uint64` and assigned to `networkid` as follows:

```go
networkid, err := cl.NetworkID(context.Background())
if err != nil {
    return err
}
```

The `PeerCount` function also takes an instance of `context.Context` as its only argument. The number of peers the node
is connected to is returned as a `uint64` and assigned to `peers` in the following example:

```go
peers, err := cl.PeerCount(context.Background())
if err != nil {
    return err
}
```

Obtaining an account balance at a specific block is achieved using `BalanceAt()`. This function takes an instance of
`context.Context` as its leading argument but also requires an Ethereum address provided with type `common.Address` and
a block number which can either be an integer or nil (in which case the latest block is used):

```go

addr = common.HexToAddress("0xb02A2EdA1b317FBd16760128836B0Ac59B560e9D")

balance, err := cl.BalanceAt(context.Background(), addr, 0)
if err != nil {
    return err
}
```

### Sending a transaction

Sending a transaction is achieved using the `SendTransaction()` function. `SendTransaction` takes an instance of
`context.Context` as its leading argument and a signed transaction as its second argument. The signed transaction
must be generated in advance of calling `sendTransaction()`. Building the signed transaction is a multi-stage 
process that requires first generating a key pair, retrieving some chain data and defining sender and recipient 
addresses. Then these data can be collected into a transaction object and signed. The resulting signed transaction
can then be passed to `SendTransaction`. 

The example below assumes the following key pair has already been generated:

```go
// SK and ADDR are the secret key and sender address
SK   = "0xaf5ead4413ff4b78bc94191a2926ae9ccbec86ce099d65aaf469e9eb1a0fa87f"
ADDR = "0x6177843db3138ae69679A54b95cf345ED759450d"
```

The secret key and address can be used to send a transaction. In the example below 1 ETH is sent from the
address `ADDR` to an arbitrary recipient.

```go
import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

// sendTransaction sends a transaction with 1 ETH to a specified address.
func sendTransaction(cl *ethclient.Client) error {
	var (
		sk       = crypto.ToECDSAUnsafe(common.FromHex(SK))
		to       = common.HexToAddress("0xb02A2EdA1b317FBd16760128836B0Ac59B560e9D")
		value    = new(big.Int).Mul(big.NewInt(1), big.NewInt(params.Ether))
		sender   = common.HexToAddress(ADDR)
		gasLimit = uint64(21000)
	)
	// Retrieve the chainid (needed for signer)
	chainid, err := cl.ChainID(context.Background())
	if err != nil {
		return err
	}
	// Retrieve the pending nonce
	nonce, err := cl.PendingNonceAt(context.Background(), sender)
	if err != nil {
		return err
	}
	// Get suggested gas price
	tipCap, _ := cl.SuggestGasTipCap(context.Background())
	feeCap, _ := cl.SuggestGasPrice(context.Background())
	// Create a new transaction
	tx := types.NewTx(
		&types.DynamicFeeTx{
			ChainID:   chainid,
			Nonce:     nonce,
			GasTipCap: tipCap,
			GasFeeCap: feeCap,
			Gas:       gasLimit,
			To:        &to,
			Value:     value,
			Data:      nil,
		})
	// Sign the transaction using our keys
	signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(chainid), sk)
	// Send the transaction to our node
	return cl.SendTransaction(context.Background(), signedTx)
}
```

### gethclient

An instance of `gethclient` can be used in exactly the same way as `ethclient`. However, `gethclient` 
also includes Geth-specific API endpoints. These additional endpoints are:

```shell
CallContract()
CreatAccessList()
GCStats()
GetNodeInfo()
GetProof()
MemStats()
SetHead()
SubscribePendingTransactions()
```
*Note that both `ethclient` and `gethclient` have a `CallContract()` function - the difference is that 
the `gethclient` version includes an `overrides` argument.*

Details relating to these endpoints can be found at [pkg.go.dev][go-api-docs] or the Geth [Github][ethclient-link]. 
The code snippets in this tutorial were adapted from a more more in-depth set of examples available on 
[Github][web3go-link]. 


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
[ethclient-link]: https://github.com/ethereum/go-ethereum/tree/master/ethclient
[go-api-docs]:https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.19/ethclient/gethclient
[web3go-link]:https://github.com/MariusVanDerWijden/web3go