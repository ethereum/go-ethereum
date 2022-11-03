---
title: Built-in tracers
description: Explanation of the tracers that come bundled in Geth as part of the tracing API.
---

Geth comes bundled with a choice of tracers that can be invoked via the [tracing API](/docs/rpc/ns-debug). Some of these built-in tracers are implemented natively in Go, and others in Javascript. The default tracer is the opcode logger (otherwise known as struct logger) which is the default tracer for all the methods. Other tracers have to be specified by name when sending a request. 

## Struct/opcode logger

The struct logger (aka opcode logger) is a native Go tracer which executes a transaction and emits the opcode and execution context at every step. This is the tracer that will be used when no name is passed to the API, e.g. `debug.traceTransaction(<txhash>)`. The following information is emitted at each step:

| field      | type          | description                                                                                                                      |
| ---------- | ------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| pc         | uint64        | program counter                                                                                                                  |
| op         | byte          | opcode to be executed                                                                                                            |
| gas        | uint64        | remaining gas                                                                                                                    |
| gasCost    | uint64        | cost for executing op                                                                                                            |
| memory     | []byte        | EVM memory. Enabled via `enableMemory`                                                                                           |
| memSize    | int           | Size of memory                                                                                                                   |
| stack      | []uint256     | EVM stack. Disabled via `disableStack`                                                                                           |
| returnData | []byte        | Last call's return data. Enabled via `enableReturnData`                                                                          |
| storage    | map[hash]hash | Storage slots of current contract read from and written to. Only emitted for `SLOAD` and `SSTORE`. Disabled via `disableStorage` |
| depth      | int           | Current call depth                                                                                                               |
| refund     | uint64        | Refund counter                                                                                                                   |
| error      | string        | Error message if any                                                                                                             |

Note that the fields `memory`, `stack`, `returnData`, and `storage` have dynamic size and depending on the exact transaction they could grow large in size. This is specially true for `memory` which could blow up the trace size. It is recommended to keep them disabled unless they are explicitly required for a given use-case.

## Native tracers

The following tracers are implement in Go and as such have offer good performance. They are selected by their name when invoking a tracing API method, e.g. `debug.traceTransaction(<txhash>, { tracer: 'callTracer' })`.

### 4byteTracer

