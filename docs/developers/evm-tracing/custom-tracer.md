---
title: Custom EVM tracer
description: Introduction to writing custom tracers for Geth
---

In addition to the default opcode tracer and the built-in tracers, Geth offers the possibility to write custom code that hook to events in the EVM to process and return the data in a consumable format. Custom tracers can be written either in Javascript or Go. JS tracers are good for quick prototyping and experimentation as well as for less intensive applications. Go tracers are performant but require the tracer to be compiled together with the Geth source code.

## Custom Go tracing

Custom tracers can also be made more performant by writing them in Go. The gain in performance mostly comes from the fact that Geth doesn't need to interpret JS code and can execute native functions. Geth comes with several built-in [native tracers](https://github.com/ethereum/go-ethereum/tree/master/eth/tracers/native) which can serve as examples. Please note that unlike JS tracers, Go tracing scripts cannot be simply passed as an argument to the API. They will need to be added to and compiled with the rest of the Geth source code.

In this section a simple native tracer that counts the number of opcodes will be covered. First follow the instructions to [clone and build](/docs/getting-started/installing-geth) Geth from source code. Next save the following snippet as a `.go` file and add it to `eth/tracers/native`:

```go
package native

import (
    "encoding/json"
    "math/big"
    "sync/atomic"
    "time"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/vm"
    "github.com/ethereum/go-ethereum/eth/tracers"
)

func init() {
    // This is how Geth will become aware of the tracer and register it under a given name
    register("opcounter", newOpcounter)
}

type opcounter struct {
    env       *vm.EVM
    counts    map[string]int // Store opcode counts
    interrupt uint32         // Atomic flag to signal execution interruption
    reason    error          // Textual reason for the interruption
}

func newOpcounter(ctx *tracers.Context, cfg json.RawMessage) tracers.Tracer {
    return &opcounter{counts: make(map[string]int)}
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *opcounter) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
        t.env = env
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *opcounter) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
    // Skip if tracing was interrupted
    if atomic.LoadUint32(&t.interrupt) > 0 {
        t.env.Cancel()
        return
    }

    name := op.String()
    if _, ok := t.counts[name]; !ok {
        t.counts[name] = 0
    }
    t.counts[name]++
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *opcounter) CaptureEnter(op vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *opcounter) CaptureExit(output []byte, gasUsed uint64, err error) {}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *opcounter) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *opcounter) CaptureEnd(output []byte, gasUsed uint64, _ time.Duration, err error) {}

func (*opcounter) CaptureTxStart(gasLimit uint64) {}

func (*opcounter) CaptureTxEnd(restGas uint64) {}

// GetResult returns the json-encoded nested list of call traces, and any
// error arising from the encoding or forceful termination (via `Stop`).
func (t *opcounter) GetResult() (json.RawMessage, error) {
    res, err := json.Marshal(t.counts)
    if err != nil {
        return nil, err
    }
    return res, t.reason
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *opcounter) Stop(err error) {
    t.reason = err
    atomic.StoreUint32(&t.interrupt, 1)
}
```

