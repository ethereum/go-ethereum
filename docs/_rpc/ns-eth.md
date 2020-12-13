---
title: eth Namespace
sort_key: C
---

Geth provides several extensions to the standard "eth" JSON-RPC namespace.

* TOC
{:toc}

### eth_subscribe, eth_unsubscribe

These methods are used for real-time events through subscriptions. See the [subscription
documentation](./pubsub) for more information.

### eth_call

Executes a new message call immediately, without creating a transaction on the block
chain. The `eth_call` method can be used to query internal contract state, to execute
validations coded into a contract or even to test what the effect of a transaction would
be without running it live.

#### Parameters

The method takes 3 parameters: an unsigned transaction object to execute in read-only
mode; the block number to execute the call against; and an optional state override-set to
allow executing the call against a modified chain state.

##### 1. `Object` - Transaction call object

The *transaction call object* is mandatory and contains all the necessary parameters to
execute a read-only EVM contract method.

| Field      | Type       | Bytes | Optional | Description |
|:-----------|:-----------|:------|:---------|:------------|
| `from`     | `Address`  | 20    | Yes      | Address the transaction is simulated to have been sent from. Defaults to first account in the local keystore or the `0x00..0` address if no local accounts are available. |
| `to`       | `Address`  | 20    | No       | Address the transaction is sent to. |
| `gas`      | `Quantity` | <8    | Yes      | Maximum gas allowance for the code execution to avoid infinite loops. Defaults to `2^63` or whatever value the node operator specified via `--rpc.gascap`. |
| `gasPrice` | `Quantity` | <32   | Yes      | Number of `wei` to simulate paying for each unit of gas during execution. Defaults to `1 gwei`. |
| `value`    | `Quantity` | <32   | Yes      | Amount of `wei` to simulate sending along with the transaction. Defaults to `0`. |
| `data`     | `Binary`   | any   | Yes      | Binary data to send to the target contract. Generally the 4 byte hash of the method signature followed by the ABI encoded parameters. For details please see the [Ethereum Contract ABI](https://github.com/ethereum/wiki/wiki/Ethereum-Contract-ABI). |

Example:

```json
{
  "from": "0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3",
  "to":   "0xebe8efa441b9302a0d7eaecc277c09d20d684540",
  "gas":  "0x1bd7c",
  "data": "0xd459fc46000000000000000000000000000000000000000000000000000000000046c650dbb5e8cb2bac4d2ed0b1e6475d37361157738801c494ca482f96527eb48f9eec488c2eba92d31baeccfb6968fad5c21a3df93181b43b4cf253b4d572b64172ef000000000000000000000000000000000000000000000000000000000000008c00000000000000000000000000000000000000000000000000000000000000e0000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000001a00000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000001c000000000000000000000000000000000000000000000000000000000000001c0000000000000000000000000000000000000000000000000000000000000002b85c0c828d7a98633b4e1b65eac0c017502da909420aeade9a280675013df36bdc71cffdf420cef3d24ba4b3f9b980bfbb26bd5e2dcf7795b3519a3fd22ffbb2000000000000000000000000000000000000000000000000000000000000000238fb6606dc2b5e42d00c653372c153da8560de77bd9afaba94b4ab6e4aa11d565d858c761320dbf23a94018d843772349bd9d92301b0ca9ca983a22d86a70628",
}
```

##### 2. `Quantity | Tag` - Block number or the string `latest` or `pending`

The *block number* is mandatory and defines the context (state) against which the
specified transaction should be executed. It is not possible to execute calls against
reorged blocks; or blocks older than 128 (unless the node is an archive node).

##### 3. `Object` - State override set

The *state override set* is an optional address-to-state mapping, where each entry
specifies some state to be ephemerally overridden prior to executing the call. Each
address maps to an object containing:

| Field       | Type       | Bytes | Optional | Description |
|:------------|:-----------|:------|:---------|:------------|
| `balance`   | `Quantity` | <32   | Yes      | Fake balance to set for the account before executing the call. |
| `nonce`     | `Quantity` | <8    | Yes      | Fake nonce to set for the account before executing the call. |
| `code`      | `Binary`   | any   | Yes      | Fake EVM bytecode to inject into the account before executing the call. |
| `state`     | `Object`   | any   | Yes      | Fake key-value mapping to override **all** slots in the account storage before executing the call. |
| `stateDiff` | `Object`   | any   | Yes      | Fake key-value mapping to override **individual** slots in the account storage before executing the call. |

The goal of the *state override set* is manyfold:

 * It can be used by DApps to reduce the amount of contract code needed to be deployed on
   chain. Code that simply returns internal state or does pre-defined validations can be
   kept off chain and fed to the node on-demand.
 * It can be used for smart contract analysis by extending the code deployed on chain with
   custom methods and invoking them. This avoids having to download and reconstruct the
   entire state in a sandbox to run custom code against.
 * It can be used to debug smart contracts in an already deployed large suite of contracts
   by selectively overriding some code or state and seeing how execution changes.
   Specialized tooling will probably be necessary.

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

The method returns a single `Binary` consisting the return value of the executed contract
call.

#### Simple example

With a synced Rinkeby node with RPC exposed on localhost (`geth --rinkeby --rpc`) we can
make a call against the [Checkpoint
Oracle](https://rinkeby.etherscan.io/address/0xebe8efa441b9302a0d7eaecc277c09d20d684540)
to retrieve the list of administrators:

```
$ curl --data '{"method":"eth_call","params":[{"to":"0xebe8efa441b9302a0d7eaecc277c09d20d684540","data":"0x45848dfc"},"latest"],"id":1,"jsonrpc":"2.0"}' -H "Content-Type: application/json" -X POST localhost:8545
```

And the result is an Ethereum ABI encoded list of accounts:

```json
{
  "id":      1,
  "jsonrpc": "2.0",
  "result":  "0x00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000004000000000000000000000000d9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f300000000000000000000000078d1ad571a1a09d60d9bbf25894b44e4c8859595000000000000000000000000286834935f4a8cfb4ff4c77d5770c2775ae2b0e7000000000000000000000000b86e2b0ab5a4b1373e40c51a7c712c70ba2f9f8e"
}
```

Just for the sake of completeness, decoded the response is:

```
0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3,
0x78d1ad571a1a09d60d9bbf25894b44e4c8859595,
0x286834935f4a8cfb4ff4c77d5770c2775ae2b0e7,
0xb86e2b0ab5a4b1373e40c51a7c712c70ba2f9f8e
```

#### Override example

The above *simple example* showed how to call a method already exposed by an on-chain
smart contract. What if we want to access some data not exposed by it?

We can gut out the
[original](https://github.com/ethereum/go-ethereum/blob/master/contracts/checkpointoracle/contract/oracle.sol)
checkpoint oracle contract with one that retains the same fields (to retain the same
storage layout), but one that includes a different method set:

```
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

With a synced Rinkeby node with RPC exposed on localhost (`geth --rinkeby --rpc`) we can
make a call against the live [Checkpoint
Oracle](https://rinkeby.etherscan.io/address/0xebe8efa441b9302a0d7eaecc277c09d20d684540),
but override its byte code with our own version that has an accessor for the voting
threshold field:

```
$ curl --data '{"method":"eth_call","params":[{"to":"0xebe8efa441b9302a0d7eaecc277c09d20d684540","data":"0x0be5b6ba"}, "latest", {"0xebe8efa441b9302a0d7eaecc277c09d20d684540": {"code":"0x6080604052348015600f57600080fd5b506004361060285760003560e01c80630be5b6ba14602d575b600080fd5b60336045565b60408051918252519081900360200190f35b6007549056fea265627a7a723058206f26bd0433456354d8d1228d8fe524678a8aeeb0594851395bdbd35efc2a65f164736f6c634300050a0032"}}],"id":1,"jsonrpc":"2.0"}' -H "Content-Type: application/json" -X POST localhost:8545
```

And the result is the Ethereum ABI encoded threshold number:

```json
{
  "id":      1,
  "jsonrpc": "2.0",
  "result":  "0x0000000000000000000000000000000000000000000000000000000000000002"
}
```

Just for the sake of completeness, decoded the response is: `2`.
