---
title: debug Namespace
sort_key: C
---

The `debug` API gives you access to several non-standard RPC methods, which will allow you
to inspect, debug and set certain debugging flags during runtime.

* TOC
{:toc}

### debug_backtraceAt

Sets the logging backtrace location. When a backtrace location
is set and a log message is emitted at that location, the stack
of the goroutine executing the log statement will be printed to stderr.

The location is specified as `<filename>:<line>`.

| Client  | Method invocation                                     |
|:--------|-------------------------------------------------------|
| Console | `debug.backtraceAt(string)`                           |
| RPC     | `{"method": "debug_backtraceAt", "params": [string]}` |

Example:

``` javascript
> debug.backtraceAt("server.go:443")
```

### debug_blockProfile

Turns on block profiling for the given duration and writes
profile data to disk. It uses a profile rate of 1 for most
accurate information. If a different rate is desired, set
the rate and write the profile manually using
`debug_writeBlockProfile`.

| Client  | Method invocation                                              |
|:--------|----------------------------------------------------------------|
| Console | `debug.blockProfile(file, seconds)`                            |
| RPC     | `{"method": "debug_blockProfile", "params": [string, number]}` |

### debug_cpuProfile

Turns on CPU profiling for the given duration and writes
profile data to disk.

| Client  | Method invocation                                            |
|:--------|--------------------------------------------------------------|
| Console | `debug.cpuProfile(file, seconds)`                            |
| RPC     | `{"method": "debug_cpuProfile", "params": [string, number]}` |

### debug_dumpBlock

Retrieves the state that corresponds to the block number and returns a list of accounts (including
storage and code).

| Client  | Method invocation                                     |
|:--------|-------------------------------------------------------|
| Go      | `debug.DumpBlock(number uint64) (state.World, error)` |
| Console | `debug.traceBlockByHash(number, [options])`           |
| RPC     | `{"method": "debug_dumpBlock", "params": [number]}`   |

#### Example

```javascript
> debug.dumpBlock(10)
{
    fff7ac99c8e4feb60c9750054bdc14ce1857f181: {
      balance: "49358640978154672",
      code: "",
      codeHash: "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
      nonce: 2,
      root: "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
      storage: {}
    },
    fffbca3a38c3c5fcb3adbb8e63c04c3e629aafce: {
      balance: "3460945928",
      code: "",
      codeHash: "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
      nonce: 657,
      root: "56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
      storage: {}
    }
  },
  root: "19f4ed94e188dd9c7eb04226bd240fa6b449401a6c656d6d2816a87ccaf206f1"
}
```

### debug_gcStats

Returns GC statistics.

See https://golang.org/pkg/runtime/debug/#GCStats for information about
the fields of the returned object.

| Client  | Method invocation                                 |
|:--------|---------------------------------------------------|
| Console | `debug.gcStats()`                                 |
| RPC     | `{"method": "debug_gcStats", "params": []}`       |

### debug_getBlockRlp

Retrieves and returns the RLP encoded block by number.

| Client  | Method invocation                                     |
|:--------|-------------------------------------------------------|
| Go      | `debug.GetBlockRlp(number uint64) (string, error)`    |
| Console | `debug.getBlockRlp(number, [options])`                |
| RPC     | `{"method": "debug_getBlockRlp", "params": [number]}` |

