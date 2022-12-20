---
title: net Namespace
description: Documentation for the JSON-RPC API "net" namespace
---

The `net` API provides insight about the networking aspect of the client.

## net_listening {#net-listening}

Returns an indication if the node is listening for network connections.

| Client  | Method invocation             |
| :------ | ----------------------------- |
| Console | `net.listening`               |
| RPC     | `{"method": "net_listening"}` |

## net_peerCount {#net-peercount}

Returns the number of connected peers.

| Client  | Method invocation             |
| :------ | ----------------------------- |
| Console | `net.peerCount`               |
| RPC     | `{"method": "net_peerCount"}` |

## net_version {#net-version}

Returns the devp2p network ID (e.g. 1 for mainnet, 5 for goerli).

| Client  | Method invocation           |
| :------ | --------------------------- |
| Console | `net.version`               |
| RPC     | `{"method": "net_version"}` |
