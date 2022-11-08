---
title: Basic traces
sort_key: B
---

The simplest type of transaction trace that Geth can generate are raw EVM opcode
traces. For every VM instruction the transaction executes, a structured log entry is
emitted, containing all contextual metadata deemed useful. This includes the *program
counter*, *opcode name*, *opcode cost*, *remaining gas*, *execution depth* and any
*occurred error*. The structured logs can optionally also contain the content of the
*execution stack*, *execution memory* and *contract storage*.

The entire output of a raw EVM opcode trace is a JSON object having a few metadata
fields: *consumed gas*, *failure status*, *return value*; and a list of *opcode entries*:

```json
{
  "gas":         25523,
  "failed":      false,
  "returnValue": "",
  "structLogs":  []
}
```

An example log for a single opcode entry has the following format:

```json
{
  "pc":      48,
  "op":      "DIV",
  "gasCost": 5,
  "gas":     64532,
  "depth":   1,
  "error":   null,
  "stack": [
    "00000000000000000000000000000000000000000000000000000000ffffffff",
    "0000000100000000000000000000000000000000000000000000000000000000",
    "2df07fbaabbe40e3244445af30759352e348ec8bebd4dd75467a9f29ec55d98d"
  ],
  "memory": [
    "0000000000000000000000000000000000000000000000000000000000000000",
    "0000000000000000000000000000000000000000000000000000000000000000",
    "0000000000000000000000000000000000000000000000000000000000000060"
  ],
  "storage": {
  }
}
```

### Generating basic traces

To generate a raw EVM opcode trace, Geth provides a few 
[RPC API endpoints](/docs/rpc/ns-debug). The most commonly used is 
[`debug_traceTransaction`](/docs/rpc/ns-debug#debug_tracetransaction).

In its simplest form, `traceTransaction` accepts a transaction hash as its 
only argument. It then traces the transaction, aggregates all the generated 
data and returns it as a **large** JSON object. A sample invocation from the 
Geth console would be:

```js
debug.traceTransaction("0xfc9359e49278b7ba99f59edac0e3de49956e46e530a53c15aa71226b7aa92c6f")
```

The same call can also be invoked from outside the node too via HTTP 
RPC (e.g. using Curl). In this case, the HTTP endpoint must be enabled in 
Geth using the `--http` command and the `debug` API namespace must be exposed 
using `--http.api=debug`.

```
$ curl -H "Content-Type: application/json" -d '{"id": 1, "method": "debug_traceTransaction", "params": ["0xfc9359e49278b7ba99f59edac0e3de49956e46e530a53c15aa71226b7aa92c6f"]}' localhost:8545
```

To follow along with this tutorial, transaction hashes can be found from 
a local Geth node (e.g. by attaching a [Javascript console](/docs/interface/javascript-console) 
and running `eth.getBlock('latest')` then passing a transaction hash from the 
returned block to `debug.traceTransaction()`) or from a block explorer (for 
[Mainnet](https://etherscan.io/) or a [testnet](https://goerli.etherscan.io/)).

It is also possible to configure the trace by passing Boolean (true/false) values 
for four parameters that tweak the verbosity of the trace. By default, the 
*EVM memory* and *Return data* are not reported but the *EVM stack* and 
*EVM storage* are. To report the maximum amount of data:

```shell
enableMemory: true
disableStack: false
disableStorage: false
enableReturnData: true
```

An example call, made in the Geth Javascript console, configured to report 
the maximum amount of data looks as follows:

```js
debug.traceTransaction("0xfc9359e49278b7ba99f59edac0e3de49956e46e530a53c15aa71226b7aa92c6f",{enableMemory: true, disableStack: false, disableStorage: false, enableReturnData: true})
```

The above operation was run on the (now-deprecated) Rinkeby network (with a node retaining 
enough history), resulting in this [trace dump](https://gist.github.com/karalabe/c91f95ac57f5e57f8b950ec65ecc697f).

Alternatively, disabling *EVM Stack*, *EVM Memory*, *Storage* and 
*Return data* (as demonstrated in the Curl request below) results in the 
following, much shorter, [trace dump](https://gist.github.com/karalabe/d74a7cb33a70f2af75e7824fc772c5b4).

```
$ curl -H "Content-Type: application/json" -d '{"id": 1, "method": "debug_traceTransaction", "params": ["0xfc9359e49278b7ba99f59edac0e3de49956e46e530a53c15aa71226b7aa92c6f", {"disableStack": true, "disableStorage": true}]}' localhost:8545
```

### Limits of basic traces

Although the raw opcode traces generated above are useful, having an 
individual log entry for every single opcode is too low level for most use cases, 
and will require developers to create additional tools to post-process the traces. 
Additionally, a full opcode trace can easily go into the hundreds of megabytes, 
making them very resource intensive to get out of the node and process externally.

To avoid those issues, Geth supports running custom JavaScript tracers *within* 
the Ethereum node, which have full access to the EVM stack, memory and contract 
storage. This means developers only have to gather the data they actually need, 
and do any processing at the source.


## Summary

This page described how to do basic traces in Geth. Basic traces are very low 
level and can generate lots of data that might not all be useful. Therefore, 
it is also possible to use a set of built-in tracers or write custom ones in 
Javascript or Go.

Read more about [built-in](/docs/evm-tracing/builtin-tracers) and 
[custom](/docs/evm-tracing/custom-tracer) traces.