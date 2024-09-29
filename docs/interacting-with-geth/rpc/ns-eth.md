---
title: eth Namespace
description: Documentation for the JSON-RPC API "eth" namespace
---

Documentation for the API methods in the `eth` namespace can be found on [ethereum.org](https://ethereum.org/en/developers/docs/apis/json-rpc/#eth_protocolversion). Geth provides several extensions to the standard "eth" JSON-RPC namespace that are defined below.

### eth_simulateV1 {#eth-simulate-v1}
The `eth_simulateV1` method allows the simulation of multiple blocks and transactions without creating transactions or blocks on the blockchain. It functions similarly to `eth_call`, but offers more control. Like `eth_call`, `eth_simulateV1` has a maximum gas limit for the entire simulation, this limit can be adjusted with command line parameter `--rpc.gascap`.

**Parameters:**
The method takes two parameters:
1) `Object` - `eth_simulate` payload
2) `Quantity | Tag` - The block number or the string `latest`, specifying the parent block for the simulation. The simulated blocks will be built on top of this.

The `eth_simulate` payload structure:
   | Field                    | Type              | Optional | Default | Description                                                                                                                                                                                                                     |
   | :----------------------- | :---------------- | :------- | :------ | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
   | `blockStateCalls`        | `BlockStateCalls` | False    | N/A     | Definition of blocks that can contain calls and overrides                                                                                                                                                                       |
   | `traceTransfers`         | `bool`            | Yes      | False   | Adds ETH transfers as ERC20 transfer events to the logs. These transfers have emitter contract parameter set as address(`0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee`). This allows you to track movements of ETH in your calls. |
   | `validation`             | `bool`            | Yes      | False   | When true, the `eth_simulateV1` does all the validation that a normal EVM would do, except contract sender and signature checks. When false, `eth_simulateV1` behaves like `eth_call`.                                          |
   | `returnFullTransactions` | `bool`            | Yes      | False   | When true, the method returns full transaction objects, otherwise, just hashes are returned.                                                                                                                                    |