References: [RLP](https://github.com/ethereum/wiki/wiki/RLP)

### debug_goTrace

Turns on Go runtime tracing for the given duration and writes
trace data to disk.

| Client  | Method invocation                                         |
|:--------|-----------------------------------------------------------|
| Console | `debug.goTrace(file, seconds)`                            |
| RPC     | `{"method": "debug_goTrace", "params": [string, number]}` |

### debug_memStats

Returns detailed runtime memory statistics.

See https://golang.org/pkg/runtime/#MemStats for information about
the fields of the returned object.

| Client  | Method invocation                                 |
|:--------|---------------------------------------------------|
| Console | `debug.memStats()`                                |
| RPC     | `{"method": "debug_memStats", "params": []}`      |

### debug_seedHash

Fetches and retrieves the seed hash of the block by number

| Client  | Method invocation                                  |
|:--------|----------------------------------------------------|
| Go      | `debug.SeedHash(number uint64) (string, error)`    |
| Console | `debug.seedHash(number, [options])`                |
| RPC     | `{"method": "debug_seedHash", "params": [number]}` |

### debug_setHead

Sets the current head of the local chain by block number. **Note**, this is a
destructive action and may severely damage your chain. Use with *extreme* caution.

| Client  | Method invocation                                 |
|:--------|---------------------------------------------------|
| Go      | `debug.SetHead(number uint64)`                    |
| Console | `debug.setHead(number)`                           |
| RPC     | `{"method": "debug_setHead", "params": [number]}` |

References:
[Ethash](https://eth.wiki/en/concepts/ethash/ethash)

### debug_setBlockProfileRate

Sets the rate (in samples/sec) of goroutine block profile
data collection. A non-zero rate enables block profiling,
setting it to zero stops the profile. Collected profile data
can be written using `debug_writeBlockProfile`.

| Client  | Method invocation                                             |
|:--------|---------------------------------------------------------------|
| Console | `debug.setBlockProfileRate(rate)`                             |
| RPC     | `{"method": "debug_setBlockProfileRate", "params": [number]}` |

### debug_stacks

Returns a printed representation of the stacks of all goroutines.
Note that the web3 wrapper for this method takes care of the printing
and does not return the string.

| Client  | Method invocation                                 |
|:--------|---------------------------------------------------|
| Console | `debug.stacks()`                                  |
| RPC     | `{"method": "debug_stacks", "params": []}`        |

### debug_startCPUProfile

Turns on CPU profiling indefinitely, writing to the given file.

| Client  | Method invocation                                         |
|:--------|-----------------------------------------------------------|
| Console | `debug.startCPUProfile(file)`                             |
| RPC     | `{"method": "debug_startCPUProfile", "params": [string]}` |

### debug_startGoTrace

Starts writing a Go runtime trace to the given file.

| Client  | Method invocation                                      |
|:--------|--------------------------------------------------------|
| Console | `debug.startGoTrace(file)`                             |
| RPC     | `{"method": "debug_startGoTrace", "params": [string]}` |

### debug_stopCPUProfile

Stops an ongoing CPU profile.

| Client  | Method invocation                                  |
|:--------|----------------------------------------------------|
| Console | `debug.stopCPUProfile()`                           |
| RPC     | `{"method": "debug_stopCPUProfile", "params": []}` |

### debug_stopGoTrace

Stops writing the Go runtime trace.

| Client  | Method invocation                                 |
|:--------|---------------------------------------------------|
| Console | `debug.startGoTrace(file)`                        |
| RPC     | `{"method": "debug_stopGoTrace", "params": []}`   |

### debug_traceBlock

The `traceBlock` method will return a full stack trace of all invoked opcodes of all transaction
that were included in this block. **Note**, the parent of this block must be present or it will
fail.

| Client  | Method invocation                                                        |
|:--------|--------------------------------------------------------------------------|
| Go      | `debug.TraceBlock(blockRlp []byte, config. *vm.Config) BlockTraceResult` |
| Console | `debug.traceBlock(tblockRlp, [options])`                                 |
| RPC     | `{"method": "debug_traceBlock", "params": [blockRlp, {}]}`               |

References:
[RLP](https://github.com/ethereum/wiki/wiki/RLP)

#### Example

```javascript
> debug.traceBlock("0xblock_rlp")
{
  gas: 85301,
  returnValue: "",
  structLogs: [{
      depth: 1,
      error: "",
      gas: 162106,
      gasCost: 3,
      memory: null,
      op: "PUSH1",
      pc: 0,
      stack: [],
      storage: {}
  },
    /* snip */
  {
      depth: 1,
      error: "",
      gas: 100000,
      gasCost: 0,
      memory: ["0000000000000000000000000000000000000000000000000000000000000006", "0000000000000000000000000000000000000000000000000000000000000000", "0000000000000000000000000000000000000000000000000000000000000060"],
      op: "STOP",
      pc: 120,
      stack: ["00000000000000000000000000000000000000000000000000000000d67cbec9"],
      storage: {
        0000000000000000000000000000000000000000000000000000000000000004: "8241fa522772837f0d05511f20caa6da1d5a3209000000000000000400000001",
        0000000000000000000000000000000000000000000000000000000000000006: "0000000000000000000000000000000000000000000000000000000000000001",
        f652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f: "00000000000000000000000002e816afc1b5c0f39852131959d946eb3b07b5ad"
      }
  }]
```

### debug_traceBlockByNumber

Similar to [debug_traceBlock](#debug_traceblock), `traceBlockByNumber` accepts a block number and will replay the
block that is already present in the database.

| Client  | Method invocation                                                              |
|:--------|--------------------------------------------------------------------------------|
| Go      | `debug.TraceBlockByNumber(number uint64, config. *vm.Config) BlockTraceResult` |
| Console | `debug.traceBlockByNumber(number, [options])`                                  |
| RPC     | `{"method": "debug_traceBlockByNumber", "params": [number, {}]}`               |

References:
[RLP](https://github.com/ethereum/wiki/wiki/RLP)

### debug_traceBlockByHash

Similar to [debug_traceBlock](#debug_traceblock), `traceBlockByHash` accepts a block hash and will replay the
block that is already present in the database.

| Client  | Method invocation                                                               |
|:--------|---------------------------------------------------------------------------------|
| Go      | `debug.TraceBlockByHash(hash common.Hash, config. *vm.Config) BlockTraceResult` |
| Console | `debug.traceBlockByHash(hash, [options])`                                       |
| RPC     | `{"method": "debug_traceBlockByHash", "params": [hash {}]}`                     |

References:
[RLP](https://github.com/ethereum/wiki/wiki/RLP)

### debug_traceBlockFromFile

Similar to [debug_traceBlock](#debug_traceblock), `traceBlockFromFile` accepts a file containing the RLP of the block.

| Client  | Method invocation                                                                |
|:--------|----------------------------------------------------------------------------------|
| Go      | `debug.TraceBlockFromFile(fileName string, config. *vm.Config) BlockTraceResult` |
| Console | `debug.traceBlockFromFile(fileName, [options])`                                  |
| RPC     | `{"method": "debug_traceBlockFromFile", "params": [fileName, {}]}`               |

References:
[RLP](https://github.com/ethereum/wiki/wiki/RLP)

### debug_standardTraceBlockToFile


When JS-based tracing (see below) was first implemented, the intended usecase was to enable long-running tracers that could stream results back via a subscription channel.
This method works a bit differently. (For full details, see [PR](https://github.com/ethereum/go-ethereum/pull/17914))

- It streams output to disk during the execution, to not blow up the memory usage on the node
- It uses `jsonl` as output format (to allow streaming)
- Uses a cross-client standardized output, so called 'standard json'
  * Uses `op` for string-representation of opcode, instead of `op`/`opName` for numeric/string, and other simlar small differences.
  * has `refund`
  * Represents memory as a contiguous chunk of data, as opposed to a list of `32`-byte segments like `debug_traceTransaction`

This means that this method is only 'useful' for callers who control the node -- at least sufficiently to be able to read the artefacts from the filesystem after the fact.

The method can be used to dump a certain transaction out of a given block:
```
> debug.standardTraceBlockToFile("0x0bbe9f1484668a2bf159c63f0cf556ed8c8282f99e3ffdb03ad2175a863bca63", {txHash:"0x4049f61ffbb0747bb88dc1c85dd6686ebf225a3c10c282c45a8e0c644739f7e9", disableMemory:true})
["/tmp/block_0x0bbe9f14-14-0x4049f61f-099048234"]
```
Or all txs from a block:
```
> debug.standardTraceBlockToFile("0x0bbe9f1484668a2bf159c63f0cf556ed8c8282f99e3ffdb03ad2175a863bca63", {disableMemory:true})
["/tmp/block_0x0bbe9f14-0-0xb4502ea7-409046657", "/tmp/block_0x0bbe9f14-1-0xe839be8f-954614764", "/tmp/block_0x0bbe9f14-2-0xc6e2052f-542255195", "/tmp/block_0x0bbe9f14-3-0x01b7f3fe-209673214", "/tmp/block_0x0bbe9f14-4-0x0f290422-320999749", "/tmp/block_0x0bbe9f14-5-0x2dc0fb80-844117472", "/tmp/block_0x0bbe9f14-6-0x35542da1-256306111", "/tmp/block_0x0bbe9f14-7-0x3e199a08-086370834", "/tmp/block_0x0bbe9f14-8-0x87778b88-194603593", "/tmp/block_0x0bbe9f14-9-0xbcb081ba-629580052", "/tmp/block_0x0bbe9f14-10-0xc254381a-578605923", "/tmp/block_0x0bbe9f14-11-0xcc434d58-405931366", "/tmp/block_0x0bbe9f14-12-0xce61967d-874423181", "/tmp/block_0x0bbe9f14-13-0x05a20b35-267153288", "/tmp/block_0x0bbe9f14-14-0x4049f61f-606653767", "/tmp/block_0x0bbe9f14-15-0x46d473d2-614457338", "/tmp/block_0x0bbe9f14-16-0x35cf5500-411906321", "/tmp/block_0x0bbe9f14-17-0x79222961-278569788", "/tmp/block_0x0bbe9f14-18-0xad84e7b1-095032683", "/tmp/block_0x0bbe9f14-19-0x4bd48260-019097038", "/tmp/block_0x0bbe9f14-20-0x1517411d-292624085", "/tmp/block_0x0bbe9f14-21-0x6857e350-971385904", "/tmp/block_0x0bbe9f14-22-0xbe3ae2ca-236639695"]

```
Files are created in a temp-location, with the naming standard `block_<blockhash:4>-<txindex>-<txhash:4>-<random suffix>`. Each opcode immediately streams to file, with no in-geth buffering aside from whatever buffering the os normally does.

On the server side, it also adds some more info when regenerating historical state, namely, the reexec-number if `required historical state is not avaiable` is encountered, so a user can experiment with increasing that setting. It also prints out the remaining block until it reaches target:

```
INFO [10-15|13:48:25.263] Regenerating historical state            block=2385959 target=2386012 remaining=53   elapsed=3m30.990537767s
INFO [10-15|13:48:33.342] Regenerating historical state            block=2386012 target=2386012 remaining=0    elapsed=3m39.070073163s
INFO [10-15|13:48:33.343] Historical state regenerated             block=2386012 elapsed=3m39.070454362s nodes=10.03mB preimages=652.08kB
INFO [10-15|13:48:33.352] Wrote trace                              file=/tmp/block_0x14490c57-0-0xfbbd6d91-715824834
INFO [10-15|13:48:33.352] Wrote trace                              file=/tmp/block_0x14490c57-1-0x71076194-187462969
INFO [10-15|13:48:34.421] Wrote trace file=/tmp/block_0x14490c57-2-0x3f4263fe-056924484
```

The `options` is as follows:
```
type StdTraceConfig struct {
  *vm.LogConfig
  Reexec *uint64
  TxHash *common.Hash
}
```

### debug_standardTraceBadBlockToFile

This method is similar to `debug_standardTraceBlockToFile`, but can be used to obtain info about a block which has been _rejected_ as invalid (for some reason).

### debug_traceTransaction

**OBS** In most scenarios, `debug.standardTraceBlockToFile` is better suited for tracing!

The `traceTransaction` debugging method will attempt to run the transaction in the exact same manner
as it was executed on the network. It will replay any transaction that may have been executed prior
to this one before it will finally attempt to execute the transaction that corresponds to the given
hash.

In addition to the hash of the transaction you may give it a secondary *optional* argument, which
specifies the options for this specific call. The possible options are:

* `disableStorage`: `BOOL`. Setting this to true will disable storage capture (default = false).
* `disableMemory`: `BOOL`. Setting this to true will disable memory capture (default = false).
* `disableStack`: `BOOL`. Setting this to true will disable stack capture (default = false).
* `tracer`: `STRING`. Setting this will enable JavaScript-based transaction tracing, described below. If set, the previous four arguments will be ignored.
* `timeout`: `STRING`. Overrides the default timeout of 5 seconds for JavaScript-based tracing calls. Valid values are described [here](https://golang.org/pkg/time/#ParseDuration).

| Client  | Method invocation                                                                            |
|:--------|----------------------------------------------------------------------------------------------|
| Go      | `debug.TraceTransaction(txHash common.Hash, logger *vm.LogConfig) (*ExecutionResurt, error)` |
| Console | `debug.traceTransaction(txHash, [options])`                                                  |
| RPC     | `{"method": "debug_traceTransaction", "params": [txHash, {}]}`                               |

#### Example

```javascript
> debug.traceTransaction("0x2059dd53ecac9827faad14d364f9e04b1d5fe5b506e3acc886eff7a6f88a696a")
{
  gas: 85301,
  returnValue: "",
  structLogs: [{
      depth: 1,
      error: "",
      gas: 162106,
      gasCost: 3,
      memory: null,
      op: "PUSH1",
      pc: 0,
      stack: [],
      storage: {}
  },
    /* snip */
  {
      depth: 1,
      error: "",
      gas: 100000,
      gasCost: 0,
      memory: ["0000000000000000000000000000000000000000000000000000000000000006", "0000000000000000000000000000000000000000000000000000000000000000", "0000000000000000000000000000000000000000000000000000000000000060"],
      op: "STOP",
      pc: 120,
      stack: ["00000000000000000000000000000000000000000000000000000000d67cbec9"],
      storage: {
        0000000000000000000000000000000000000000000000000000000000000004: "8241fa522772837f0d05511f20caa6da1d5a3209000000000000000400000001",
        0000000000000000000000000000000000000000000000000000000000000006: "0000000000000000000000000000000000000000000000000000000000000001",
        f652222313e28459528d920b65115c16c04f3efc82aaedc97be59f3f377c0d3f: "00000000000000000000000002e816afc1b5c0f39852131959d946eb3b07b5ad"
      }
  }]
```


#### JavaScript-based tracing

Specifying the `tracer` option in the second argument enables JavaScript-based tracing. In this mode, `tracer` is interpreted as a JavaScript expression that is expected to evaluate to an object which must expose the `result` and `fault` methods. There exist 3 additional methods, namely: `step`, `enter` and `exit`. You must provide either `step`, or `enter` AND `exit` (i.e. these two must be exposed together). You may expose all three if you choose to do so.

##### Step

`step`is a function that takes two arguments, log and db, and is called for each step of the EVM, or when an error occurs, as the specified transaction is traced.

`log` has the following fields:

 - `op`: Object, an OpCode object representing the current opcode
 - `stack`: array[big.Int], the EVM execution stack
 - `memory`: Object, a structure representing the contract's memory space
 - `contract`: Object, an object representing the account executing the current operation

and the following methods:

 - `getPC()` - returns a Number with the current program counter
 - `getGas()` - returns a Number with the amount of gas remaining
 - `getCost()` - returns the cost of the opcode as a Number
 - `getDepth()` - returns the execution depth as a Number
 - `getRefund()` - returns the amount to be refunded as a Number
 - `getError()` - returns information about the error if one occured, otherwise returns `undefined`

If error is non-empty, all other fields should be ignored.

For efficiency, the same `log` object is reused on each execution step, updated with current values; make sure to copy values you want to preserve beyond the current call. For instance, this step function will not work:

    function(log) {
      this.logs.append(log);
    }

But this step function will:

    function(log) {
      this.logs.append({gas: log.getGas(), pc: log.getPC(), ...});
    }

`log.op` has the following methods:

 - `isPush()` - returns true iff the opcode is a PUSHn
 - `toString()` - returns the string representation of the opcode
 - `toNumber()` - returns the opcode's number

`log.memory` has the following methods:

 - `slice(start, stop)` - returns the specified segment of memory as a byte slice
 - `getUint(offset)` - returns the 32 bytes at the given offset

`log.stack` has the following methods:

 - `peek(idx)` - returns the idx-th element from the top of the stack (0 is the topmost element) as a big.Int
 - `length()` - returns the number of elements in the stack

`log.contract` has the following methods:

- `getCaller()` - returns the address of the caller
- `getAddress()` - returns the address of the current contract
- `getValue()` - returns the amount of value sent from caller to contract as a big.Int
- `getInput()` - returns the input data passed to the contract

`db` has the following methods:

 - `getBalance(address)` - returns a `big.Int` with the specified account's balance
 - `getNonce(address)` - returns a Number with the specified account's nonce
 - `getCode(address)` - returns a byte slice with the code for the specified account
 - `getState(address, hash)` - returns the state value for the specified account and the specified hash
 - `exists(address)` - returns true if the specified address exists

If the step function throws an exception or executes an illegal operation at any point, it will not be called on any further VM steps, and the error will be returned to the caller.

##### Result

`result` is a function that takes two arguments `ctx` and `db`, and is expected to return a JSON-serializable value to return to the RPC caller.

`ctx` is the context in which the transaction is executing and has the following fields:

- `type` - String, one of the two values `CALL` and `CREATE`
- `from` - Address, sender of the transaction
- `to` - Address, target of the transaction
- `input` - Buffer, input transaction data
- `gas` - Number, gas budget of the transaction
- `value` - big.Int, amount to be transferred in wei
- `block` - Number, block number
- `output` - Buffer, value returned from EVM
- `gasUsed` - Number, amount of gas used in executing the transaction (excludes txdata costs)
- `time` - String, execution runtime

##### Fault

`fault` is a function that takes two arguments, `log` and `db`, just like `step` and is invoked when an error happens during the execution of an opcode which wasn't reported in `step`. The method `log.getError()` has information about the error.

##### Enter & Exit

`enter` and `exit` are respectively invoked on stepping in and out of an internal call. More specifically they are invoked on the `CALL` variants, `CREATE` variants and also for the transfer implied by a `SELFDESTRUCT`.

`enter` takes a `callFrame` object as argument which has the following methods:

- `getType()` - returns a string which has the type of the call frame
- `getFrom()` - returns the address of the call frame sender
- `getTo()` - returns the address of the call frame target
- `getInput()` - returns the input as a buffer
- `getGas()` - returns a Number which has the amount of gas provided for the frame
- `getValue()` - returns a `big.Int` with the amount to be transferred only if available, otherwise `undefined`

`exit` takes in a `frameResult` object which has the following methods:

- `getGasUsed()` - returns amount of gas used throughout the frame as a Number
- `getOutput()` - returns the output as a buffer
` -getError()` - returns an error if one occured during execution and `undefined` otherwise 

##### Usage

Note that several values are Golang big.Int objects, not JavaScript numbers or JS bigints. As such, they have the same interface as described in the godocs. Their default serialization to JSON is as a Javascript number; to serialize large numbers accurately call `.String()` on them. For convenience, `big.NewInt(x)` is provided, and will convert a uint to a Go BigInt.

Usage example, returns the top element of the stack at each CALL opcode only:

    debug.traceTransaction(txhash, {tracer: '{data: [], fault: function(log) {}, step: function(log) { if(log.op.toString() == "CALL") this.data.push(log.stack.peek(0)); }, result: function() { return this.data; }}'});

### debug_traceCall

The `debug_traceCall` method lets you run an `eth_call` within the context of the given block execution using the final state of parent block as the base. The block can be specified either by hash or by number. It takes the same input object as a `eth_call`.
It returns the same output as `debug_traceTransaction`. A tracer can be specified as a third argument, similar to `debug_traceTransaction`.

`Object` - The transaction call object
- `from`: `DATA`, 20 Bytes - (optional) The address the transaction is sent from.
- `to`: `DATA`, 20 Bytes - The address the transaction is directed to.
- `gas`: `QUANTITY` - (optional) Integer of the gas provided for the transaction execution. eth_call consumes zero gas, but this parameter may be needed by some executions.
- `gasPrice`: `QUANTITY` - (optional) Integer of the gasPrice used for each paid gas
- `value`: `QUANTITY` - (optional) Integer of the value sent with this transaction
- `data`: `DATA` - (optional) Hash of the method signature and encoded parameters. For details see Ethereum Contract ABI in the Solidity documentation

| Client  | Method invocation                 |
|:-------:|-----------------------------------|
| Go      | `debug.TraceCall(args ethapi.CallArgs, blockNrOrHash rpc.BlockNumberOrHash, config *TraceConfig) (*ExecutionResult, error)` |
| Console | `debug.traceCall(object, blockNrOrHash, [options])`                   |
| RPC     | `{"method": "debug_traceCall", "params": [object, blockNrOrHash, {}]}`     |

#### Example

No specific call options:
```
> debug.traceCall(null, "0x0")
{
  failed: false,
  gas: 53000,
  returnValue: "",
  structLogs: []
}
```
Tracing a call with a destination and specific sender, disabling the storage and memory output (less data returned over RPC)
```
debug.traceCall({
	"from": "0xdeadbeef29292929192939494959594933929292",
	"to": "0xde929f939d939d393f939393f93939f393929023",
	"gas": "0x7a120",
	"data": "0xf00d4b5d00000000000000000000000001291230982139282304923482304912923823920000000000000000000000001293123098123928310239129839291010293810"
	},
	"latest", {"disableStorage": true, "disableMemory": true})
```
Curl example: 
```
> curl -H "Content-Type: application/json" -X POST  localhost:8545 --data '{"jsonrpc":"2.0","method":"debug_traceCall","params":[null, "pending"],"id":1}'
{"jsonrpc":"2.0","id":1,"result":{"gas":53000,"failed":false,"returnValue":"","structLogs":[]}}
```


### debug_verbosity

Sets the logging verbosity ceiling. Log messages with level
up to and including the given level will be printed.

The verbosity of individual packages and source files
can be raised using `debug_vmodule`.

| Client  | Method invocation                                 |
|:--------|---------------------------------------------------|
| Console | `debug.verbosity(level)`                          |
| RPC     | `{"method": "debug_vmodule", "params": [number]}` |

### debug_vmodule

Sets the logging verbosity pattern.

| Client  | Method invocation                                 |
|:--------|---------------------------------------------------|
| Console | `debug.vmodule(string)`                           |
| RPC     | `{"method": "debug_vmodule", "params": [string]}` |


#### Examples

If you want to see messages from a particular Go package (directory)
and all subdirectories, use:

``` javascript
> debug.vmodule("eth/*=6")
```

If you want to restrict messages to a particular package (e.g. p2p)
but exclude subdirectories, use:

``` javascript
> debug.vmodule("p2p=6")
```

If you want to see log messages from a particular source file, use

``` javascript
> debug.vmodule("server.go=6")
```

You can compose these basic patterns. If you want to see all
output from peer.go in a package below eth (eth/peer.go,
eth/downloader/peer.go) as well as output from package p2p
at level <= 5, use:

``` javascript
debug.vmodule("eth/*/peer.go=6,p2p=5")
```

### debug_writeBlockProfile

Writes a goroutine blocking profile to the given file.

| Client  | Method invocation                                           |
|:--------|-------------------------------------------------------------|
| Console | `debug.writeBlockProfile(file)`                             |
| RPC     | `{"method": "debug_writeBlockProfile", "params": [string]}` |

### debug_writeMemProfile

Writes an allocation profile to the given file.
Note that the profiling rate cannot be set through the API,
it must be set on the command line using the `--pprof.memprofilerate`
flag.

| Client  | Method invocation                                           |
|:--------|-------------------------------------------------------------|
| Console | `debug.writeMemProfile(file string)`                        |
| RPC     | `{"method": "debug_writeBlockProfile", "params": [string]}` |
