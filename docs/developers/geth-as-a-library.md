---
title: Geth as a Library
description: Geth is not just a client. It is also a composable Go library for building custom Ethereum applications and infrastructure.
---

Geth is commonly known as a command-line Ethereum client, but it is also a well-structured Go library that can be imported into any Go project. The library code (everything outside `cmd/`) is licensed under LGPL v3, deliberately chosen to allow third-party use as a library. This means developers can build custom Ethereum nodes, specialized tooling, or entirely new infrastructure on top of battle-tested code that has been securing Ethereum mainnet since 2015.

This page provides an overview of what's possible. Each section links to deeper documentation where available.

## How Geth is structured {#architecture}

Geth's architecture separates cleanly into a **node container** and the **services** that run inside it:

```
┌─────────────────────────────────────────────┐
│  node.Node (the container)                  │
│                                             │
│  ┌────────────┐  ┌────────────────────────┐ │
│  │ RPC Server │  │ P2P Server             │ │
│  │ HTTP/WS/IPC│  │ devp2p, discovery      │ │
│  └────────────┘  └────────────────────────┘ │
│                                             │
│  ┌────────────────────────────────────────┐ │
│  │  Registered services (Lifecycle)       │ │
│  │  - eth.Ethereum (chain + EVM + txpool) │ │
│  │  - Your custom services                │ │
│  └────────────────────────────────────────┘ │
│                                             │
│  ┌────────────────────────────────────────┐ │
│  │  Account Manager + Key Store           │ │
│  └────────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
```

The [`node`](https://pkg.go.dev/github.com/ethereum/go-ethereum/node) package manages the lifecycle: RPC servers, peer-to-peer networking, data directory, and account management. Services register into the node and get started/stopped together with it. The standard `geth` binary is just one way to wire these pieces together. You can easily write your own.

## Building a custom node {#custom-node}

The `geth` binary at `cmd/geth/main.go` is roughly 200 lines of CLI glue. The core is simple: create a `node.Node`, register the Ethereum service, start the node. A minimal custom node looks like this:

```go
package main

import (
	"log"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
)

func main() {
	// Configure and create the node
	stack, err := node.New(&node.Config{
		DataDir:  "/tmp/my-ethereum-node",
		HTTPHost: "127.0.0.1",
		HTTPPort: 8545,
		HTTPModules: []string{"eth", "net", "web3"},
	})
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}
	defer stack.Close()

	// Register the Ethereum protocol
	ethCfg := ethconfig.Defaults
	if _, err := eth.New(stack, &ethCfg); err != nil {
		log.Fatalf("Failed to register Ethereum service: %v", err)
	}

	// Start the node
	if err := stack.Start(); err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}
	log.Println("Node running...")
	stack.Wait()
}
```

This gives you a fully functional Ethereum node — syncing, P2P networking, JSON-RPC — in under 30 lines of Go. From here you can customise every aspect: swap in a custom genesis, register additional services, add your own RPC endpoints, or change how blocks are processed.

## What you can customize {#what-you-can-customize}

The following areas are available for customization when building on Geth as a library. Each links to more detailed documentation where available.

### Custom RPC APIs

Register new JSON-RPC namespaces on the node with your own methods. Any Go struct with exported methods can become an RPC service:

```go
stack.RegisterAPIs([]rpc.API{{
	Namespace: "myapp",
	Service:   &MyCustomAPI{backend: ethBackend},
}})
```

Clients can then call `myapp_myMethod` via HTTP, WebSocket or IPC. Subscriptions (pub/sub over WebSocket) are also supported: just return an `rpc.Subscription` from your method.

See the [JSON-RPC docs](/docs/interacting-with-geth/rpc) for the full RPC system documentation.

### EVM and precompiles

Custom precompiled contracts can be implemented via the [`PrecompiledContract`](https://pkg.go.dev/github.com/ethereum/go-ethereum/core/vm#PrecompiledContract) interface — just `RequiredGas()` and `Run()`. The `core/vm/runtime` package also lets you execute EVM bytecode directly without a full node, useful for tooling and testing.

### Simulated blockchain

The [`ethclient/simulated`](https://pkg.go.dev/github.com/ethereum/go-ethereum/ethclient/simulated) package provides a full in-memory Ethereum blockchain for testing. It supports the entire `ethclient` API and lets you mine blocks on demand — no networking, no disk, no consensus client needed.

### Contract bindings

The `abigen` tool generates type-safe Go bindings from Solidity contract ABIs, giving you native Go functions for every contract method. See [Go contract bindings](/docs/developers/dapp-developer/native-bindings) and [v2 bindings](/docs/developers/dapp-developer/native-bindings-v2).

### P2P networking

Custom devp2p protocols can be registered on the node alongside the standard Ethereum protocol. This enables building application-specific peer-to-peer communication on top of Geth's networking stack.

### Transaction pool

The transaction pool is configurable with settings for price limits, account slot limits, and global pool sizes via `txpool.Config`. This lets you tune mempool behaviour for your specific use case.

## Key packages at a glance {#packages}

| Package | Purpose |
|---|---|
| [`node`](https://pkg.go.dev/github.com/ethereum/go-ethereum/node) | Node container, lifecycle, RPC server |
| [`eth`](https://pkg.go.dev/github.com/ethereum/go-ethereum/eth) | Full Ethereum service (chain, txpool, syncing) |
| [`eth/ethconfig`](https://pkg.go.dev/github.com/ethereum/go-ethereum/eth/ethconfig) | Configuration for the Ethereum service |
| [`ethclient`](https://pkg.go.dev/github.com/ethereum/go-ethereum/ethclient) | Go client for the Ethereum JSON-RPC API |
| [`ethclient/simulated`](https://pkg.go.dev/github.com/ethereum/go-ethereum/ethclient/simulated) | In-memory simulated blockchain for testing |
| [`rpc`](https://pkg.go.dev/github.com/ethereum/go-ethereum/rpc) | JSON-RPC server and client framework |
| [`accounts/abi/bind`](https://pkg.go.dev/github.com/ethereum/go-ethereum/accounts/abi/bind) | Contract binding generation and interaction |
| [`core/vm`](https://pkg.go.dev/github.com/ethereum/go-ethereum/core/vm) | EVM implementation, precompiles, runtime |
| [`p2p`](https://pkg.go.dev/github.com/ethereum/go-ethereum/p2p) | Peer-to-peer networking (devp2p) |
| [`core/txpool`](https://pkg.go.dev/github.com/ethereum/go-ethereum/core/txpool) | Transaction pool |

## Further reading {#further-reading}

- [Go API](/docs/developers/dapp-developer/native) — Programmatic access via `ethclient` and `gethclient`
- [Go Account Management](/docs/developers/dapp-developer/native-accounts) — Key management from Go
- [Go Contract Bindings](/docs/developers/dapp-developer/native-bindings) — Type-safe contract interaction
- [Go Contract Bindings v2](/docs/developers/dapp-developer/native-bindings-v2) — Next-generation bindings
- [Dev mode](/docs/developers/dapp-developer/dev-mode) — Local development networks
- [EVM tracing](/docs/developers/evm-tracing) — Custom tracers and live tracing
- [JSON-RPC](/docs/interacting-with-geth/rpc) — RPC server documentation
