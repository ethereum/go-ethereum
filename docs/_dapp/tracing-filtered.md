---
title: Filtered Tracing
sort_key: B
---

In the previous section you learned how to create a complete trace. However, those traces can include the complete status of the EVM at every point 
in the execution, which is huge. Usually you are only interested in a small subset of this information. To get it, you can specify a JavaScript filter.

**Note:** The JavaScript package used by Geth is [Goja](https://github.com/dop251/goja), which is only up to the
[ECMAScript 5.1 standard](https://262.ecma-international.org/5.1/). This means we cannot use [arrow functions](https://www.w3schools.com/js/js_arrow_function.asp)
and [template literals](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Template_literals).


## Running a Simple Trace

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

   We could specify this command directly in the JavaScript console, but it would be excessively long and unwieldy.
   
2. Run the [JavaScript console](https://geth.ethereum.org/docs/interface/javascript-console). 
3. Get the hash of a recent transaction. For example, if you use the Goerli network, you can get such a value
   [here](https://goerli.etherscan.io/).
4. Run this command to run the script:

   ```javascript
   loadScript("filterTrace_1.js")
   ```

5. Run the trancer from the script. Be patient, it could take a long time.

   ```javascript
   tracer("<hash of transaction>")
   ```
   
   The bottom of the output looks similar to:
   ```json
   ...
   "3366:POP", "3367:JUMP", "1355:JUMPDEST", "1356:PUSH1", "1358:MLOAD", "1359:DUP1", "1360:DUP3", "1361:ISZERO", "1362:ISZERO", 
   "1363:ISZERO", "1364:ISZERO", "1365:DUP2", "1366:MSTORE", "1367:PUSH1", "1369:ADD", "1370:SWAP2", "1371:POP", "1372:POP", "1373:PUSH1", 
   "1375:MLOAD", "1376:DUP1", "1377:SWAP2", "1378:SUB", "1379:SWAP1", "1380:RETURN"]
   ```
   
6. This output isn't very readable. Run this line to get a more readable output:

   ```javascript
   console.log(JSON.stringify(tracer("<hash of transaction>"), null, 2))
   ```
   
   You can read about thhe `JSON.stringify` function 
   [here](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/JSON/stringify).
   
### How Does It Work?

We call the same `debug.traceTransaction` function we use for [basic traces](https://geth.ethereum.org/docs/dapp/tracing), but
with a new parameter, `tracer`. This parameter is a string that is the JavaScript object we use. In the case of the trace
above, it is:

```javascript
{
   retVal: [],
   step: function(log,db) {this.retVal.push(log.getPC() + ":" + log.op.toString())},
   fault: function(log,db) {this.retVal.push("FAULT: " + JSON.stringify(log))},
   result: function(ctx,db) {return this.retVal}
}
```

This object has to have three member functions:

- `step`, called for each opcode
- `fault`, called if there is a problem in the execution
- `result`, called to produce the results that are returned by `debug.traceTransaction` after the execution is done

It can have additional members. In this case, we use `retVal` to store the list of strings that we'll return in `result`.

The `step` function here adds to `retVal` the program counter and the name of the opcode there. Then, in `result`, we return this
list to be sent to the caller.


## Actual Filtering

For actual filtered tracing we need an `if` statement to only log relevant information. For example, if we are interested in
the transaction's interaction with storage, we might use:

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

```json
[
  "5921: SLOAD",
  "5936: SSTORE",
  "12734: SLOAD",
  "12739: SLOAD",
  "1209: SLOAD",
  "8310: SLOAD",
  "8360: SSTORE",
  "11409: SSTORE",
  "4065: SLOAD",
  "4081: SSTORE",
  "2117: SLOAD",
  "2413: SSTORE",
  "2420: SLOAD",
  "2475: SSTORE",
  "6094: SSTORE"
]
```


## Stack Information

The trace above tells us the PC and whether the program read from storage or wrote to it. That isn't very
useful. To know more, you can use the `log.stack.peek` function to peek into the stack. `log.stack.peek(0)`
is the stack top, `log.stack.peek(1)` the entry below it, etc. The values returned by `log.stack.peek` are
Go `big.Int` objects. By default they are converted to JavaScript floating point numbers, so you need 
`toString(16)` to get them as hexadecimals, which is how we normally represent 256-bit values such as
storage cells and their content.

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

This function gives you a trace of all the storage operations, and show you their parameters. This gives
you a more complete picture of the program's interaction with storage. The output is similar to:

```json
[
  "5921: SLOAD 0",
  "5936: SSTORE 0 <- 2",
  "12734: SLOAD bbbc8f1fb1317a27197ac2561e8db9a96b185f7d5ed27d9e72ced42831f4b313",
  "12739: SLOAD bbbc8f1fb1317a27197ac2561e8db9a96b185f7d5ed27d9e72ced42831f4b314",
  "1209: SLOAD 2",
  "8310: SLOAD 6",
  "8360: SSTORE 6 <- 1e66a40569b713c0000000000000000000002a4cdfba02d",
  "11409: SSTORE bbbc8f1fb1317a27197ac2561e8db9a96b185f7d5ed27d9e72ced42831f4b313 <- 4cfa320000000003e2c5a4a4941972ea530000000000fb75538b44eec5420a",
  "4065: SLOAD 3f0af0a7a3ed17f5ba6a93e0a2a05e766ed67bf82195d2dd15feead3749a575d",
  "4081: SSTORE 3f0af0a7a3ed17f5ba6a93e0a2a05e766ed67bf82195d2dd15feead3749a575d <- 4f688248dcd61ec96a4",
  "2117: SLOAD 3f0af0a7a3ed17f5ba6a93e0a2a05e766ed67bf82195d2dd15feead3749a575d",
  "2413: SSTORE 3f0af0a7a3ed17f5ba6a93e0a2a05e766ed67bf82195d2dd15feead3749a575d <- fb8629ad13d9a12456",
  "2420: SLOAD cc39b177dd3a7f50d4c09527584048378a692aed24d31d2eabeddb7f3c041870",
  "2475: SSTORE cc39b177dd3a7f50d4c09527584048378a692aed24d31d2eabeddb7f3c041870 <- 358c3de691bd19",
  "6094: SSTORE 0 <- 1"
]
```

## Operation Results

One piece of information missing from the function above is the result on an `SLOAD` operation. The 
state we get inside `log` is the state prior to the execution of the opcode, so that value is not
known yet.



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

## Dealing with intra-Contract Calls

   
## Conclusion

This tutorial only taught the basics of using JavaScript to filter traces. We did not go over access to memory,
or how to use the `db` parameter to know the state of the chain at the time of execution. All this and more is
covered [in the reference](https://geth.ethereum.org/docs/rpc/ns-debug#javascript-based-tracing).

Hopefully with this tool you will find it easier to step over and debug thorny issues with contracts and the EVM.

Original version by [Ori Pomerantz](qbzzt1@gmail.com)
