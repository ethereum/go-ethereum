# Changelog

All notable changes to the tracing interface will be documented in this file.

## [Unreleased]

There have been minor backwards-compatible changes to the tracing interface to explicitly mark the execution of **system** contracts. As of now the only system call updates the parent beacon block root as per [EIP-4788](https://eips.ethereum.org/EIPS/eip-4788). Other system calls are being considered for the future hardfork.

### New methods

- `OnSystemCallStart()`: This hook is called when EVM starts processing a system call. Note system calls happen outside the scope of a transaction. This event will be followed by normal EVM execution events.
- `OnSystemCallEnd()`: This hook is called when EVM finishes processing a system call.

## [v1.14.0]

There has been a major breaking change in the tracing interface for custom native tracers. JS and built-in tracers are not affected by this change and tracing API methods may be used as before. This overhaul has been done as part of the new live tracing feature ([#29189](https://github.com/ethereum/go-ethereum/pull/29189)). To learn more about live tracing please refer to the [docs](https://geth.ethereum.org/docs/developers/evm-tracing/live-tracing).

**The `EVMLogger` interface which the tracers implemented has been removed.** It has been replaced by a new struct `tracing.Hooks`. `Hooks` keeps pointers to event listening functions. Internally the EVM will use these function pointers to emit events and can skip an event if the tracer has opted not to implement it. In fact this is the main reason for this change of approach. Another benefit is the ease of adding new hooks in future, and dynamically assigning event receivers.

The consequence of this change can be seen in the constructor of a tracer. Let's take the 4byte tracer as an example. Previously the constructor return an instance which satisfied the interface. Now it should return a pointer to `tracers.Tracer` (which is now also a struct as opposed to an interface) and explicitly assign the event listeners. As a side-benefit the tracers will not have to provide empty implementation of methods just to satisfy the interface:

```go
func newFourByteTracer(ctx *tracers.Context, _ json.RawMessage) (tracers.Tracer, error) {
	t := &fourByteTracer{
		ids: make(map[string]int),
	}
	return t, nil

}
```

And now:

```go
func newFourByteTracer(ctx *tracers.Context, _ json.RawMessage) (*tracers.Tracer, error) {
	t := &fourByteTracer{
		ids: make(map[string]int),
	}
	return &tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnTxStart: t.onTxStart,
			OnEnter:   t.onEnter,
		},
		GetResult: t.getResult,
		Stop:      t.stop,
	}, nil
}
```

### Event listeners

If you have sharp eyes you might have noticed the new names for `OnTxStart` and `OnEnter`, previously called `CaptureTxStart` and `CaptureEnter`. Indeed there have been various modifications to the signatures of the event listeners. All method names now follow the `On*` pattern instead of `Capture*`. However the modifications are not limited to the names.

#### New methods

The live tracing feature was half about adding more observability into the state of the blockchain. As such there have been a host of method additions. Please consult the [Hooks](./hooks.go) struct for the full list of methods. Custom tracers which are invoked through the API (as opposed to "live" tracers) can benefit from the following new methods:

- `OnGasChange(old, new uint64, reason GasChangeReason)`: This hook tracks the lifetime of gas within a transaction and its subcalls. It will first track the initial purchase of gas with ether, then the following consumptions and refunds of gas until at the end the rest is returned.
- `OnBalanceChange(addr common.Address, prev, new *big.Int, reason BalanceChangeReason)`: This hook tracks the balance changes of accounts. Where possible a reason is provided for the change (e.g. a transfer, gas purchase, withdrawal deposit etc).
- `OnNonceChange(addr common.Address, prev, new uint64)`: This hook tracks the nonce changes of accounts.
- `OnCodeChange(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte)`: This hook tracks the code changes of accounts.
- `OnStorageChange(addr common.Address, slot common.Hash, prev, new common.Hash)`: This hook tracks the storage changes of accounts.
- `OnLogChange(log *types.Log)`: This hook tracks the logs emitted by the EVM.

#### Removed methods

The hooks `CaptureStart` and `CaptureEnd` have been removed. These hooks signaled the top-level call frame of a transaction. The relevant info will be now emitted by `OnEnter` and `OnExit` which are emitted for every call frame. They now contain a `depth` parameter which can be used to distinguish the top-level call frame when necessary. The `create bool` parameter to `CaptureStart` can now be inferred from `typ byte` in `OnEnter`, i.e. `vm.OpCode(typ) == vm.CREATE`.

#### Modified methods

- `CaptureTxStart` -> `OnTxStart(vm *VMContext, tx *types.Transaction, from common.Address)`. It now emits the full transaction object as well as `from` which should be used to get the sender address. The `*VMContext` is a replacement for the `*vm.EVM` object previously passed to `CaptureStart`.
- `CaptureTxEnd` -> `OnTxEnd(receipt *types.Receipt, err error)`. It now returns the full receipt object.
- `CaptureEnter` -> `OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int)`. The new `depth int` parameter indicates the call stack depth. It is 0 for the top-level call. Furthermore, the location where `OnEnter` is called in the EVM is now made a soon as a call is started. This means some specific error cases that were not before calling `OnEnter/OnExit` will now do so, leading some transaction to have an extra call traced.
- `CaptureExit` -> `OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool)`. It has the new `depth` parameter, same as `OnEnter`. The new `reverted` parameter indicates whether the call frame was reverted.
- `CaptureState` -> `OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error)`. `op` is of type `byte` which can be cast to `vm.OpCode` when necessary. A `*vm.ScopeContext` is not passed anymore. It is replaced by `tracing.OpContext` which offers access to the memory, stack and current contract.
- `CaptureFault` -> `OnFault(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error)`. Similar to above.

[unreleased]: https://github.com/ethereum/go-ethereum/compare/v1.14.0...master
[v1.14.0]: https://github.com/ethereum/go-ethereum/releases/tag/v1.14.0