Solidity contract functions are [addressed](https://docs.soliditylang.org/en/develop/abi-spec.html#function-selector) by the first four four byte of the Keccak-256 hash of their signature. Therefore when calling the function of a contract, the caller must send this function selector as well as the ABI-encoded arguments as call data.

The `4byteTracer` collects the function selectors of every function executed in the lifetime of a transaction, along with the size of the supplied call data. The result is a `map[string]int` where the keys are `SELECTOR-CALLDATASIZE` and the values are number of occurances of this key. E.g.:

```terminal
> debug.traceTransaction( "0x214e597e35da083692f5386141e69f47e973b2c56e7a8073b1ea08fd7571e9de", {tracer: "4byteTracer"})
{
  "0x27dc297e-128": 1,
  "0x38cc4831-0": 2,
  "0x524f3889-96": 1,
  "0xadf59f99-288": 1,
  "0xc281d19e-0": 1
}
```

### callTracer

The `callTracer` tracks all the call frames executed during a transaction, including depth 0. The result will be a nested list of call frames, resembling how EVM works. They form a tree with the top-level call at root and sub-calls as children of the higher levels. Each call frame has the following fields:

| field   | type        | description                          |
| ------- | ----------- | ------------------------------------ |
| type    | string      | CALL or CREATE                       |
| from    | string      | address                              |
| to      | string      | address                              |
| value   | string      | hex-encoded amount of value transfer |
| gas     | string      | hex-encoded gas provided for call    |
| gasUsed | string      | hex-encoded gas used during call     |
| input   | string      | call data                            |
| output  | string      | return data                          |
| error   | string      | error, if any                        |
| calls   | []callframe | list of sub-calls                    |

Things to note about the call tracer:

- Calls to precompiles are also included in the result
- In case a frame reverts, the field `output` will contain the raw return data, unlike [revertReasonTracer](#revertreasontracer) which parses the data and returns the revert message

### noopTracer

This tracer is noop. It returns an empty object and is only meant for testing the setup.

### prestateTracer

Executing a transaction requires the prior state, including account of sender and recipient, contracts that are called during execution, etc. The `prestateTracer` replays the tx and tracks every part of state that is touched. This is similar to the concept of a [stateless witness](https://ethresear.ch/t/the-stateless-client-concept/172), the difference being this tracer doesn't return any cryptographic proof, rather only the trie leaves. The result is an object. The keys are addresses of accounts. The value is an object with the following fields:

| field   | type              | description                   |
| ------- | ----------------- | ----------------------------- |
| balance | string            | balance in Wei                |
| nonce   | uint64            | nonce                         |
| code    | string            | hex-encoded bytecode          |
| storage | map[string]string | storage slots of the contract |

### revertReasonTracer

The `revertReasonTracer` is useful for analyzing failed transactions. The return value is:

- In case the transaction reverted: reason of the revert as returned by the Solidity contract
- Error message for any other failure

Example:

```terminal
> debug.traceTransaction('0x97695ffb034be7e1faeb372a564bb951ba4ebf4fee4caff2f9d1702497bb2b8b', { tracer: 'revertReasonTracer' })
"execution reverted: tokensMintedPerAddress exceed MAX_TOKENS_MINTED_PER_ADDRESS"
```

## JS tracers

The following are a list of tracers written in JS that come as part of Geth:

- `bigramTracer`: Counts the opcode bigrams, i.e. how many times 2 opcodes were executed one after the other
- `evmdisTracer`: Returns sufficient information from a trace to perform [evmdis](https://github.com/Arachnid/evmdis)-style disassembly
- `opcountTracer` Counts the total number of opcodes executed
- `trigramTracer`: Counts the opcode trigrams
- `unigramTracer`: Counts the occurances of each opcode






#############################

To follow along with this tutorial, transaction hashes can be found from a local Geth node (e.g. by attaching a [Javascript console](/docs/interface/javascript-console) and running `eth.getBlock('latest')` then passing a transaction hash from the returned block to `debug.traceTransaction()`) or from a block explorer (for [Mainnet](https://etherscan.io/) or a [testnet](https://goerli.etherscan.io/)).

It is also possible to configure the trace by passing Boolean (true/false) values for four parameters that tweak the verbosity of the trace. By default, the _EVM memory_ and _Return data_ are not reported but the _EVM stack_ and _EVM storage_ are. To report the maximum amount of data:

```shell
enableMemory: true
disableStack: false
disableStorage: false
enableReturnData: true
```

An example call, made in the Geth Javascript console, configured to report the maximum amount of data looks as follows:

```js
debug.traceTransaction('0xfc9359e49278b7ba99f59edac0e3de49956e46e530a53c15aa71226b7aa92c6f', {
  enableMemory: true,
  disableStack: false,
  disableStorage: false,
  enableReturnData: true
});
```

Running the above operation on the Rinkeby network (with a node retaining enough history) will result in this [trace dump](https://gist.github.com/karalabe/c91f95ac57f5e57f8b950ec65ecc697f).

Alternatively, disabling _EVM Stack_, _EVM Memory_, _Storage_ and _Return data_ (as demonstrated in the Curl request below) results in the following, much shorter, [trace dump](https://gist.github.com/karalabe/d74a7cb33a70f2af75e7824fc772c5b4).

```
$ curl -H "Content-Type: application/json" -d '{"id": 1, "method": "debug_traceTransaction", "params": ["0xfc9359e49278b7ba99f59edac0e3de49956e46e530a53c15aa71226b7aa92c6f", {"disableStack": true, "disableStorage": true}]}' localhost:8545
```



######################################################


