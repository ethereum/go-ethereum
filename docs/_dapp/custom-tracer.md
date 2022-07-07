---
title: Custom EVM tracer
sort_key: B
---

In addition to the default opcode tracer and the built-in tracers, Geth offers the possibility to write custom code
that hook to events in the EVM to process and return the data in a consumable format. Custom tracers can be
written either in Javascript or Go. JS tracers are good for quick prototyping and experimentation as well as for
less intensive applications. Go tracers are performant but require Geth to be compiled with the tracer.

## Javascript trace filters

Transaction traces include the complete status of the EVM at every point during the transaction execution, which
can be a very large amount of data. Often, users are only interested in a small subset of that data. Javascript trace
filters are available to isolate the useful information. Detailed information about `debug_traceTransaction` and its
component parts is available in the [reference documentation](/docs/rpc/ns-debug#debug_tracetransaction).

### A simple filter

Filters are Javascript functions that select information from the trace to persist and discard based on some
conditions. The following Javascript function returns only the sequence of opcodes executed by the transaction as a
comma-separated list. The function could be written directly in the Javascript console, but it is cleaner to 
write it in a separate re-usable file and load it into the console. 

1. Create a file, `filterTrace_1.js`, with this content:

   ```javascript

   tracer = function(tx) {
      return debug.traceTransaction(tx, {tracer:
         '{' +
            'retVal: [],' +
            'step: function(log,db) {this.retVal.push(log.getPC() + ":" + log.op.toString())},' +
            'fault: function(log,db) {this.retVal.push("FAULT: " + JSON.stringify(log))},' +
            'result: function(ctx,db) {return this.retVal}' +
         '}'
      }) // return debug.traceTransaction ...
   }   // tracer = function ...

   ```

2. Run the [JavaScript console](https://geth.ethereum.org/docs/interface/javascript-console).
   
3. Get the hash of a recent transaction from a node or block explorer.

4. Run this command to run the script:

   ```javascript
   loadScript("filterTrace_1.js")
   ```

5. Run the tracer from the script. Be patient, it could take a long time.

   ```javascript
   tracer("<hash of transaction>")
   ```

   The bottom of the output looks similar to:
   ```sh
   "3366:POP", "3367:JUMP", "1355:JUMPDEST", "1356:PUSH1", "1358:MLOAD", "1359:DUP1", "1360:DUP3", "1361:ISZERO", "1362:ISZERO",
   "1363:ISZERO", "1364:ISZERO", "1365:DUP2", "1366:MSTORE", "1367:PUSH1", "1369:ADD", "1370:SWAP2", "1371:POP", "1372:POP", "1373:PUSH1",
   "1375:MLOAD", "1376:DUP1", "1377:SWAP2", "1378:SUB", "1379:SWAP1", "1380:RETURN"
   ```

6. Run this line to get a more readable output with each string in its own line.

   ```javascript
   console.log(JSON.stringify(tracer("<hash of transaction>"), null, 2))
   ```

More information about the `JSON.stringify` function is available
[here](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/JSON/stringify). 

The commands above worked by calling the same `debug.traceTransaction` function that was previously 
explained in [basic traces](https://geth.ethereum.org/docs/dapp/tracing), but with a new parameter, `tracer`. 
This parameter takes the JavaScript object formated as a string. In the case of the trace above, it is:

```javascript
{
   retVal: [],
   step: function(log,db) {this.retVal.push(log.getPC() + ":" + log.op.toString())},
   fault: function(log,db) {this.retVal.push("FAULT: " + JSON.stringify(log))},
   result: function(ctx,db) {return this.retVal}
}
```
This object has three member functions:

- `step`, called for each opcode.
- `fault`, called if there is a problem in the execution.
- `result`, called to produce the results that are returned by `debug.traceTransaction` after the execution is done.

In this case, `retVal` is used to store the list of strings to return in `result`.

The `step` function adds to `retVal` the program counter and the name of the opcode there. Then, in `result`, this
list is returned to be sent to the caller.


### Filtering with conditions

For actual filtered tracing we need an `if` statement to only log relevant information. For example, to isolate
the transaction's interaction with storage, the following tracer could be used:

```javascript
tracer = function(tx) {
      return debug.traceTransaction(tx, {tracer:
      '{' +
         'retVal: [],' +
         'step: function(log,db) {' +
         '   if(log.op.toNumber() == 0x54) ' +
         '     this.retVal.push(log.getPC() + ": SLOAD");' +
         '   if(log.op.toNumber() == 0x55) ' +
         '     this.retVal.push(log.getPC() + ": SSTORE");' +
         '},' +
         'fault: function(log,db) {this.retVal.push("FAULT: " + JSON.stringify(log))},' +
         'result: function(ctx,db) {return this.retVal}' +
      '}'
      }) // return debug.traceTransaction ...
}   // tracer = function ...
```

The `step` function here looks at the opcode number of the op, and only pushes an entry if the opcode is
`SLOAD` or `SSTORE` ([here is a list of EVM opcodes and their numbers](https://github.com/wolflo/evm-opcodes)).
We could have used `log.op.toString()` instead, but it is faster to compare numbers rather than strings.

The output looks similar to this:

```javascript
[
  "5921: SLOAD",
  .
  .
  .
  "2413: SSTORE",
  "2420: SLOAD",
  "2475: SSTORE",
  "6094: SSTORE"
]
```


### Stack Information

The trace above reports the program counter (PC) and whether the program read from storage or wrote to it. 
That alone isn't particularly useful. To know more, the `log.stack.peek` function can be used to peek 
into the stack. `log.stack.peek(0)` is the stack top, `log.stack.peek(1)` the entry below it, etc.

The values returned by `log.stack.peek` are Go `big.Int` objects. By default they are converted to JavaScript 
floating point numbers, so you need `toString(16)` to get them as hexadecimals, which is how 256-bit values such as
storage cells and their content are normally represented.

#### Storage Information

The function below provides a trace of all the storage operations and their parameters. This gives
a more complete picture of the program's interaction with storage. 

```javascript
tracer = function(tx) {
      return debug.traceTransaction(tx, {tracer:
      '{' +
         'retVal: [],' +
         'step: function(log,db) {' +
         '   if(log.op.toNumber() == 0x54) ' +
         '     this.retVal.push(log.getPC() + ": SLOAD " + ' +
         '        log.stack.peek(0).toString(16));' +
         '   if(log.op.toNumber() == 0x55) ' +
         '     this.retVal.push(log.getPC() + ": SSTORE " +' +
         '        log.stack.peek(0).toString(16) + " <- " +' +
         '        log.stack.peek(1).toString(16));' +
         '},' +
         'fault: function(log,db) {this.retVal.push("FAULT: " + JSON.stringify(log))},' +
         'result: function(ctx,db) {return this.retVal}' +
      '}'
      }) // return debug.traceTransaction ...
}   // tracer = function ...

```

The output is similar to:

```javascript
[
  "5921: SLOAD 0",
  .
  .
  .
  "2413: SSTORE 3f0af0a7a3ed17f5ba6a93e0a2a05e766ed67bf82195d2dd15feead3749a575d <- fb8629ad13d9a12456",
  "2420: SLOAD cc39b177dd3a7f50d4c09527584048378a692aed24d31d2eabeddb7f3c041870",
  "2475: SSTORE cc39b177dd3a7f50d4c09527584048378a692aed24d31d2eabeddb7f3c041870 <- 358c3de691bd19",
  "6094: SSTORE 0 <- 1"
]
```

#### Operation Results

One piece of information missing from the function above is the result on an `SLOAD` operation. The
state we get inside `log` is the state prior to the execution of the opcode, so that value is not
known yet. For more operations we can figure it out for ourselves, but we don't have access to the
storage, so here we can't.

The solution is to have a flag, `afterSload`, which is only true in the opcode right after an
`SLOAD`, when we can see the result at the top of the stack.

```javascript
tracer = function(tx) {
      return debug.traceTransaction(tx, {tracer:
      '{' +
         'retVal: [],' +
         'afterSload: false,' +
         'step: function(log,db) {' +
         '   if(this.afterSload) {' +
         '     this.retVal.push("    Result: " + ' +
         '          log.stack.peek(0).toString(16)); ' +
         '     this.afterSload = false; ' +
         '   } ' +
         '   if(log.op.toNumber() == 0x54) {' +
         '     this.retVal.push(log.getPC() + ": SLOAD " + ' +
         '        log.stack.peek(0).toString(16));' +
         '        this.afterSload = true; ' +
         '   } ' +
         '   if(log.op.toNumber() == 0x55) ' +
         '     this.retVal.push(log.getPC() + ": SSTORE " +' +
         '        log.stack.peek(0).toString(16) + " <- " +' +
         '        log.stack.peek(1).toString(16));' +
         '},' +
         'fault: function(log,db) {this.retVal.push("FAULT: " + JSON.stringify(log))},' +
         'result: function(ctx,db) {return this.retVal}' +
      '}'
      }) // return debug.traceTransaction ...
}   // tracer = function ...
```

The output now contains the result in the line that follows the `SLOAD`. 

```javascript
[
  "5921: SLOAD 0",
  "    Result: 1",
  .
  .
  .
  "2413: SSTORE 3f0af0a7a3ed17f5ba6a93e0a2a05e766ed67bf82195d2dd15feead3749a575d <- fb8629ad13d9a12456",
  "2420: SLOAD cc39b177dd3a7f50d4c09527584048378a692aed24d31d2eabeddb7f3c041870",
  "    Result: 0",
  "2475: SSTORE cc39b177dd3a7f50d4c09527584048378a692aed24d31d2eabeddb7f3c041870 <- 358c3de691bd19",
  "6094: SSTORE 0 <- 1"
]
```

### Dealing With Calls Between Contracts

So the storage has been treated as if there are only 2<sup>256</sup> cells. However, that is not true. 
Contracts can call other contracts, and then the storage involved is the storage of the other contract. 
We can see the address of the current contract in `log.contract.getAddress()`. This value is the execution 
context - the contract whose storage we are using - even when code from another contract is executed (by using
[`CALLCODE` or `DELEGATECALL`][solidity-delcall]).

However, `log.contract.getAddress()` returns an array of bytes. To convert this to the familiar hexadecimal
representation of Ethereum addresses, `this.byteHex()` and `array2Hex()` can be used.

```javascript
tracer = function(tx) {
      return debug.traceTransaction(tx, {tracer:
      '{' +
         'retVal: [],' +
         'afterSload: false,' +
         'callStack: [],' +

         'byte2Hex: function(byte) {' +
         '  if (byte < 0x10) ' +
         '      return "0" + byte.toString(16); ' +
         '  return byte.toString(16); ' +
         '},' +

         'array2Hex: function(arr) {' +
         '  var retVal = ""; ' +
         '  for (var i=0; i<arr.length; i++) ' +
         '    retVal += this.byte2Hex(arr[i]); ' +
         '  return retVal; ' +
         '}, ' +

         'getAddr: function(log) {' +
         '  return this.array2Hex(log.contract.getAddress());' +
         '}, ' +

         'step: function(log,db) {' +
         '   var opcode = log.op.toNumber();' +

         // SLOAD
         '   if (opcode == 0x54) {' +
         '     this.retVal.push(log.getPC() + ": SLOAD " + ' +
         '        this.getAddr(log) + ":" + ' +
         '        log.stack.peek(0).toString(16));' +
         '        this.afterSload = true; ' +
         '   } ' +

         // SLOAD Result
         '   if (this.afterSload) {' +
         '     this.retVal.push("    Result: " + ' +
         '          log.stack.peek(0).toString(16)); ' +
         '     this.afterSload = false; ' +
         '   } ' +

         // SSTORE
         '   if (opcode == 0x55) ' +
         '     this.retVal.push(log.getPC() + ": SSTORE " +' +
         '        this.getAddr(log) + ":" + ' +
         '        log.stack.peek(0).toString(16) + " <- " +' +
         '        log.stack.peek(1).toString(16));' +

         // End of step
         '},' +

         'fault: function(log,db) {this.retVal.push("FAULT: " + JSON.stringify(log))},' +

         'result: function(ctx,db) {return this.retVal}' +
      '}'
      }) // return debug.traceTransaction ...
}   // tracer = function ...
```

The output is similar to:

```javascript
[
  "423: SLOAD 22ff293e14f1ec3a09b137e9e06084afd63addf9:360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc",
  "    Result: 360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc",
  "10778: SLOAD 22ff293e14f1ec3a09b137e9e06084afd63addf9:6",
  "    Result: 6",
  .
  .
  .
  "13529: SLOAD f2d68898557ccb2cf4c10c3ef2b034b2a69dad00:8328de571f86baa080836c50543c740196dbc109d42041802573ba9a13efa340",
  "    Result: 8328de571f86baa080836c50543c740196dbc109d42041802573ba9a13efa340",
  "423: SLOAD f2d68898557ccb2cf4c10c3ef2b034b2a69dad00:360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc",
  "    Result: 360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc",
  "13529: SLOAD f2d68898557ccb2cf4c10c3ef2b034b2a69dad00:b38558064d8dd9c883d2a8c80c604667ddb90a324bc70b1bac4e70d90b148ed4",
  "    Result: b38558064d8dd9c883d2a8c80c604667ddb90a324bc70b1bac4e70d90b148ed4",
  "11041: SSTORE 22ff293e14f1ec3a09b137e9e06084afd63addf9:6 <- 0"
]
```


## Other traces

This tutorial has focused on `debug_traceTransaction()` which reports information about individual transactions. There are
also RPC endpoints that provide different information, including tracing the EVM execution within a block, between two blocks, 
for specific `eth_call`s or rejected blocks. The fill list of trace functions can be explored in the 
[reference documentation][debug-docs].


## Go Native Tracing

It is also possible to trace EVM execution using Go functions that wrap RPC calls to a Geth node. This provides 
programmatic access to trace dumps meaning retrieval and downstream analysis of the trace information can all be 
self-contained within a Go application. The source code for the tracers is available to browse on the
[Geth Github][go-tracer-source]. This page will demonstrate how to handle `traceTransaction` from a Go application.
The concepts covered here can be transferred to the other types of trace described in the reference documentation.

The tracers are implemnted as functions associated with the `debug` class. The arguments
are a `ctx` object (which configures the request), a transaction hash and a `TraceConfig` object. The `TraceConfig`
object has the following structure:

```go
type TraceConfig struct {
	*logger.Config
	Tracer  *string
	Timeout *string
	Reexec  *uint64
}
```

The `tracer` field in `TraceConfig` is a string-formatted Javascript object that configures the trace as described in
the [Javascript trace filters](#javascript-trace-filters) section. Timeout is a string that overrides the default 5 second
request timeout - the valid values are described in the [Go Time][go-time] documentation. `Reexec` is an integer that 
defines how many blocks behind the head of the chain to look for a checkpoint to rebuild the requested state from - large
values can lead to long state regeneration times. If the user encounter a `required historical state is not available` error
then adjusting `Reexec` is likely to fix it.

The `Tracetransaction()` Go function returns the structured logs created during the execution of EVM
and returns them as a JSON object. 

A call to `TraceTransaction()` in Go therefore looks as follows:

```go
trace, err := debug.TraceTransaction(txHash, traceConfig) (*ExecutionResult, error)
if err != nil {
	return nil, err
	}
```

[solidity-delcall]:https://docs.soliditylang.org/en/v0.8.14/introduction-to-smart-contracts.html#delegatecall-callcode-and-libraries
[debug-docs]: /docs/rpc/ns-debug
[go-tracer-source]: https://github.com/ethereum/go-ethereum/blob/master/eth/tracers/api.go
[go-time]:https://pkg.go.dev/time#ParseDuration
