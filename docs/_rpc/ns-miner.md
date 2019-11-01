---
title: miner Namespace
sort_key: C
---

The `miner` API allows you to remote control the node's mining operation and set various
mining specific settings.

* TOC
{:toc}

### miner_getHashrate

Get your hashrate in H/s (Hash operations per second).

| Client  | Method invocation                                           |
|:--------|-------------------------------------------------------------|
| Console | `miner.getHashrate()`                                       |
| RPC     | `{"method": "miner_getHashrate", "params": []}`             |

### miner_setExtra

Sets the extra data a miner can include when miner blocks. This is capped at
32 bytes.

| Client  | Method invocation                                  |
|:--------|----------------------------------------------------|
| Go      | `miner.setExtra(extra string) (bool, error)`       |
| Console | `miner.setExtra(string)`                           |
| RPC     | `{"method": "miner_setExtra", "params": [string]}` |

### miner_setGasPrice

Sets the minimal accepted gas price when mining transactions. Any transactions that are
below this limit are excluded from the mining process.

| Client  | Method invocation                                     |
|:--------|-------------------------------------------------------|
| Go      | `miner.setGasPrice(number *rpc.HexNumber) bool`       |
| Console | `miner.setGasPrice(number)`                           |
| RPC     | `{"method": "miner_setGasPrice", "params": [number]}` |

### miner_start

Start the CPU mining process with the given number of threads and generate a new DAG
if need be.

| Client  | Method invocation                                   |
|:--------|-----------------------------------------------------|
| Go      | `miner.Start(threads *rpc.HexNumber) (bool, error)` |
| Console | `miner.start(number)`                               |
| RPC     | `{"method": "miner_start", "params": [number]}`     |

### miner_stop

Stop the CPU mining operation.

| Client  | Method invocation                            |
|:--------|----------------------------------------------|
| Go      | `miner.Stop() bool`                          |
| Console | `miner.stop()`                               |
| RPC     | `{"method": "miner_stop", "params": []}`     |

### miner_setEtherbase

Sets the etherbase, where mining rewards will go.

| Client  | Method invocation                                           |
|:--------|-------------------------------------------------------------|
| Go      | `miner.SetEtherbase(common.Address) bool`                   |
| Console | `miner.setEtherbase(address)`                               |
| RPC     | `{"method": "miner_setEtherbase", "params": [address]}`     |
