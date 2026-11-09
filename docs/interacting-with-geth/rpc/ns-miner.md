---
title: miner Namespace
description: Documentation for the JSON-RPC API "miner" namespace
---

The `miner` API changes the settings this node uses when building blocks. The methods that controlled proof-of-work mining (`miner_getHashrate`, `miner_setEtherbase`, `miner_setRecommitInterval`, `miner_start` and `miner_stop`) have been removed.

## miner_setExtra {#miner-setextra}

Sets the extra data to include in blocks built by this node. This is capped at 32 bytes.

| Client  | Method invocation                                  |
| :------ | -------------------------------------------------- |
| Go      | `miner.SetExtra(extra string) (bool, error)`       |
| Console | `miner.setExtra(string)`                           |
| RPC     | `{"method": "miner_setExtra", "params": [string]}` |

## miner_setGasPrice {#miner-setgasprice}

Sets the minimum gas tip accepted by the transaction pool and the block builder. Over RPC the value is a hex string such as `"0x1"`; a JSON number is rejected.

| Client  | Method invocation                                     |
| :------ | ----------------------------------------------------- |
| Go      | `miner.SetGasPrice(gasPrice hexutil.Big) bool`        |
| Console | `miner.setGasPrice(number)`                           |
| RPC     | `{"method": "miner_setGasPrice", "params": [number]}` |

## miner_setGasLimit {#miner-setgaslimit}

Sets the gas limit the node will target when building blocks. Note: on networks where EIP-1559 is activated, this should be set to twice the gas target (i.e. the effective gas used on average per block) to be. Over RPC the value is a hex string such as `"0x1c9c380"`; a JSON number is rejected.

| Client  | Method invocation                                     |
| :------ | ----------------------------------------------------- |
| Go      | `miner.SetGasLimit(gasLimit hexutil.Uint64) bool`     |
| Console | `miner.setGasLimit(number)`                           |
| RPC     | `{"method": "miner_setGasLimit", "params": [number]}` |
