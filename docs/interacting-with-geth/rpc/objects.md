---
title: Objects
description: Data structures used for RPC methods
---

The following are data structures which are used for various RPC methods.

## Transaction call object {#transaction-call-object}

The _transaction call object_ contains all the necessary parameters for executing an EVM contract method.

| Field                  | Type         | Bytes | Optional | Description                                                                                                                                                                                                                                                                               |
| :--------------------- | :----------- | :---- | :------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `from`                 | `Address`    | 20    | Yes      | Address the transaction is simulated to have been sent from. Defaults to first account in the local keystore or the `0x00..0` address if no local accounts are available.                                                                                                                 |
| `to`                   | `Address`    | 20    | No       | Address the transaction is sent to.                                                                                                                                                                                                                                                       |
| `gas`                  | `Quantity`   | <8    | Yes      | Maximum gas allowance for the code execution to avoid infinite loops. Defaults to `2^63` or whatever value the node operator specified via `--rpc.gascap`.                                                                                                                                |
| `gasPrice`             | `Quantity`   | <32   | Yes      | Number of `wei` to simulate paying for each unit of gas during execution. Defaults to `1 gwei`.                                                                                                                                                                                           |
| `maxFeePerGas`         | `Quantity`   | <32   | Yes      | Maximum fee per gas the transaction should pay in total. Relevant for type-2 transactions.                                                                                                                                                                                                |
| `maxPriorityFeePerGas` | `Quantity`   | <32   | Yes      | Maximum tip per gas that's given directly to the miner. Relevant for type-2 transactions.                                                                                                                                                                                                 |
| `value`                | `Quantity`   | <32   | Yes      | Amount of `wei` to simulate sending along with the transaction. Defaults to `0`.                                                                                                                                                                                                          |
| `nonce`                | `Quantity`   | <8    | Yes      | Nonce of sender account.                                                                                                                                                                                                                                                                  |
| `input`                | `Binary`     | any   | Yes      | Binary data to send to the target contract. Generally the 4 byte hash of the method signature followed by the ABI encoded parameters. For details please see the [Ethereum Contract ABI](https://docs.soliditylang.org/en/v0.7.0/abi-spec.html). This field was previously called `data`. |
| `accessList`           | `AccessList` | any   | Yes      | A list of addresses and storage keys that the transaction plans to access. Used in non-legacy, i.e. type 1 and 2 transactions.                                                                                                                                                            |
| `chainId`              | `Quantity`   | <32   | Yes      | Transaction only valid on networks with this chain ID. Used in non-legacy, i.e. type 1 and 2 transactions.                                                                                                                                                                                |
| `maxFeePerBlobGas`     | `Quantity`   | <32   | Yes      | Max fee per blob gas the transaction should pay in total. Relevant for blob transactions.  |
| `blobVersionedHashes`  | `Array`      | any   | Yes      | Blob versioned hashes that can be accessed from EVM. They will be validated in case `blobs` also provided. Relevant for blob transactions.       |
| `blobs`                | `Array`      | any   | Yes      | EIP-4844 Blobs.       |
| `commitments`          | `Array`      | any   | Yes      | Commitments to EIP-4844 Blobs. They will be generated if only `blobs` are present and validated if both `blobs` and `commitments` are provided.      |
| `proofs`               | `Array`      | any   | Yes      | Proofs for EIP-4844 Blobs. They must be provided along with `commitments`. Else they will be generated.    |

Example for a legacy transaction:

```json
{
  "from": "0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3",
  "to": "0xebe8efa441b9302a0d7eaecc277c09d20d684540",
  "gas": "0x1bd7c",
  "data": "0xd459fc46000000000000000000000000000000000000000000000000000000000046c650dbb5e8cb2bac4d2ed0b1e6475d37361157738801c494ca482f96527eb48f9eec488c2eba92d31baeccfb6968fad5c21a3df93181b43b4cf253b4d572b64172ef000000000000000000000000000000000000000000000000000000000000008c00000000000000000000000000000000000000000000000000000000000000e0000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000001a00000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000001c000000000000000000000000000000000000000000000000000000000000001c0000000000000000000000000000000000000000000000000000000000000002b85c0c828d7a98633b4e1b65eac0c017502da909420aeade9a280675013df36bdc71cffdf420cef3d24ba4b3f9b980bfbb26bd5e2dcf7795b3519a3fd22ffbb2000000000000000000000000000000000000000000000000000000000000000238fb6606dc2b5e42d00c653372c153da8560de77bd9afaba94b4ab6e4aa11d565d858c761320dbf23a94018d843772349bd9d92301b0ca9ca983a22d86a70628"
}
```

Example for a type-1 transaction:

```json
{
  "from": "0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3",
  "to":   "0xebe8efa441b9302a0d7eaecc277c09d20d684540",
  "gas":  "0x1bd7c",
  "data": "0xd459fc46000000000000000000000000000000000000000000000000000000000046c650dbb5e8cb2bac4d2ed0b1e6475d37361157738801c494ca482f96527eb48f9eec488c2eba92d31baeccfb6968fad5c21a3df93181b43b4cf253b4d572b64172ef000000000000000000000000000000000000000000000000000000000000008c00000000000000000000000000000000000000000000000000000000000000e0000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000001a00000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000001c000000000000000000000000000000000000000000000000000000000000001c0000000000000000000000000000000000000000000000000000000000000002b85c0c828d7a98633b4e1b65eac0c017502da909420aeade9a280675013df36bdc71cffdf420cef3d24ba4b3f9b980bfbb26bd5e2dcf7795b3519a3fd22ffbb2000000000000000000000000000000000000000000000000000000000000000238fb6606dc2b5e42d00c653372c153da8560de77bd9afaba94b4ab6e4aa11d565d858c761320dbf23a94018d843772349bd9d92301b0ca9ca983a22d86a70628",
  "chainId": "0x1",
  "accessList":  [
    {
      "address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
      "storageKeys": ["0xda650992a54ccb05f924b3a73ba785211ba39a8912b6d270312f8e2c223fb9b1", "0x10d6a54a4754c8869d6886b5f5d7fbfa5b4
                  522237ea5c60d11bc4e7a1ff9390b"]
    }, {
      "address": "0xa2327a938febf5fec13bacfb16ae10ecbc4cbdcf",
      "storageKeys": []
    },
  ]
}
```

Example for a type-2 transaction:

```json
{
  "from": "0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3",
  "to": "0xebe8efa441b9302a0d7eaecc277c09d20d684540",
  "gas": "0x1bd7c",
  "maxFeePerGas": "0x6b44b0285",
  "maxPriorityFeePerGas": "0x6b44b0285",
  "data": "0xd459fc46000000000000000000000000000000000000000000000000000000000046c650dbb5e8cb2bac4d2ed0b1e6475d37361157738801c494ca482f96527eb48f9eec488c2eba92d31baeccfb6968fad5c21a3df93181b43b4cf253b4d572b64172ef000000000000000000000000000000000000000000000000000000000000008c00000000000000000000000000000000000000000000000000000000000000e0000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000001a00000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000001c000000000000000000000000000000000000000000000000000000000000001c0000000000000000000000000000000000000000000000000000000000000002b85c0c828d7a98633b4e1b65eac0c017502da909420aeade9a280675013df36bdc71cffdf420cef3d24ba4b3f9b980bfbb26bd5e2dcf7795b3519a3fd22ffbb2000000000000000000000000000000000000000000000000000000000000000238fb6606dc2b5e42d00c653372c153da8560de77bd9afaba94b4ab6e4aa11d565d858c761320dbf23a94018d843772349bd9d92301b0ca9ca983a22d86a70628",
  "chainId": "0x1",
  "accessList": []
}
```

## State override set {#state-override-set}

The _state override set_ is an optional address-to-state mapping, used in `eth_call` and `eth_simulateV1`, where each entry specifies some state to be ephemerally overridden prior to executing the call. Each address maps to an object containing:

| Field                     | Type       | Bytes | Optional | Description                                                                                               |
| :------------------------ | :--------- | :---- | :------- | :-------------------------------------------------------------------------------------------------------- |
| `balance`                 | `Quantity` | <32   | Yes      | Fake balance to set for the account before executing the call.                                            |
| `nonce`                   | `Quantity` | <8    | Yes      | Fake nonce to set for the account before executing the call.                                              |
| `code`                    | `Binary`   | any   | Yes      | Fake EVM bytecode to inject into the account before executing the call.                                   |
| `state`                   | `Object`   | any   | Yes      | Fake key-value mapping to override **all** slots in the account storage before executing the call.        |
| `stateDiff`               | `Object`   | any   | Yes      | Fake key-value mapping to override **individual** slots in the account storage before executing the call. |
| `movePrecompileToAddress` | `address`  | 20    | Yes      | Moves precompile to given address                                                                         |                                                                                                          |

The goal of the _state override set_ is manyfold:

- It can be used by websites to reduce the amount of contract code needed to be deployed on chain. Code that simply returns internal state or does pre-defined validations can be kept off chain and fed to the node on-demand.
- It can be used for smart contract analysis by extending the code deployed on chain with custom methods and invoking them. This avoids having to download and reconstruct the entire state in a sandbox to run custom code against.
- It can be used to debug smart contracts in an already deployed large suite of contracts by selectively overriding some code or state and seeing how execution changes. Specialized tooling will probably be necessary.
- It can be used to override `ecrecover` precompile to spoof signatures

**Example:**

```json
{
 "0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3": {
   "balance": "0xde0b6b3a7640000"
 },
 "0xebe8efa441b9302a0d7eaecc277c09d20d684540": {
   "code": "0x...",
   "state": {
	 ""
   }
 }
}
```

## Block overrides

The fields of this object customize the block as part of which a call is simulated. This object can be passed to eth_call, eth_simulateV1 as well as debug_traceCall methods and contains the following fields:

| Field           | Type          | Description |
| :-------------- | :------------ | :----------------------------------------------------------------------------------------------------------------------------------------- |
| `number`        | `uint64`      | Block number.                                                                                                                              | 
| `prevRandao`    | `uint256`     | The previous value of randomness beacon.                                                                                                   |
| `time`          | `uint64`      | Block timestamp.                                                                                                                           |
| `gasLimit`      | `uint64`      | Gas limit.                                                                                                                                 |
| `feeRecipient`  | `address`     | Fee recipient (also known as coinbase).                                                                                                    |
| `withdrawals`   | `Withdrawals` | Withdrawals made by validators.                                                                                                            |
| `baseFeePerGas` | `uint256`     | Base fee per unit of gas (see EIP-1559).                                                                                                   |
| `blobBaseFee`   | `uint64`      | Base fee per unit of blob gas (see EIP-4844).                                                                                              |

The object `withdrawals` is an array of withdrawal objects:

| Field            | Type      | Description     |
| :--------------- | :-------- | :-------------- |
| `index`          | `uint64`  | index           |
| `validatorIndex` | `uint64`  | validator index |
| `address`        | `address` | address         |
| `amount`         | `uint64`  | amount          |
