---
title: Built-in tracers
description: Explanation of the tracers that come bundled in Geth as part of the tracing API.
---

Geth comes bundled with a choice of tracers that can be invoked via the [tracing API](/docs/interacting-with-geth/rpc/ns-debug). Some of these built-in tracers are implemented natively in Go, and others in Javascript. The default tracer is the opcode logger (otherwise known as struct logger) which is the default tracer for all the methods. Other tracers have to be specified by passing their name to the `tracer` parameter in the API call.

## Struct/opcode logger {#struct-opcode-logger}

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

It is also possible to configure the trace by passing Boolean (true/false) values for four parameters that tweak the verbosity of the trace. By default, the _EVM memory_ and _Return data_ are not reported but the _EVM stack_ and _EVM storage_ are. To report the maximum amount of data:

```sh
enableMemory: true
disableStack: false
disableStorage: false
enableReturnData: true
```

An example call:

```js
debug.traceTransaction('0xfc9359e49278b7ba99f59edac0e3de49956e46e530a53c15aa71226b7aa92c6f', {
  enableMemory: true,
  disableStack: false,
  disableStorage: false,
  enableReturnData: true
});
```

Return:

```terminal
{
   "gas":25523,
   "failed":false,
   "returnValue":"",
   "structLogs":[
      {
         "pc":0,
         "op":"PUSH1",
         "gas":64580,
         "gasCost":3,
         "depth":1,
         "error":null,
         "stack":[

         ],
         "memory":null,
         "storage":{

         }
      },
      {
         "pc":2,
         "op":"PUSH1",
         "gas":64577,
         "gasCost":3,
         "depth":1,
         "error":null,
         "stack":[
            "0000000000000000000000000000000000000000000000000000000000000060"
         ],
         "memory":null,
         "storage":{

         }
      },

      ...

```

## Native tracers {#native-tracers}

The following tracers are implement in Go. This means they are much more performant than other tracers that are written in Javascript. The tracers are selected by passing their name to the `tracer` parameter when invoking a tracing API method, e.g. `debug.traceTransaction(<txhash>, { tracer: 'callTracer' })`.

### 4byteTracer {#4byte-tracer}

