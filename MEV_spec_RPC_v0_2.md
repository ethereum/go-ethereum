---
tags: spec
---

# MEV-Geth RPC v0.2

# eth_sendBundle

### Description

Sends a bundle of transactions to the miner. The bundle has to be executed at the beginning of the block (before any other transactions), with bundle transactions executed exactly in the same order as provided in the bundle. During the Flashbots Alpha this is only called by the Flashbots relay.

| Name | Type | Description | Comment
--------|----------|-----------|-----------
txs |	`Array<Data>` | Array of signed transactions (`eth_sendRawTransaction` style, signed and RLP-encoded)	| a no-op in the light mode
blockNumber	|`Quantity`	|Exact block number at which the bundle can be included.	|bundle is evicted after the block
minTimestamp	|`Quantity`	|Minimum (inclusive) block timestamp at which the bundle can be included. If this value is 0 then any timestamp is acceptable.
maxTimestamp	|`Quantity`	|Maximum (inclusive) block timestamp at which the bundle can be included. If this value is 0 then any timestamp is acceptable.
revertingTxHashes	|Array<`Data`>	|Array of tx hashes within the bundle that are allowed to cause the EVM execution to revert without preventing the bundle inclusion in a block.

### Returns

{`boolean`} - `true` if bundle has been accepted by the node, otherwise `false`

### Example

```bash
# Request
curl -X POST --data '{
    "id": 1337,
    "jsonrpc": "2.0",
    "method": "eth_sendBundle",
    "params": [
        {
          "txs" : [
              "f9014946843b9aca00830493e094a011e5f4ea471ee4341a135bb1a4af368155d7a280b8e40d5f2659000000000000000000000000fdd45a22dd1d606b3782f2119621e928e32743000000000000000000000000000000000000000000000000000000000077359400000000000000000000000000000000000000000000000",
              "f86e8204d085012a05f200830c350094daf24c20717f428f00d8448d74d67a77f67ceb8287354a6ba7a18000802ea00e411bcb660dd8d47717df89078d2e8160c08e7f11cb7ad0ee935e7436eceb32a013ee00a21b7fa0a9f9c1224d11261648191875d4633aed6003543ea319f12b62"
          ],
          "blockNumber" : "0x12ab34",
          "minTimestamp" : "0x0",
          "minTimestamp" :"0x0"
        }
    ]
}' <url>

# Response
{
    "id": 1337,
    "jsonrpc": "2.0",
    "result": "true"
}
```

# eth_callBundle

### Description

Simulate a bundle of transactions at the top of a block.

After retrieving the block specified in the `blockNrOrHash` it takes the same `blockhash`, `gasLimit`, `difficulty`, same `timestamp` unless the `blockTimestamp` property is specified, and increases the block number by `1`. `eth_callBundle` will timeout after `5` seconds.

| Name | Type | Description |
| ---- | ---- | ----------- |
| encodedTxs | `Array<Data>` | Array of signed transactions (`eth_sendRawTransaction` style, signed and RLP-encoded) |
| blockNrOrHash	| `Quantity\|string\|Block Identifier` | Block number, or one of "latest", "earliest" or "pending", or a block identifier as described in {Block Identifier} |
| blockTimestamp	|`Quantity`	|Block timestamp to be used in replacement of the timestamp taken from the parent block. |

### Returns

Map<`Data`, "error|value" : `Data`> - a mapping from transaction hashes to execution results with error or output (value) for each of the transactions

### Example

```bash
# Request
curl -X POST --data '{
    "id": 1337,
    "jsonrpc": "2.0",
    "method": "eth_callBundle",
    "params": [
        [
            "f9014946843b9aca00830493e094a011e5f4ea471ee4341a135bb1a4af368155d7a280b8e40d5f2659000000000000000000000000fdd45a22dd1d606b3782f2119621e928e32743000000000000000000000000000000000000000000000000000000000077359400000000000000000000000000000000000000000000000",
            "f86e8204d085012a05f200830c350094daf24c20717f428f00d8448d74d67a77f67ceb8287354a6ba7a18000802ea00e411bcb660dd8d47717df89078d2e8160c08e7f11cb7ad0ee935e7436eceb32a013ee00a21b7fa0a9f9c1224d11261648191875d4633aed6003543ea319f12b62"
        ],
        "0x12ab34"
    ]
}' <url>

# Response
{
    "id": 1337,
    "jsonrpc": "2.0",
    "result":
        {
            "0x22b3806fbef9532db4105475222983404783aacd4d865ea5dab76a84aa1a07eb" : {
                "value" : "0x0012"
            },
            "0x489e3b5493af31d55059f8e296351b267720bc4ba7dc170871c1d789e5541027" : {
                "value" : "0xabcd"
            }
        }
}
```

---

Below type description can also be found in [EIP-1474](https://eips.ethereum.org/EIPS/eip-1474)

### `Quantity`

- A `Quantity` value **MUST** be hex-encoded.
- A `Quantity` value **MUST** be "0x"-prefixed.
- A `Quantity` value **MUST** be expressed using the fewest possible hex digits per byte.
- A `Quantity` value **MUST** express zero as "0x0".

### `Data`

- A `Data` value **MUST** be hex-encoded.
- A `Data` value **MUST** be “0x”-prefixed.
- A `Data` value **MUST** be expressed using two hex digits per byte.

### `Block Identifier`

Since there is no way to clearly distinguish between a `Data` parameter and a `Quantity` parameter, [EIP-1898](https://eips.ethereum.org/EIPS/eip-1898) provides a format to specify a block either using the block hash or block number. The block identifier is a JSON `object` with the following fields:

| Position | Name | Type | Description |
| -------- | ---- | ---- | ------------|
| 0A	|blockNumber	|`Quantity`	|The block in the canonical chain with this number |
| 0B	|blockHash	|`Data`	| The block uniquely identified by this hash. The blockNumber and blockHash properties are mutually exclusive; exactly one of them must be set. |
| 1B	|requireCanonical	|`boolean`	| (optional) Whether or not to throw an error if the block is not in the canonical chain as described below. Only allowed in conjunction with the blockHash tag. Defaults to false. |


If the block is not found, the callee SHOULD raise a JSON-RPC error (the recommended error code is `-32001: Resource not found`. If the tag is `blockHash` and `requireCanonical` is `true`, the callee SHOULD additionally raise a JSON-RPC error if the block is not in the canonical chain (the recommended error code is `-32000: Invalid input` and in any case should be different than the error code for the block not found case so that the caller can distinguish the cases). The block-not-found check SHOULD take precedence over the block-is-canonical check, so that if the block is not found the callee raises block-not-found rather than block-not-canonical.