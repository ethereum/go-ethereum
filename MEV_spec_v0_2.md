---
tags: spec
---

# MEV-Geth v0.2 specification

## Simple Summary

Defines the construction and usage of MEV bundles by miners. Provides a specification for custom implementations of the required node changes so that MEV bundles can be used correctly.

## Abstract

`MevBundles` are stored by the node and the bundles that are providing extra profit for miners are added to the block in front of other transactions.

## Motivation

We believe that without the adoption of neutral, public, open-source infrastructure for permissionless MEV extraction, MEV risks becoming an insiders' game. We commit as an organisation to releasing reference implementations for participation in fair, ethical, and politically neutral MEV extraction.

## Specification

The key words `MUST`, `MUST NOT`, `REQUIRED`, `SHALL`, `SHALL NOT`, `SHOULD`, `SHOULD NOT`, `RECOMMENDED`,  `MAY`, and `OPTIONAL` in this document are to be interpreted as described in [RFC-2119](https://www.ietf.org/rfc/rfc2119.txt).

### Definitions

#### `Bundle`
A set of transactions that `MUST` be executed together, `MUST` be executed before any non-bundle transactions only after other bundles that have a higher `simulated MEV equivalent gas price`, and `MUST` execute without failure (return status 1 on transaction receipts).

#### `Unit of work`
A `transaction`, a `bundle` or a `block`.

#### `Subunit`
A discernible `unit of work` that is a part of a bigger `unit of work`. A `transaction` is a `subunit` of a `bundle` or a `block`. A `bundle` is a `subunit` of a `block`.

#### `Total gas used`
A sum of gas units used by each transaction from the `unit of work`.

#### `Average gas price`
For a transaction it is equivalent to the transaction gas price and for other `units of work` it is a sum of (`average gas price` * `total gas used`) of all `subunits` divided by the `total gas used` of the unit.

#### `Tail gas price`
The greater of the 80 GWei and the gas price of the last transaction of the parent block.

#### `Direct coinbase payment`
A value of a transaction with a recipient set to be the same as the `coinbase` address.

#### `Contract coinbase payment`
A payment from a smart contract to the `coinbase` address.

#### `Coinbase payment`
A sum of all `direct coinbase payments` and `contract coinbase payments` within the `unit of work`.

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

#### `MEV equivalent gas price`
`Coinbase payment` divided by the `total gas used` by the `unit of work`.

#### `MevBundle` 
An object with four properties:

|Property| Type|Description|
|-|-|-|
|`transactions`|`Array<RLP(SignedTransaction)>`|A list of transactions in the bundle. Each transaction is signed and RLP-encoded.|
|`blockNumber`|`uint64`|The exact block number at which the bundle can be executed|
|`minTimestamp`|`uint64`|Minimum block timestamp (inclusive) at which the bundle can be executed|
|`maxTimestamp`|`uint64`|Maximum block timestamp (inclusive) at which the bundle can be executed|

### Bundle construction

A bundle `SHOULD` contain transactions with nonces that are following the current nonces of the signing addresses or other transactions preceding them in the same bundle.

A bundle `MUST` contain at least one transaction. There is no upper limit for the number of transactions in the bundle, however bundles that exceed the block gas limit will always be rejected. 

A bundle `MAY` include a `direct coinbase payment` or a `contract coinbase payment`. Bundles that do not contain such payments may be discarded when their `MEV equivalent gas price` is compared with other bundles or the `tail gas price`.

The `maxTimestamp` value `MUST` be greater or equal the `minTimestamp` value.

### Accepting bundles from the network

Node `MUST` provide a way of exposing a JSON RPC endpoint accepting `eth_sendBundle` calls (specified [here](https://hackmd.io/kbW7uxzqS_6Bi2xi9VI2ww)). Such endpoint `SHOULD` only be accepting calls from `MEV-relay` but there is no requirement to restrict it through the node source code as it can be done on the infrastructure level.

### Bundle eligibility

Any bundle that is correctly constructed `MUST` have a `blockNumber` field set which specifies in which block it can be included. If the node has already progressed to a later block number then such bundle `MAY` be removed from memory.

Any bundle that is correctly constructed `MAY` have a `minTimestamp` and/or a `maxTimestamp` field set. Default values for both of these fields are `0` and the meaning of `0` is that any block timestamp value is accepted. When these values are not `0`, then `block.timestamp` is compared with them. If the current `block.timestamp` is greater than the `maxTimestamp` then the bundle `MUST NOT` be included in the block and `MAY` be removed from memory. If the `block.timestamp` is less than `minTimestamp` then the bundle `MUST NOT` be included in the block and `SHOULD NOT` be removed from memory (it awaits future blocks).

### Block construction

`MevBundles` `MUST` be sorted by their `MEV equivalent gas price` first and then one by one merged into one bundle as long as there is any gas left in the block and the `MEV equivalent gas price` is greater than the `tail gas price`. During the bundle merging process any `MEV bundle` that does not satisfy the gas price requirements is skipped and the following bundles are then tested. After all bundle are tested the remaining block gas is used for non-MEV transactions.

The node `SHOULD` be able to compare a `block profit` in cases when bundles are included (MEV block) and when no bundles are included (regular block) and choose a block with the highest `profit`.

A block with bundles `MUST` place the bundles at the beginning of the block and `MUST NOT` insert any transactions between the bundles or bundle transactions.

### Bundle eviction

Node `SHOULD` be able to limit the number of bundles kept in memory and apply an algorithm for selecting bundles to be evicted when too many eligible bundles have been received.

## Rationale

### Min 80 GWei tail gas price.

We pick an arbitrary number for this intial proposal. Min 80 GWei `tail gas price` addresses some problems including empty parent blocks.

### Naive bundle merging

The bundle merging process is not necessarily picking the most profitable combination of bundles but the best guess achievable without degrading latency. The first bundle included is always the bundle with the highest `MEV equivalent gas price`

### Using MEV equivalent gas price instead of adjusted gas price

The `MEV equivalent gas price` is used to prevent bundle creators from artificially increasing the `adjusted gas price` by adding unrelated high gas price transactions.

### Each bundle needs a blockNumber

This allows specifying bundles to be included in the future blocks (e.g. just after some smart contracts change their state). This cannot be used to ensure a specific parent block / hash.

## Future Considerations

### Full block submission

A proposal to allow MEV-Geth accepting fully constructed blocks as well as bundles is considered for inclusion in v0.2.

### Contract coinbase payments via proxy payment contract

A purposefully crafted proxy payment contract is proposed as a requirement and the only payment method considered for `MEV equivalent gas price` calculation.

### Web sockets endpoints

A web socket endpoint for bundles submission is considered as a replacement to the current HTTP enpoints.

## Backwards Compatibility

This change is not affecting consensus and is fully backwards compatible.

## Security Considerations

`MevBundles` that are awaiting future blocks must be stored by the miner's node and it is important to ensure that there is a mechanism to ensure that the storage is limits are not exceeded (whether they are store in memory or persisted).