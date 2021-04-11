---
tags: spec
---

# MEV-Geth v0.2 specification

## Simple Summary

Defines the construction and usage of MEV bundles by miners. Provides a specification for custom implementations of the required node changes so that MEV bundles can be used correctly.

## Abstract

`MEV bundles` are stored by the node and the bundles that are providing extra profit for miners are added to the block in front of other transactions.

## Motivation

We believe that without the adoption of neutral, public, open-source infrastructure for permissionless MEV extraction, MEV risks becoming an insiders' game. We commit as an organisation to releasing reference implementations for participation in fair, ethical, and politically neutral MEV extraction.

## Specification

The key words `MUST`, `MUST NOT`, `REQUIRED`, `SHALL`, `SHALL NOT`, `SHOULD`, `SHOULD NOT`, `RECOMMENDED`,  `MAY`, and `OPTIONAL` in this document are to be interpreted as described in [RFC-2119](https://www.ietf.org/rfc/rfc2119.txt).

### Miner Configuration

Miner should accept the following configuration options:
* miner.strictprofitswitch (int) - time in miliseconds to wait for both the non-MEV (vanilla) and the `MEV block` construction before selecting the best available block for mining. If value is zero then no waiting is necessary.
* miner.maxmergedbundles (int) - max number of `MEV bundles` to be included within a single block
* relayWSURL (string) - address of the relay WS endpoint
* relayWSSigningKey (bytes32) - a signing key for communication with the relay's WebSockets endpoint. The key should be a valid private key from a secp256k1 based ECDSA.
* relayWSSigningKeystoreFile (string) - path to the keystore file with the WS signing key (used when signing key not provided)
* relayWSKeystorePW (string) - password to unlock the relay WS keystore file

### Definitions

#### `Relay`

An external system delivering `MEV bundles` to the node. Relay provides protection against DoS attacks.

#### `MEV bundle` or `bundle`

A list of transactions that `MUST` be executed together and in the same order as provided in the bundle, `MUST` be executed before any non-bundle transactions and only after other bundles that have a higher `bundle adjusted gas price`.

Transactions in the bundle `MUST` execute without failure (return status 1 on transaction receipts) unless their hashes are included in the `revertingTxHashes` list.

When representing a bundle in communication between the `relay` and the node we use an object with the following properties:

|Property| Type|Description|
|-|-|-|
|`txs`|`Array<RLP(SignedTransaction)>`|A list of transactions in the bundle. Each transaction is signed and RLP-encoded.|
|`blockNumber`|`uint64`|The exact block number at which the bundle can be executed|
|`minTimestamp`|`uint64`|Minimum block timestamp (inclusive) at which the bundle can be executed|
|`maxTimestamp`|`uint64`|Maximum block timestamp (inclusive) at which the bundle can be executed|
|`revertingTxHashes`|`Array<bytes32>`|List of hashes of transactions that are allowed to return status 0 on transaction receipts|

#### `MEV block`

A block containing more than zero 'MEV bundles'.

Whenever we say that a block contains a `bundle` we mean that the block includes all transactions of that bundle in the same order as in the `bundle`.

#### `Unit of work`
A `transaction`, a `bundle` or a `block`.

#### `Subunit`
A discernible `unit of work` that is a part of a bigger `unit of work`. A `transaction` is a `subunit` of a `bundle` or a `block`. A `bundle` is a `subunit` of a `block`.

#### `Total gas used`
The sum of gas units used by each transaction from the `unit of work`.

#### `Average gas price`
For a transaction it is equivalent to the transaction gas price and for other `units of work` it is a sum of (`average gas price` * `total gas used`) of all `subunits` divided by the `total gas used` by the unit.

#### `Direct coinbase payment`
The value of a transaction with a recipient set to be the same as the `coinbase` address.

#### `Contract coinbase payment`
A payment from a smart contract to the `coinbase` address.

#### `Coinbase payment`
A sum of all `direct coinbase payments` and `contract coinbase payments` within the `unit of work`.

#### `Eligible coinbase payment`
A sum of all `direct coinbase payments` and `contract coinbase payments` within the `unit of work`.

#### `Gas fee payment`
An `average gas price` * `total gas used` within the `unit of work`.

#### `Eligible gas fee payment`
A `gas fee payment` excluding `gas fee payments` from transactions that can be spotted by the miner in the publicly visible transaction pool.

#### `Bundle scoring profit`
A sum of all `eligible coinbase payments` and `eligible gas payments` of a `bundle`.

#### `Profit`
A difference between the balance of the `coinbase` account at the end and at the beginning of the execution of a `unit of work`. We can measure a `transaction profit`, a `bundle profit`, and a `block profit`.

Balance of the `coinbase` account changes in the following way
|Unit of work|Balance Change|
|-|-|
|Transaction| `average gas price` * `total gas used` + `direct coinbase payment` + `contract coinbase payment` |
|Bundle | `average gas price` * `total gas used` + `direct coinbase payment`  + `contract coinbase payment` |
|Block | `block reward` + `average gas price` * `total gas used` + `direct coinbase payment`  + `contract coinbase payment` |

#### `Adjusted gas price`
`Unit of work` `profit` divided by the `total gas used` by the `unit of work`.

#### `Bundle adjusted gas price`
`Bundle scoring profit` divided by the `total gas used` by the `bundle`.

### Bundle construction

A bundle `SHOULD` contain transactions with nonces that are following the current nonces of the signing addresses or other transactions preceding them in the same bundle.

A bundle `MUST` contain at least one transaction. There is no upper limit for the number of transactions in the bundle, however bundles that exceed the block gas limit will always be rejected. 

A bundle `MAY` include `eligible coinbase payments`. Bundles that do not contain such payments may be discarded when their `bundle adjusted gas price` is compared with other bundles.

The `maxTimestamp` value `MUST` be greater or equal the `minTimestamp` value.

### Accepting bundles from the network

#### JSON RPC

Node `MUST` provide a way of exposing a JSON RPC endpoint accepting `eth_sendBundle` calls (specified [here](MEV_spec_RPC_v0_2.md)). Such endpoint `SHOULD` only be accepting calls from the `relay` but there is no requirement to restrict it through the node source code as it can be done on the infrastructure level.

#### WebSockets

Node `MUST` be able to connect to the relay WebSocket endpoint provided as a configuration option named `RelayWSURL` using an authentication key and maintain an open connection. Authentication key `MUST` be provided with the key security in mind.

During the WebSocket connection initiation and on each reconnection, the node `MUST` sign a timestamp (unix epoch in seconds, UTC) using the `RelayWSSigningKey` and send it in the request header body (X-Auth-Message). The X-Auth-Message should be a JSON encoded object with three properties called timestamp, signature and coinbase.

### Bundle eligibility

Any bundle that is correctly constructed `MUST` have a `blockNumber` field set which specifies in which block it can be included. If the node has already progressed to a later block number then such bundle `MAY` be removed from memory.

Any bundle that is correctly constructed `MAY` have a `minTimestamp` and/or a `maxTimestamp` field set. Default values for both of these fields are `0` and the meaning of `0` is that any block timestamp value is accepted. When these values are not `0`, then `block.timestamp` is compared with them. If the current `block.timestamp` is greater than the `maxTimestamp` then the bundle `MUST NOT` be included in the block and `MAY` be removed from memory. If the `block.timestamp` is less than `minTimestamp` then the bundle `MUST NOT` be included in the block and `SHOULD NOT` be removed from memory (it awaits future blocks).

### Block construction

`MEV bundles` `MUST` be sorted by their `bundle adjusted gas price` first and then one by one added to the block as long as there is any gas left in the block and the number of bundles added is less or equal the `MaxMergedBundles` parameter. The remaining block gas `SHOULD` be used for non-MEV transactions.

Block `MUST` contain between 0 and `MaxMergedBundles` bundles.

A block with bundles `MUST` place the bundles at the beginning of the block and `MUST NOT` insert any transactions between the bundles or bundle transactions.

When constructing the block the node should reject any bundle that has a reverting transaction unless its hash is included in the `RevertingTxHashes` list of the bundle object.

The node `SHOULD` be able to compare the `block profit` for each number of bundles between 0 and `MaxMergedBundles` and choose a block with the highest `profit`, e.g. if `MaxMergedBundles` is 3 then the node `SHOULD` build 4 different blocks - with the maximum of respectively 0, 1, 2, and 3 bundles and choose the one with the highest `profit`.

When constructing blocks, if the 0 bundles block has not yet been constructed and the `StrictProfitSwitch` parameter is set to a value other than 0, then the node `MUST` wait `StrictProfitSwitch` miliseconds before accepting any MEV block for mining.

### Bundle eviction

Node `SHOULD` be able to limit the number of bundles kept in memory and apply an algorithm for selecting bundles to be evicted when too many eligible bundles have been received.

## Rationale

### Naive bundle merging

The bundle merging process is not necessarily picking the most profitable combination of bundles but only the best guess achievable without degrading latency. The first bundle included is always the bundle with the highest `bundle adjusted gas price`

### Using bundle adjusted gas price instead of adjusted gas price

The `bundle adjusted gas price` is used to prevent bundle creators from artificially increasing the `adjusted gas price` by adding unrelated high gas price transactions from the publicly visible transaction pool.

### Each bundle needs a blockNumber

This allows specifying bundles to be included in the future blocks (e.g. just after some smart contracts change their state). This cannot be used to ensure a specific parent block / hash.

## Future Considerations

### Full block submission

A proposal to allow MEV-Geth accepting fully constructed blocks as well as bundles is considered for inclusion in next versions.

## Backwards Compatibility

This change is not affecting consensus and is fully compatible with Ethereum specification.

Bundle formats are not backwards compatible and the v0.2 bundles would be rejected by v0.1 MEV clients.

## Security Considerations

`MEV bundles` that are awaiting future blocks must be stored by the node to ensure that the storage limits are not exceeded (whether they are store in memory or persisted).