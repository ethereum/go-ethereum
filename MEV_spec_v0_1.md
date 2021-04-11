---
tags: spec
---

# MEV-Geth v0.1 specification

## Simple Summary

Defines the construction and usage of MEV bundles by the miners. Provides specification for custom implementation of required node changes so that MEV bundles can be used correctly.

## Abstract

`MevBundles` are stored by the node and the best bundle is added to the block in front of other transactions. `MevBundles` are sorted by their `adjusted gas price`.

## Motivation

We believe that without the adoption of neutral, public, open-source infrastructure for permissionless MEV extraction, MEV risks becoming an insiders' game. We commit as an organisation to releasing reference implementations for participation in fair, ethical, and politically neutral MEV extraction.

## Specification

The key words `MUST`, `MUST NOT`, `REQUIRED`, `SHALL`, `SHALL NOT`, `SHOULD`, `SHOULD NOT`, `RECOMMENDED`,  `MAY`, and `OPTIONAL` in this document are to be interpreted as described in [RFC-2119](https://www.ietf.org/rfc/rfc2119.txt).

### Definitions

#### `Bundle`
A set of transactions that `MUST` be executed together and `MUST` be executed at the beginning of the block.

#### `Unit of work`
A `transaction`, a `bundle` or a `block`.

#### `Subunit`
A discernible `unit of work` that is a part of a bigger `unit of work`. A `transaction` is a `subunit` of a `bundle` or a `block`. A `bundle` is a `subunit` of a `block`.

#### `Total gas used`
A sum of gas units used by each transaction from the `unit of work`.

#### `Average gas price`
Sum of (`gas price` * `total gas used`) of all `subunits` divided by the `total gas used` of the unit.

#### `Direct coinbase payment`
A value of a transaction with a recipient set to be the same as the `coinbase` address.

#### `Contract coinbase payment`
A payment from a smart contract to the `coinbase` address.

#### `Profit`
A difference between the balance of the `coinbase` account at the end and at the beginning of the execution of a `unit of work`. We can measure a `transaction profit`, a `bundle profit`, and a `block profit`.

Balance of the `coinbase` account changes in the following way
|Unit of work|Balance Change|
|-|-|
|Transaction| `average gas price` * `total gas used` + `direct coinbase payment`  + `contract coinbase payment`  |
|Bundle | `average gas price` * `total gas used` + `direct coinbase payment`  + `contract coinbase payment` |
|Block | `block reward` + `average gas price` * `total gas used` + `direct coinbase payment`  + `contract coinbase payment` |

#### `Adjusted gas price`
`Unit of work` `profit` divided by the `total gas used` by the `unit of work`.

#### `MevBundle` 
An object with four properties:

|Property| Type|Description|
|-|-|-|
|`transactions`|`Array<RLP(SignedTransaction)>`|A list of transactions in the bundle. Each transaction is signed and RLP-encoded.|
|`blockNumber`|`uint64`|The exact block number at which the bundle can be executed|
|`minTimestamp`|`uint64`|Minimum block timestamp (inclusive) at which the bundle can be executed|
|`maxTimestamp`|`uint64`|Maximum block timestamp (inclusive) at which the bundle can be executed|

### Bundle construction

Bundle `SHOULD` contain transactions with nonces that are following the current nonces of the signing addresses or other transactions preceding them in the same bundle.

A bundle `MUST` contain at least one transaction. There is no upper limit for the number of transactions in the bundle, however bundles that exceed the block gas limit will always be rejected. 

A bundle `MAY` include a `direct coinbase payment` or a `contract coinbase payment`. Bundles that do not contain such payments may lose comparison when their `profit` is compared with other bundles.

The `maxTimestamp` value `MUST` be greater or equal the `minTimestamp` value.

### Accepting bundles from the network

Node `MUST` provide a way of exposing a JSON RPC endpoint accepting `eth_sendBundle` calls (specified [here](MEV_spec_RPC_v0_1.md)). Such endpoint `SHOULD` only be accepting calls from `MEV-relay` but there is no requirement to restrict it through the node source code as it can be done on the infrastructure level.

### Bundle eligibility

Any bundle that is correctly constructed `MUST` have a `blockNumber` field set which specifies in which block it can be included. If the node has already progressed to a later block number then such bundle `MAY` be removed from memory.

Any bundle that is correctly constructed `MAY` have a `minTimestamp` and/or a `maxTimestamp` field set. Default values for both of these fields are `0` and the meaning of `0` is that any block timestamp value is accepted. When these values are not `0`, then `block.timestamp` is compared with them. If the current `block.timestamp` is greater than the `maxTimestamp` then the bundle `MUST NOT` be included in the block and `MAY` be removed from memory. If the `block.timestamp` is less than `minTimestamp` then the bundle `MUST NOT` be included in the block and `SHOULD NOT` be removed from memory (it awaits future blocks).

### Block construction

A block `MUST` either contain one bundle or no bundles. When a bundle is included it `MUST` be the bundle with the highest `adjusted gas price` among eligible bundles. The node `SHOULD` be able to compare a `block profit` in cases when a bundle is included (MEV block) and when no bundles are included (regular block) and choose a block with the highest `profit`.

A block with a bundle `MUST` place the bundle at the beginning of the block and `MUST NOT` insert any transactions between the bundle transactions.

### Bundle eviction

Node `SHOULD` be able to limit the number of bundles kept in memory and apply an algorithm for selecting bundles to be evicted when too many eligible bundles have been received.

## Rationale

### At most one MevBundle gets included in the block

There are two reasons for which multiple bundles in a block may cause problems:

- two bundles may affect each other's `profit` and so the bundle creator may not be willing to accept a possibility of not being added in the front of the block
- simulating multiple bundle combinations may be very straining for the node infrastructure and introduce excessive latency into the block creation process

Both of these problems may be addressed in the future versions.

## Each bundle needs a blockNumber

This allows specifying bundles to be included in the future blocks (e.g. just after some smart contracts change their state). This cannot be used to ensure a specific parent block / hash.

## Backwards Compatibility

This change is not affecting consensus and is fully backwards compatible.

## Security Considerations

`MevBundles` that are awaiting future blocks must be stored by the miner's node and it is important to ensure that there is a mechanism to ensure that the storage is limits are not exceeded (whether they are store in memory or persisted).