Every method of the [EVMLogger interface](https://pkg.go.dev/github.com/ethereum/go-ethereum/core/vm#EVMLogger) needs to be implemented (even if empty). Key parts to notice are the `init()` function which registers the tracer in Geth, the `CaptureState` hook where the opcode counts are incremented and `GetResult` where the result is serialized and delivered. Note that the constructor takes in a `cfg json.RawMessage`. This will be filled with a JSON object that user provides to the tracer to pass in optional config fields.

To test out this tracer the source is first compiled with `make geth`. Then in the console it can be invoked through the usual API methods by passing in the name it was registered under:

```sh
> debug.traceTransaction('0x7ae446a7897c056023a8104d254237a8d97783a92900a7b0f7db668a9432f384', { tracer: 'opcounter' })
{
    ADD: 4,
    AND: 3,
    CALLDATALOAD: 2,
    ...
}
```

## Custom Javascript tracing

Transaction traces include the complete status of the EVM at every point during the transaction execution, which can be a very large amount of data. Often, users are only interested in a small subset of that data. Javascript trace filters are available to isolate the useful information.

Specifying the `tracer` option in one of the tracing methods (see list in [reference](/docs/interacting-with-geth/rpc/ns-debug)) enables JavaScript-based tracing. In this mode, `tracer` is interpreted as a JavaScript expression that is expected to evaluate to an object which must expose the `result` and `fault` methods. There exist 4 additional methods, namely: `setup`, `step`, `enter`, and `exit`. `enter` and `exit` must be present or omitted together.

### Setup

`setup` is invoked once, in the beginning when the tracer is being constructed by Geth for a given transaction. It takes in one argument `config`. `config` is tracer-specific and allows users to pass in options to the tracer. `config` is to be JSON-decoded for usage and its default value is `"{}"`.

The `config` in the following example is the `onlyTopCall` option available in the `callTracer`:

```js
debug.traceTransaction('<txhash>, { tracer: 'callTracer', tracerConfig: { onlyTopCall: true } })
```

The config in the following example is the `diffMode` option available in the `prestateTracer`:

```js
debug.traceTransaction('<txhash>, { tracer: 'prestateTracer': tracerConfig: { diffMode: true } })
```

### Step

`step` is a function that takes two arguments, `log` and `db`, and is called for each step of the EVM, or when an error occurs, as the specified transaction is traced.

`log` has the following fields:

- `op`: Object, an OpCode object representing the current opcode
- `stack`: Object, a structure representing the EVM execution stack
- `memory`: Object, a structure representing the contract's memory space
- `contract`: Object, an object representing the account executing the current operation

and the following methods:

- `getPC()` - returns a Number with the current program counter
- `getGas()` - returns a Number with the amount of gas remaining
- `getCost()` - returns the cost of the opcode as a Number
- `getDepth()` - returns the execution depth as a Number
- `getRefund()` - returns the amount to be refunded as a Number
- `getError()` - returns information about the error if one occurred, otherwise returns `undefined`

If error is non-empty, all other fields should be ignored.

For efficiency, the same `log` object is reused on each execution step, updated with current values; make sure to copy values you want to preserve beyond the current call. For instance, this step function will not work:

```js
function(log) {
  this.logs.append(log);
}
```

But this step function will:

```js
function(log) {
  this.logs.append({gas: log.getGas(), pc: log.getPC(), ...});
}
```

`log.op` has the following methods:

- `isPush()` - returns true if the opcode is a PUSHn
- `toString()` - returns the string representation of the opcode
- `toNumber()` - returns the opcode's number

`log.memory` has the following methods:

- `slice(start, stop)` - returns the specified segment of memory as a byte slice
- `getUint(offset)` - returns the 32 bytes at the given offset
- `length()` - returns the memory size

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

### Result

`result` is a function that takes two arguments `ctx` and `db`, and is expected to return a JSON-serializable value to return to the RPC caller.

`ctx` is the context in which the transaction is executing and has the following fields:

- `type` - String, one of the two values `CALL` and `CREATE`
- `from` - Address, sender of the transaction
- `to` - Address, target of the transaction
- `input` - Buffer, input transaction data
- `gas` - Number, gas budget of the transaction
- `gasUsed` - Number, amount of gas used in executing the transaction (excludes txdata costs)
- `gasPrice` - Number, gas price configured in the transaction being executed
- `value` - big.Int, amount to be transferred in wei
- `block` - Number, block number
- `output` - Buffer, value returned from EVM
- `error` - String, non-empty if there was an EVM error

And these fields are only available for tracing mined transactions (i.e. not available when doing `debug_traceCall`):

- `blockHash` - Buffer, hash of the block that holds the transaction being executed
- `txIndex` - Number, index of the transaction being executed in the block
- `txHash` - Buffer, hash of the transaction being executed

### Fault

`fault` is a function that takes two arguments, `log` and `db`, just like `step` and is invoked when an error happens during the execution of an opcode which wasn't reported in `step`. The method `log.getError()` has information about the error.

### Enter & Exit

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
- `getError()` - returns an error if one occurred during execution and `undefined` otherwise

**Usage:**

Note that several values are Golang `big.Int` objects, not JavaScript numbers or JS bigints. As such, they have the same interface as described in the godocs. Their default serialization to JSON is as a Javascript number; to serialize large numbers accurately call `.String()` on them. For convenience, `big.NewInt(x)` is provided, and will convert a uint to a Go BigInt.

Here is an example, returns the top element of the stack at each CALL opcode only:

```js
debug.traceTransaction(txhash, {
  tracer:
    '{data: [], fault: function(log) {}, step: function(log) { if(log.op.toString() == "CALL") this.data.push(log.stack.peek(0)); }, result: function() { return this.data; }}'
});
```

## Other traces

This tutorial has focused on `debug_traceTransaction()` which reports information about individual transactions. There are also RPC endpoints that provide different information, including tracing the EVM execution within a block, between two blocks, for specific `eth_call`s or rejected blocks. The full list of trace functions can be explored in the [reference documentation](/docs/interacting-with-geth/rpc/ns-debug).

## Summary

This page described how to write custom tracers for Geth. Custom tracers can be written in Javascript or Go.
