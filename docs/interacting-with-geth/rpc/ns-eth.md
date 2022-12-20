---
title: eth Namespace
description: Documentation for the JSON-RPC API "eth" namespace
---

Documentation for the API methods in the `eth` namespace can be found on [ethereum.org](https://ethereum.org/en/developers/docs/apis/json-rpc/#eth_protocolversion). Geth provides several extensions to the standard "eth" JSON-RPC namespace that are defined below.

### eth_subscribe, eth_unsubscribe {#eth-subscribe-unsubscribe}

These methods are used for real-time events through subscriptions. See the [subscription documentation](/docs/interacting-with-geth/rpc/pubsub) for more information.

### eth_call {#eth-call}

Executes a new message call immediately, without creating a transaction on the block chain. The `eth_call` method can be used to query internal contract state, to execute validations coded into a contract or even to test what the effect of a transaction would be without running it live.

#### Parameters

The method takes 3 parameters: an unsigned transaction object to execute in read-only mode; the block number to execute the call against; and an optional state override-set to allow executing the call against a modified chain state.

##### 1. `Object` - Transaction call object

The _transaction call object_ is mandatory. Please see [here](/docs/interacting-with-geth/rpc/objects) for details.

##### 2. `Quantity | Tag` - Block number or the string `latest` or `pending`

The _block number_ is mandatory and defines the context (state) against which the specified transaction should be executed. It is not possible to execute calls against reorged blocks; or blocks older than 128 (unless the node is an archive node).

##### 3. `Object` - State override set

The _state override set_ is an optional address-to-state mapping, where each entry specifies some state to be ephemerally overridden prior to executing the call. Each address maps to an object containing:

| Field       | Type       | Bytes | Optional | Description                                                                                               |
| :---------- | :--------- | :---- | :------- | :-------------------------------------------------------------------------------------------------------- |
| `balance`   | `Quantity` | <32   | Yes      | Fake balance to set for the account before executing the call.                                            |
| `nonce`     | `Quantity` | <8    | Yes      | Fake nonce to set for the account before executing the call.                                              |
| `code`      | `Binary`   | any   | Yes      | Fake EVM bytecode to inject into the account before executing the call.                                   |
| `state`     | `Object`   | any   | Yes      | Fake key-value mapping to override **all** slots in the account storage before executing the call.        |
| `stateDiff` | `Object`   | any   | Yes      | Fake key-value mapping to override **individual** slots in the account storage before executing the call. |

The goal of the _state override set_ is manyfold:

- It can be used by DApps to reduce the amount of contract code needed to be deployed on chain. Code that simply returns internal state or does pre-defined validations can be kept off chain and fed to the node on-demand.
- It can be used for smart contract analysis by extending the code deployed on chain with custom methods and invoking them. This avoids having to download and reconstruct the entire state in a sandbox to run custom code against.

- It can be used to debug smart contracts in an already deployed large suite of contracts by selectively overriding some code or state and seeing how execution changes. Specialized tooling will probably be necessary.

Example:

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

#### Return Values

The method returns a single `Binary` consisting the return value of the executed contract call.

#### Simple example

**note that this example uses the Rinkeby network, which is now deprecated**

With a synced Rinkeby node with RPC exposed on localhost (`geth --rinkeby --http`) we can make a call against the [CheckpointOracle](https://rinkeby.etherscan.io/address/0xebe8efa441b9302a0d7eaecc277c09d20d684540) to retrieve the list of administrators:

```sh
$ curl --data '{"method":"eth_call","params":[{"to":"0xebe8efa441b9302a0d7eaecc277c09d20d684540","data":"0x45848dfc"},"latest"],"id":1,"jsonrpc":"2.0"}' -H "Content-Type: application/json" -X POST localhost:8545
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

#### Override example

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
$ curl --data '{"method":"eth_call","params":[{"to":"0xebe8efa441b9302a0d7eaecc277c09d20d684540","data":"0x0be5b6ba"}, "latest", {"0xebe8efa441b9302a0d7eaecc277c09d20d684540": {"code":"0x6080604052348015600f57600080fd5b506004361060285760003560e01c80630be5b6ba14602d575b600080fd5b60336045565b60408051918252519081900360200190f35b6007549056fea265627a7a723058206f26bd0433456354d8d1228d8fe524678a8aeeb0594851395bdbd35efc2a65f164736f6c634300050a0032"}}],"id":1,"jsonrpc":"2.0"}' -H "Content-Type: application/json" -X POST localhost:8545
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

This method creates an [EIP2930](https://eips.ethereum.org/EIPS/eip-2930) type `accessList` based on a given `Transaction`. The `accessList` contains all storage slots and addresses read and written by the transaction, except for the sender account and the precompiles. This method uses the same `transaction` call [object](/docs/interacting-with-geth/rpc/objects#transaction-call-object) and `blockNumberOrTag` object as `eth_call`. An `accessList` can be used to unstuck contracts that became inaccessible due to gas cost increases.

#### Parameters

| Field              | Type     | Description                                    |
| :----------------- | :------- | :--------------------------------------------- |
| `transaction`      | `Object` | `TransactionCall` object                       |
| `blockNumberOrTag` | `Object` | Optional, blocknumber or `latest` or `pending` |

#### Usage

```
curl --data '{"method":"eth_createAccessList","params":[{"from": "0x8cd02c6cbd8375b39b06577f8d50c51d86e8d5cd", "data": "0x608060806080608155"}, "pending"],"id":1,"jsonrpc":"2.0"}' -H "Content-Type: application/json" -X POST localhost:8545
```

#### Response

The method `eth_createAccessList` returns list of addresses and storage keys used by the transaction, plus the gas consumed when the access list is added.

That is, it gives the list of addresses and storage keys that will be used by that transaction, plus the gas consumed if the access list is included. Like `eth_estimateGas`, this is an estimation; the list could change when the transaction is actually mined. Adding an `accessList` to a transaction does not necessary result in lower gas usage compared to a transaction without an access list.

Example:

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