Solidity contract functions are
[addressed](https://docs.soliditylang.org/en/develop/abi-spec.html#function-selector) using the first four four byte of the Keccak-256 hash of their signature. Therefore when calling the function of a contract, the caller must send this function selector as well as the ABI-encoded arguments as call data.

The `4byteTracer` collects the function selectors of every function executed in the lifetime of a transaction, along with the size of the supplied call data. The result is a `map[string]int` where the keys are `SELECTOR-CALLDATASIZE` and the values are number of occurances of this key. For example:

Example call:

```sh
debug.traceTransaction( "0x214e597e35da083692f5386141e69f47e973b2c56e7a8073b1ea08fd7571e9de", {tracer: "4byteTracer"})
```

Return:

```terminal
{
  "0x27dc297e-128": 1,
  "0x38cc4831-0": 2,
  "0x524f3889-96": 1,
  "0xadf59f99-288": 1,
  "0xc281d19e-0": 1
}
```

### callTracer {#call-tracer}

The `callTracer` tracks all the call frames executed during a transaction, including depth 0. The result will be a nested list of call frames, resembling how EVM works. They form a tree with the top-level call at root and sub-calls as children of the higher levels. Each call frame has the following fields:

| field        | type        | description                          |
| ------------ | ----------- | ------------------------------------ |
| type         | string      | CALL or CREATE                       |
| from         | string      | address                              |
| to           | string      | address                              |
| value        | string      | hex-encoded amount of value transfer |
| gas          | string      | hex-encoded gas provided for call    |
| gasUsed      | string      | hex-encoded gas used during call     |
| input        | string      | call data                            |
| output       | string      | return data                          |
| error        | string      | error, if any                        |
| revertReason | string      | Solidity revert reason, if any       |
| calls        | []callframe | list of sub-calls                    |

Example Call:

```sh
> debug.traceTransaction("0x44bed3dc0f584b2a2ab32f5e2948abaaca13917eeae7ae3b959de3371a6e9a95", {tracer: 'callTracer'})
```

Return:

```terminal
{
  calls: [{
      from: "0xc8ba32cab1757528daf49033e3673fae77dcf05d",
      gas: "0x18461",
      gasUsed: "0x60",
      input: "0x000000204895cd480cc8412691a880028a25aec86786f1ed2aa5562bc400000000000000c6403c14f35be1da6f433eadbb6e9178a47fbc7c6c1d568d2f2b876e929089c8d8db646304fd001a187dc8a6",
      output: "0x557904b74478f8810cc02198544a030d1829bb491e14fe1dd0354e933c5e87bd",
      to: "0x0000000000000000000000000000000000000002",
      type: "STATICCALL"
  }, {
      from: "0xc8ba32cab1757528daf49033e3673fae77dcf05d",
      gas: "0x181db",
      gasUsed: "0x48",
      input: "0x557904b74478f8810cc02198544a030d1829bb491e14fe1dd0354e933c5e87bd",
      output: "0x5fb393023b12544491a5b8fb057943b4ebf5b1401e88e44a7800000000000000",
      to: "0x0000000000000000000000000000000000000002",
      type: "STATICCALL"
  }],
  from: "0x35a9f94af726f07b5162df7e828cc9dc8439e7d0",
  gas: "0x1a310",
  gasUsed: "0xfcb6",
  input: "0xd1a2eab2000000000000000000000000000000000000000000000000000000000024aea100000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000050000000204895cd480cc8412691a880028a25aec86786f1ed2aa5562bc400000000000000c6403c14f35be1da6f433eadbb6e9178a47fbc7c6c1d568d2f2b876e929089c8d8db646304fd001a187dc8a600000000000000000000000000000000",
  to: "0xc8ba32cab1757528daf49033e3673fae77dcf05d",
  type: "CALL",
  value: "0x0"
}
```

Things to note about the call tracer:

- Calls to precompiles are also included in the result
- In case a frame reverts, the field `output` will contain the raw return data

- In case the top level frame reverts, its `revertReason` field will contain the parsed reason of revert as returned by the Solidity contract

#### Config

`callTracer` accepts two options:

- `onlyTopCall: true` instructs the tracer to only process the main (top-level) call and none of the sub-calls. This avoids extra processing for each call frame if only the top-level call info are required.
- `withLog: true` instructs the tracer to also collect the logs emitted during each call.

Example invokation with the `onlyTopCall` flag:

```terminal
> debug.traceTransaction('0xc73e70f6d60e63a71dabf90b9983f2cdd56b0cb7bcf1a205f638d630a95bba73', { tracer: 'callTracer', tracerConfig: { onlyTopCall: true } })
```

### prestateTracer {#prestate-tracer}

The prestate tracer has two modes: `prestate` and `diff`. The `prestate` mode returns the accounts necessary to execute a given transaction. `diff` mode returns the differences between the transaction's pre and post-state (i.e. what changed because the transaction happened). The `prestateTracer` defaults to `prestate` mode. It reexecutes the given transaction and tracks every part of state that is touched. This is similar to the concept of a [stateless witness](https://ethresear.ch/t/the-stateless-client-concept/172), the difference being this tracer doesn't return any cryptographic proof, rather only the trie leaves. The result is an object. The keys are addresses of accounts. The value is an object with the following fields:

| field   | type              | description                   |
| ------- | ----------------- | ----------------------------- |
| balance | string            | balance in Wei                |
| nonce   | uint64            | nonce                         |
| code    | string            | hex-encoded bytecode          |
| storage | map[string]string | storage slots of the contract |

To run this tracer in `diff` mode, pass `tracerConfig: {diffMode: true}` in the APi call.

Example:

```js
debug.traceCall(
  {
    from: '0x35a9f94af726f07b5162df7e828cc9dc8439e7d0',
    to: '0xc8ba32cab1757528daf49033e3673fae77dcf05d',
    data: '0xd1a2eab2000000000000000000000000000000000000000000000000000000000024aea100000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000050000000204895cd480cc8412691a880028a25aec86786f1ed2aa5562bc400000000000000c6403c14f35be1da6f433eadbb6e9178a47fbc7c6c1d568d2f2b876e929089c8d8db646304fd001a187dc8a600000000000000000000000000000000'
  },
  'latest',
  { tracer: 'prestateTracer' }
);
```

Return:

```terminal
{
  0x0000000000000000000000000000000000000002: {
    balance: "0x0"
  },
  0x008b3b2f992c0e14edaa6e2c662bec549caa8df1: {
    balance: "0x2638035a26d133809"
  },
  0x35a9f94af726f07b5162df7e828cc9dc8439e7d0: {
    balance: "0x7a48734599f7284",
    nonce: 1133
  },
  0xc8ba32cab1757528daf49033e3673fae77dcf05d: {
    balance: "0x0",
    code: "0x608060405234801561001057600080fd5b50600436106100885760003560e01c8063a9c2d...
    nonce: 1,
    storage: {
      0x0000000000000000000000000000000000000000000000000000000000000000: "0x000000000000000000000000000000000000000000000000000000000024aea6",
      0x59fb7853eb21f604d010b94c123acbeae621f09ce15ee5d7616485b1e78a72e9: "0x00000000000000c42b56a52aedf18667c8ae258a0280a8912641c80c48cd9548",
      0x8d8ebb65ec00cb973d4fe086a607728fd1b9de14aa48208381eed9592f0dee9a: "0x00000000000000784ae4881e40b1f5ebb4437905fbb8a5914454123b0293b35f",
      0xff896b09014882056009dedb136458f017fcef9a4729467d0d00b4fd413fb1f1: "0x000000000000000e78ac39cb1c20e9edc753623b153705d0ccc487e31f9d6749"
    }
  }
}
```

Return (same call with `{diffMode: True}`):

```terminal
{
  post: {
    0x35a9f94af726f07b5162df7e828cc9dc8439e7d0: {
      nonce: 1135
    }
  },
  pre: {
    0x35a9f94af726f07b5162df7e828cc9dc8439e7d0: {
      balance: "0x7a48429e177130a",
      nonce: 1134
    }
  }
}
```

### noopTracer {#noop-tracer}

This tracer is noop. It returns an empty object and is only meant for testing the setup.

## Javascript tracers {#js-tracers}

There are also a set of tracers written in Javascript. These are less performant than the Go native tracers because of overheads associated with interpreting the Javascript in Geth's Go environment.

### bigram {#bigram}

`bigramTracer` counts the opcode bigrams, i.e. how many times 2 opcodes were executed one after the other.

Example:

```js
debug.traceCall(
  {
    from: '0x35a9f94af726f07b5162df7e828cc9dc8439e7d0',
    to: '0xc8ba32cab1757528daf49033e3673fae77dcf05d',
    data: '0xd1a2eab2000000000000000000000000000000000000000000000000000000000024aea100000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000050000000204895cd480cc8412691a880028a25aec86786f1ed2aa5562bc400000000000000c6403c14f35be1da6f433eadbb6e9178a47fbc7c6c1d568d2f2b876e929089c8d8db646304fd001a187dc8a600000000000000000000000000000000'
  },
  'latest',
  { tracer: 'bigramTracer' }
);
```

Returns:

```terminal
{
  ADD-ADD: 1,
  ADD-AND: 2,
  ADD-CALLDATALOAD: 1,
  ADD-DUP1: 2,
  ADD-DUP2: 2,
  ADD-GT: 1,
  ADD-MLOAD: 1,
  ADD-MSTORE: 4,
  ADD-PUSH1: 1,
  ADD-PUSH2: 4,
  ADD-SLT: 1,
  ADD-SWAP1: 10,
  ADD-SWAP2: 1,
  ADD-SWAP3: 1,
  ADD-SWAP4: 3,
  ADD-SWAP5: 1,
  AND-DUP3: 2,
  AND-ISZERO: 4,
  ...
  }

```

### evmdis {#evmdis}

`evmdisTracer` returns sufficient information from a trace to perform [evmdis](https://github.com/Arachnid/evmdis)-style disassembly

Example:

```js
> debug.traceCall({from: "0x35a9f94af726f07b5162df7e828cc9dc8439e7d0", to: "0xc8ba32cab1757528daf49033e3673fae77dcf05d", data: "0xd1a2eab2000000000000000000000000000000000000000000000000000000000024aea100000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000050000000204895cd480cc8412691a880028a25aec86786f1ed2aa5562bc400000000000000c6403c14f35be1da6f433eadbb6e9178a47fbc7c6c1d568d2f2b876e929089c8d8db646304fd001a187dc8a600000000000000000000000000000000"}, 'latest', {tracer: 'evmdisTracer'})
```

Returns:

```terminal
[{
    depth: 1,
    len: 2,
    op: 96,
    result: ["80"]
}, {
    depth: 1,
    len: 2,
    op: 96,
    result: ["40"]
}, {
    depth: 1,
    op: 82,
    result: []
}, {
    depth: 1,
    op: 52,
    result: ["0"]
}, {
    depth: 1,
    op: 128,
    result: ["0", "0"]
}, {
    depth: 1,
    op: 21,
    result: ["1"]
}, {
    depth: 1,
    len: 3,
    op: 97,
    result: ["10"]
}, {
    depth: 1,
    op: 87,
    result: []
}, {
    depth: 1,
    op: 91,
    pc: 16,
    result: []
},
...
```

### opcount {#opcount}

`opcountTracer` counts the total number of opcodes executed and simply returns the number.

Example:

```js
debug.traceCall(
  {
    from: '0x35a9f94af726f07b5162df7e828cc9dc8439e7d0',
    to: '0xc8ba32cab1757528daf49033e3673fae77dcf05d',
    data: '0xd1a2eab2000000000000000000000000000000000000000000000000000000000024aea100000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000050000000204895cd480cc8412691a880028a25aec86786f1ed2aa5562bc400000000000000c6403c14f35be1da6f433eadbb6e9178a47fbc7c6c1d568d2f2b876e929089c8d8db646304fd001a187dc8a600000000000000000000000000000000'
  },
  'latest',
  { tracer: 'opcountTracer' }
);
```

Returns:

```terminal
1384
```

### trigram {#trigram}

`trigramTracer` counts the opcode trigrams. Trigrams are the possible combinations of three opcodes this tracer reports how many times each combination is seen during execution.

Example:

```js
debug.traceCall(
  {
    from: '0x35a9f94af726f07b5162df7e828cc9dc8439e7d0',
    to: '0xc8ba32cab1757528daf49033e3673fae77dcf05d',
    data: '0xd1a2eab2000000000000000000000000000000000000000000000000000000000024aea100000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000050000000204895cd480cc8412691a880028a25aec86786f1ed2aa5562bc400000000000000c6403c14f35be1da6f433eadbb6e9178a47fbc7c6c1d568d2f2b876e929089c8d8db646304fd001a187dc8a600000000000000000000000000000000'
  },
  'latest',
  { tracer: 'trigramTracer' }
);
```

Returns:

```terminal
{
  --PUSH1: 1,
  -PUSH1-MSTORE: 1,
  ADD-ADD-GT: 1,
  ADD-AND-DUP3: 2,
  ADD-CALLDATALOAD-PUSH8: 1,
  ADD-DUP1-PUSH1: 2,
  ADD-DUP2-ADD: 1,
  ADD-DUP2-MSTORE: 1,
  ADD-GT-ISZERO: 1,
  ADD-MLOAD-DUP6: 1,
  ADD-MSTORE-ADD: 1,
  ADD-MSTORE-PUSH1: 2,
  ADD-MSTORE-PUSH32: 1,
  ADD-PUSH1-KECCAK256: 1,
  ADD-PUSH2-JUMP: 2,
  ADD-PUSH2-JUMPI: 1,
  ADD-PUSH2-SWAP2: 1,
  ADD-SLT-PUSH2: 1,
...
}
```

### unigram {#unigram}

`unigramTracer` counts the frequency of occurrance of each opcode.

Example:

```js
> debug.traceCall({from: "0x35a9f94af726f07b5162df7e828cc9dc8439e7d0", to: "0xc8ba32cab1757528daf49033e3673fae77dcf05d", data: "0xd1a2eab2000000000000000000000000000000000000000000000000000000000024aea100000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000050000000204895cd480cc8412691a880028a25aec86786f1ed2aa5562bc400000000000000c6403c14f35be1da6f433eadbb6e9178a47fbc7c6c1d568d2f2b876e929089c8d8db646304fd001a187dc8a600000000000000000000000000000000"}, 'latest', {tracer: 'unigramTracer'})
```

Returns:

```terminal
{
  ADD: 36,
  AND: 23,
  BYTE: 4,
  CALLDATACOPY: 1,
  CALLDATALOAD: 6,
  CALLDATASIZE: 2,
  CALLVALUE: 1,
  DIV: 9,
  DUP1: 29,
  DUP10: 2,
  DUP11: 1,
  DUP12: 3,
  DUP13: 2,
  ...
  }
```

## State overrides {#state-overrides}

It is possible to give temporary state modifications to Geth in order to simulate the effects of `eth_call`. For example, some new byetcode could be deployed to some address _temporarily just for the duration of the execution_ and then a transaction interacting with that address canm be traced. This can be used for scenario testing or determining the outcome of some hypothetical transaction before executing for real.

To do this, the tracer is written as normal, but the parameter `stateOverrides` is passed an address and some bytecode.

```js
var code = //contract bytecode
var tracer = //tracer name
debug.traceCall({from: , to: , input: }, 'latest', {stateOverrides: {'0x...': {code: code}}, tracer: tracer})
```

## Summary {#summary}

This page showed how to use the tracers that come bundled with Geth. There are a set written in Go and a set written in Javascript. They are invoked by passing their names when calling an API method. State overrides can be used in combination with tracers to examine precisely what the EVM will do in some hypothetical scenarios.