`BlockStateCalls` is an array of objects, the single object definition is described below. The total number of blocks for a single request is limited to 256. This includes any gap blocks that client generated.
   | Field            | Type                       | Description                                                                                                  |
   | :--------------- | :------------------------- | :----------------------------------------------------------------------------------------------------------- |
   | `blockOverrides` | `BlockOverrides`           | Overrides fields such as block number or time in a simulated block.                                          |
   | `stateOverrides` | `StateOverrides`           | State overrides can be used to replace existing blockchain state with new state.                             |
   | `calls`          | `GenericCallTransaction[]` | An aray of transaction call objects. Please see [Transaction Call Object](/docs/interacting-with-geth/rpc/objects#transaction-call-object) for details. |

The optional `BlockOverrides` object modifies the context in which the transactions of that given block are executed. Refer to [Block overrides](/docs/interacting-with-geth/rpc/objects#block-overrides) for a list of modifiable fields. When overriding multiple blocks, block numbers must increment. Skipping numbers is allowed and skipped blocks are included in the response. When overriding time across multiple blocks, time need to be increasing. If time is not specified, it's incremented by one for each block.

The StateOverrides is an optional address-to-state mapping, where each entry specifies some state to be ephemerally overridden prior to executing each block. Please see [State Override Set](/docs/interacting-with-geth/rpc/objects#state-override-set) for details.

**Output**
On a succesfull `eth_simulateV1` call, an array of generated full blocks is returned (the same object that you would get with `eth_getBlockByHash`, except with an added `calls` field), otherwise an error is returned. The blocks contain `calls` field that is defined as follows:

On failure:
   | Field        | Type                                             | Description                                   |
   | :----------- | :----------------------------------------------- | :-------------------------------------------- |
   | `status`     | `"0x0"`                                          | Status indicating that the transaction failed |
   | `returnData` | `bytes`                                          | Transactions return data                      |
   | `gasUsed`    | `uint64`                                         | Gas used by the transaction                   |
   | `error`      | `{ code: uint64, message: string, data: bytes }` | Error code, data and message                  |

On success:
   | Field        | Type              | Description                                                                            |
   | :----------- | :---------------- | :------------------------------------------------------------------------------------- |
   | `status`     | `"0x1"`           | Status indicating that the transaction succeeded                                       |
   | `returnData` | `bytes`           | Transactions return data                                                               |
   | `gasUsed`    | `uint64`          | Gas used by the transaction                                                            |
   | `logs`       | `CallResultLog[]` | Log events emitted during call. This includes ETH logs, if `traceTransfers` is enabled |

the `CallResultLog` is object of form:
   | Field              | Type        | Description                                                                                                                                |
   | :----------------- | :---------- | :----------------------------------------------------------------------------------------------------------------------------------------- |
   | `logIndex`         | `uint256`   | Log index                                                                                                                                  |
   | `blockHash`        | `hash32`    | Block hash                                                                                                                                 |
   | `blockNumber`      | `uint64`    | Block number                                                                                                                               |
   | `transactionHash`  | `hash32`    | Transaction hash                                                                                                                           |
   | `transactionIndex` | `uint256`   | Transaction index                                                                                                                          |
   | `address`          | `address`   | Contract that sent the log. When trace transfers is enabled, this field is `0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee` for ETH transfers. |
   | `data`             | `bytes`     | Event data                                                                                                                                 |
   | `topics`           | `bytes32[]` | Array of topics                                                                                                                            |
   | `removed`          | `bool`      | Always false. A flag indicating if a log was removed in a chain reorganization, which cannot happen in eth_simulateV1.                     |
 
**Example:**
Here's an simple `eth_simulateV1` call that sets blocks `baseFeePerGas` to `9`, gives us `0.00000002` ETH and then we send ETH to two addresses. You can find that the output has two logs produced by these ETH sends. This is because we have set `traceTransfers` to true.
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "eth_simulateV1",
    "params": [
        {
            "blockStateCalls": [
                {
                    "blockOverrides": {
                        "baseFeePerGas": "0x9"
                    },
                    "stateOverrides": {
                        "0xc000000000000000000000000000000000000000": {
                            "balance": "0x4a817c800"
                        }
                    },
                    "calls": [
                        {
                            "from": "0xc000000000000000000000000000000000000000",
                            "to": "0xc000000000000000000000000000000000000001",
                            "maxFeePerGas": "0xf",
                            "value": "0x1"
                        },
                        {
                            "from": "0xc000000000000000000000000000000000000000",
                            "to": "0xc000000000000000000000000000000000000002",
                            "maxFeePerGas": "0xf",
                            "value": "0x1"
                        }
                    ]
                }
            ],
            "validation": true,
            "traceTransfers": true
        },
        "latest"
    ]
}
```

**Example response:**
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": [
        {
            "baseFeePerGas": "0x9",
            "blobGasUsed": "0x0",
            "calls": [
                {
                    "returnData": "0x",
                    "logs": [
                        {
                            "address": "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
                            "topics": [
                                "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
                                "0x000000000000000000000000c000000000000000000000000000000000000000",
                                "0x000000000000000000000000c000000000000000000000000000000000000001"
                            ],
                            "data": "0x0000000000000000000000000000000000000000000000000000000000000001",
                            "blockNumber": "0x13d2747",
                            "transactionHash": "0xe7217784e0c3f7b35d39303b1165046e9b7e8af9b9cf80d5d5f96c3163de8f51",
                            "transactionIndex": "0x0",
                            "blockHash": "0x5e28f54a56dc9df973a058cd54b3eeef8c67a1a613cb5db1df8a0a434c931d56",
                            "logIndex": "0x0",
                            "removed": false
                        }
                    ],
                    "gasUsed": "0x5208",
                    "status": "0x1"
                },
                {
                    "returnData": "0x",
                    "logs": [
                        {
                            "address": "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
                            "topics": [
                                "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
                                "0x000000000000000000000000c000000000000000000000000000000000000000",
                                "0x000000000000000000000000c000000000000000000000000000000000000002"
                            ],
                            "data": "0x0000000000000000000000000000000000000000000000000000000000000001",
                            "blockNumber": "0x13d2747",
                            "transactionHash": "0xf0182201606ec03701ba3a07d965fabdb4b7d06b424f226ea7ec3581802fc6fa",
                            "transactionIndex": "0x1",
                            "blockHash": "0x5e28f54a56dc9df973a058cd54b3eeef8c67a1a613cb5db1df8a0a434c931d56",
                            "logIndex": "0x1",
                            "removed": false
                        }
                    ],
                    "gasUsed": "0x5208",
                    "status": "0x1"
                }
            ],
            "difficulty": "0x0",
            "excessBlobGas": "0x0",
            "extraData": "0x",
            "gasLimit": "0x1c9c380",
            "gasUsed": "0xa410",
            "hash": "0x5e28f54a56dc9df973a058cd54b3eeef8c67a1a613cb5db1df8a0a434c931d56",
            "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
            "miner": "0x4838b106fce9647bdf1e7877bf73ce8b0bad5f97",
            "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
            "nonce": "0x0000000000000000",
            "number": "0x13d2747",
            "parentBeaconBlockRoot": "0x0000000000000000000000000000000000000000000000000000000000000000",
            "parentHash": "0xd24222b93a05a066cf79dc20e333f5aa6bb06d36eb50eb2b6b0b744b937e7975",
            "receiptsRoot": "0x75308898d571eafb5cd8cde8278bf5b3d13c5f6ec074926de3bb895b519264e1",
            "sha3Uncles": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
            "size": "0x298",
            "stateRoot": "0xbb0740745211507e2a2a6cdb627dfa171ef5050ad2a01e5401c2e3df4be5b919",
            "timestamp": "0x66ec2853",
            "totalDifficulty": "0xc70d815d562d3cfa955",
            "transactions": [
                "0xe7217784e0c3f7b35d39303b1165046e9b7e8af9b9cf80d5d5f96c3163de8f51",
                "0xf0182201606ec03701ba3a07d965fabdb4b7d06b424f226ea7ec3581802fc6fa"
            ],
            "transactionsRoot": "0x9bdb74f3ce41f5893a02a631e904ae0d21ae8c4e416786d8dbd9cb5c54f1dc0f",
            "uncles": [],
            "withdrawals": [],
            "withdrawalsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
        }
    ]
}
```

### eth_subscribe, eth_unsubscribe {#eth-subscribe-unsubscribe}

These methods are used for real-time events through subscriptions. See the [subscription documentation](/docs/interacting-with-geth/rpc/pubsub) for more information.

### eth_call {#eth-call}

Executes a new message call immediately, without creating a transaction on the block chain. The `eth_call` method can be used to query internal contract state, to execute validations coded into a contract or even to test what the effect of a transaction would be without running it live.

**Parameters:**

The method takes 4 parameters: an unsigned transaction object to execute in read-only mode; the block number to execute the call against; an optional state override-set to allow executing the call against a modified chain state; and an optional set of overrides for the block context.

1. `Object` - Transaction call object

   The _transaction call object_ is mandatory. Please see [here](/docs/interacting-with-geth/rpc/objects) for details.

2. `Quantity | Tag` - Block number or the string `latest` or `pending`

   The _block number_ is mandatory and defines the context (state) against which the specified transaction should be executed. It is not possible to execute calls against reorged blocks; or blocks older than 128 (unless the node is an archive node).

3. `Object` - State override set

   The _state override set_ is an optional address-to-state mapping, where each entry specifies some state to be ephemerally overridden prior to executing the call. Please see [State Override Set](/docs/interacting-with-geth/rpc/objects#state-override-set) for details.

4. `Object` - Block override set

   The _block override set_ is an optional object with the purpose of modifying the context in which the call is executed. Refer to [Block overrides](/docs/interacting-with-geth/rpc/objects#block-overrides) for a rundown of the fields.


**Response:**

The method returns a single `Binary` consisting the return value of the executed contract call.

#### eth_call Simple example

**note that this example uses the Rinkeby network, which is now deprecated**

With a synced Rinkeby node with RPC exposed on localhost (`geth --rinkeby --http`) we can make a call against the [CheckpointOracle](https://rinkeby.etherscan.io/address/0xebe8efa441b9302a0d7eaecc277c09d20d684540) to retrieve the list of administrators:

```sh
curl --data '{"method":"eth_call","params":[{"to":"0xebe8efa441b9302a0d7eaecc277c09d20d684540","data":"0x45848dfc"},"latest"],"id":1,"jsonrpc":"2.0"}' -H "Content-Type: application/json" -X POST localhost:8545
```

And the result is an Ethereum ABI encoded list of accounts:

```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": "0x00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000004000000000000000000000000d9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f300000000000000000000000078d1ad571a1a09d60d9bbf25894b44e4c8859595000000000000000000000000286834935f4a8cfb4ff4c77d5770c2775ae2b0e7000000000000000000000000b86e2b0ab5a4b1373e40c51a7c712c70ba2f9f8e"
}
```

Just for the sake of completeness, decoded the response is:

```sh
0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3,
0x78d1ad571a1a09d60d9bbf25894b44e4c8859595,
0x286834935f4a8cfb4ff4c77d5770c2775ae2b0e7,
0xb86e2b0ab5a4b1373e40c51a7c712c70ba2f9f8e
```

#### eth_call Override example

The above _simple example_ showed how to call a method already exposed by an on-chain smart contract. What if we want to access some data not exposed by it?

We can gut out the [original](https://github.com/ethereum/go-ethereum/blob/master/contracts/checkpointoracle/contract/oracle.sol) checkpoint oracle contract with one that retains the same fields (to retain the same storage layout), but one that includes a different method set:

```js
pragma solidity ^0.5.10;

contract CheckpointOracle {
    mapping(address => bool) admins;
    address[] adminList;
    uint64 sectionIndex;
    uint height;
    bytes32 hash;
    uint sectionSize;
    uint processConfirms;
    uint threshold;

    function VotingThreshold() public view returns (uint) {
        return threshold;
    }
}
```

With a synced Rinkeby node with RPC exposed on localhost (`geth --rinkeby --http`) we can make a call against the live [Checkpoint Oracle](https://rinkeby.etherscan.io/address/0xebe8efa441b9302a0d7eaecc277c09d20d684540), but override its byte code with our own version that has an accessor for the voting
threshold field:

```sh
curl --data '{"method":"eth_call","params":[{"to":"0xebe8efa441b9302a0d7eaecc277c09d20d684540","data":"0x0be5b6ba"}, "latest", {"0xebe8efa441b9302a0d7eaecc277c09d20d684540": {"code":"0x6080604052348015600f57600080fd5b506004361060285760003560e01c80630be5b6ba14602d575b600080fd5b60336045565b60408051918252519081900360200190f35b6007549056fea265627a7a723058206f26bd0433456354d8d1228d8fe524678a8aeeb0594851395bdbd35efc2a65f164736f6c634300050a0032"}}],"id":1,"jsonrpc":"2.0"}' -H "Content-Type: application/json" -X POST localhost:8545
```

And the result is the Ethereum ABI encoded threshold number:

```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": "0x0000000000000000000000000000000000000000000000000000000000000002"
}
```

Just for the sake of completeness, decoded the response is: `2`.

### eth_createAccessList {#eth-createaccesslist}

This method creates an [EIP2930](https://eips.ethereum.org/EIPS/eip-2930) type `accessList` based on a given `Transaction`. The `accessList` contains all storage slots and addresses read and written by the transaction, except for the sender account and the precompiles. This method uses the same `transaction` call [Transaction Call Object](/docs/interacting-with-geth/rpc/objects#transaction-call-object) and `blockNumberOrTag` object as `eth_call`. An `accessList` can be used to unstuck contracts that became inaccessible due to gas cost increases.

**Parameters:**

| Field              | Type     | Description                                    |
| :----------------- | :------- | :--------------------------------------------- |
| `transaction`      | `Object` | `TransactionCall` object                       |
| `blockNumberOrTag` | `Object` | Optional, blocknumber or `latest` or `pending` |

**Usage:**

```sh
curl --data '{"method":"eth_createAccessList","params":[{"from": "0x8cd02c6cbd8375b39b06577f8d50c51d86e8d5cd", "data": "0x608060806080608155"}, "pending"],"id":1,"jsonrpc":"2.0"}' -H "Content-Type: application/json" -X POST localhost:8545
```

**Response:**

The method `eth_createAccessList` returns list of addresses and storage keys used by the transaction, plus the gas consumed when the access list is added.

That is, it gives the list of addresses and storage keys that will be used by that transaction, plus the gas consumed if the access list is included. Like `eth_estimateGas`, this is an estimation; the list could change when the transaction is actually mined. Adding an `accessList` to a transaction does not necessary result in lower gas usage compared to a transaction without an access list.

**Example:**

```json
{
  "accessList": [
    {
      "address": "0xa02457e5dfd32bda5fc7e1f1b008aa5979568150",
      "storageKeys": [
        "0x0000000000000000000000000000000000000000000000000000000000000081",
      ]
    }
  ]
  "gasUsed": "0x125f8"
}
```

### eth_getHeaderByNumber

Returns a block header.

**Parameters:**

| Field         | Type       | Description  |
| :------------ | :--------- | :----------- |
| `blockNumber` | `Quantity` | Block number |

**Usage:**

```sh
curl localhost:8545 -X POST -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","method":"eth_getHeaderByNumber","params":["0x10823a8"],"id":0}'
```

**Response:**

```json
{
  "baseFeePerGas": "0x6c3f71624",
  "difficulty": "0x0",
  "extraData": "0x496c6c756d696e61746520446d6f63726174697a6520447374726962757465",
  "gasLimit": "0x1c9c380",
  "gasUsed": "0x1312759",
  "hash": "0x4574b6f248bf3295f76ae797454f4ec21c8ef5b53c0f7fee8534b65623d9360a",
  "logsBloom": "0x04a13010898372c9ca19007ccd04eed1f707098f04123de47da9d0b67ce1a60ab8ea324cd8291c36a8ca5a520893d1552711012dba82ad817332008d90ac788047c0fcd2d1200cb82bd1690b32b6d7ab8ab28a86b1f7095a19b59104d062882093746d041b510537a4d0015518c1583de073045981792d0030aa5cd5089a0a700160f74b0b250a9e30ea90596fdf851732815da30d800ace471e2768e09bc0d45e79f97238136523021a4bd52d45a5e184c8c810a9c22afa8670b6bab0eb2636ea1981120a400040829021a3e96cbe0262d8a6ba06006b37249117230968eecc0c16a7ae4090e888673f1101a27159d5cd12a190f5aa85cb524dbc72f5d4ed14",
  "miner": "0xdafea492d9c6733ae3d56b7ed1adb60692c98bc5",
  "mixHash": "0xec33ce424110ddd8f7e7db1cbc1261a63e44dacd158b4e801566cd6d5849295b",
  "nonce": "0x0000000000000000",
  "number": "0x10823a8",
  "parentHash": "0x956846b5012b1df4f4c928b85db2f6456b2faed2c0ca136e89c928a87ceec69c",
  "receiptsRoot": "0x89b73c221ca0d721f8805edbecbf55524b0556dc5111680bac1c4dd02a286457",
  "sha3Uncles": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
  "size": "0x25e",
  "stateRoot": "0xe38ef58ddfbf00b03f7bd431fca306e5fcaecc138f4208501d2588657a65a0f3",
  "timestamp": "0x646a982b",
  "totalDifficulty": "0xc70d815d562d3cfa955",
  "transactionsRoot": "0xe44699ea734cee851a852db4d257617c8369b8a7e68bd54b6de829377234017b",
  "withdrawalsRoot": "0x917f5a8e4d652233a80b0973ff20bde517ed2a6a93defe7e99c5263089453e17"
}
```

### eth_getHeaderByHash

Returns a block header.

**Parameters:**

| Field       | Type     | Description |
| :---------- | :------- | :---------- |
| `blockHash` | `string` | Block hash  |

**Usage:**

```sh
curl localhost:8545 -X POST -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","method":"eth_getHeaderByHash","params":["0x4574b6f248bf3295f76ae797454f4ec21c8ef5b53c0f7fee8534b65623d9360a"],"id":0}'
```

**Response:**

```json
{
  "baseFeePerGas": "0x6c3f71624",
  "difficulty": "0x0",
  "extraData": "0x496c6c756d696e61746520446d6f63726174697a6520447374726962757465",
  "gasLimit": "0x1c9c380",
  "gasUsed": "0x1312759",
  "hash": "0x4574b6f248bf3295f76ae797454f4ec21c8ef5b53c0f7fee8534b65623d9360a",
  "logsBloom": "0x04a13010898372c9ca19007ccd04eed1f707098f04123de47da9d0b67ce1a60ab8ea324cd8291c36a8ca5a520893d1552711012dba82ad817332008d90ac788047c0fcd2d1200cb82bd1690b32b6d7ab8ab28a86b1f7095a19b59104d062882093746d041b510537a4d0015518c1583de073045981792d0030aa5cd5089a0a700160f74b0b250a9e30ea90596fdf851732815da30d800ace471e2768e09bc0d45e79f97238136523021a4bd52d45a5e184c8c810a9c22afa8670b6bab0eb2636ea1981120a400040829021a3e96cbe0262d8a6ba06006b37249117230968eecc0c16a7ae4090e888673f1101a27159d5cd12a190f5aa85cb524dbc72f5d4ed14",
  "miner": "0xdafea492d9c6733ae3d56b7ed1adb60692c98bc5",
  "mixHash": "0xec33ce424110ddd8f7e7db1cbc1261a63e44dacd158b4e801566cd6d5849295b",
  "nonce": "0x0000000000000000",
  "number": "0x10823a8",
  "parentHash": "0x956846b5012b1df4f4c928b85db2f6456b2faed2c0ca136e89c928a87ceec69c",
  "receiptsRoot": "0x89b73c221ca0d721f8805edbecbf55524b0556dc5111680bac1c4dd02a286457",
  "sha3Uncles": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
  "size": "0x25e",
  "stateRoot": "0xe38ef58ddfbf00b03f7bd431fca306e5fcaecc138f4208501d2588657a65a0f3",
  "timestamp": "0x646a982b",
  "totalDifficulty": "0xc70d815d562d3cfa955",
  "transactionsRoot": "0xe44699ea734cee851a852db4d257617c8369b8a7e68bd54b6de829377234017b",
  "withdrawalsRoot": "0x917f5a8e4d652233a80b0973ff20bde517ed2a6a93defe7e99c5263089453e17"
}
```
