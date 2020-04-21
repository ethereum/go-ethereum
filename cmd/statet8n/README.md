## `statet8n`

The `statet8n` tool is a stateless state transition utility. It is a utility which
can 
1. Take a prestate, including
 - Accounts, 
 - Block context information, 
 - Previous blockshashes (*optional)
2. Apply a set of transactions, 
3. Apply a mining-reward (*optional), 
4. And generate a post-state, including
  - State root, transaction root, receipt root, 
  - Information about rejected transactions, 
  - Optionally: a full or partial post-state dump

## Specification

The idea is to specify the behaviour of this binary very _strict_, so that other 
node implementors can build replicas based on their own state-machines, and the 
state generators can swap between a `geth`-based implementation and a `parityvm`-based 
implementation. 

### Command line params

Command line params that has to be supported are

* `--trace` (boolean) Output full trace logs to files <txhash>.jsonl
* `--trace.nostack` (boolean) Disable stack output in traces
* `--trace` (boolean) Output full trace logs to files <txhash>.jsonl
* `--trace.nomemory` (boolean) Disable full memory dump in traces
* `--trace.nostack` (boolean) Disable stack output in traces
* `--output.alloc` (`stdout` or `stderr` or a filename). Determine how to output the poststate alloc section
* `--output.result` (`stdout` or `stderr` or a filename). Determine how to output the poststate result section
* `--state.pre` <string>` (boolean) File name of where to find the pre-state json. Use '-' to read from stdin (default: "prestate.json")
* `--state.reward <int>` (boolean) Mining reward. Set to -1 to disable (default: 0)
* `--state.fork <string>` (boolean) Name of ruleset to use.

### Error codes and output

All logging should happen against the `stderr`.
There are a few (not many) errors that can occur, those are defined below. 

#### EVM-based errors (`2` to `9`)

- Other EVM error. Exit code `2`
- Failed configuration: when a non-supported or invalid fork was specified. Exit code `3`. 
- Block history is not supplied, but needed for a `BLOCKHASH` operation. If `BLOCKHASH`
  is invoked targeting a block which history has not been provided for, the program will
  exit with code `4`. 

Example:
```
./statet8n --input.alloc=./testdata/alloc.json --input.txs=./testdata/txs.json --input.env=./testdata/env.json --state.fork=Frontier+1346 2>/dev/null
ERROR(3): Failed constructing chain configuration: syntax error, invalid eip number 1346
[user@work statet8n]$ printf '%d\n' $?
3
```

#### IO errors (`10`-`20`)

- Invalid input json: the supplied data could not be marshalled. 
  The program will exit with code `10`
- IO problems: failure to load or save files, the program will exit with code `11`

Ok
## Examples
### Basic usage

Invoking it with the provided example files
```
./statet8n --input.alloc=./testdata/alloc.json --input.txs=./testdata/txs.json --input.env=./testdata/env.json
```
Two resulting files:
`alloc.json`:
```json
{
 "0x8a8eafb1cf62bfbeb1741769dae1a9dd47996192": {
  "balance": "0xfeed1a9d",
  "nonce": "0x1"
 },
 "0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b": {
  "balance": "0x5ffd4878be161d74",
  "nonce": "0xac"
 },
 "0xc94f5374fce5edbc8e2a8697c15331677e6ebf0b": {
  "balance": "0xa410"
 }
}
```
`result.json`:
```json
{
 "postState": "0x84208a19bc2b46ada7445180c1db162be5b39b9abc8c0a54b05d32943eae4e13",
 "txHash": "0xe9bd66ea8f932b2b610632074c8b2c10bd3a1de96365a31568b1c776d80779b8",
 "receiptRoot": "0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2",
 "receipts": [
  {
   "root": "0x",
   "status": "0x1",
   "cumulativeGasUsed": "0x5208",
   "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
   "logs": null,
   "transactionHash": "0x0557bacce3375c98d806609b8d5043072f0b6a8bae45ae5a67a00d3a1a18d673",
   "contractAddress": "0x0000000000000000000000000000000000000000",
   "gasUsed": "0x5208",
   "blockHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
   "transactionIndex": "0x0"
  }
 ],
 "rejected": [
  1
 ]
}
```

We can make them spit out the data to e.g. `stdout` like this:
```
./statet8n --input.alloc=./testdata/alloc.json --input.txs=./testdata/txs.json --input.env=./testdata/env.json --output.result=stdout --output.alloc=stdout
```
Output:
```json
{
 "alloc": {
  "0x8a8eafb1cf62bfbeb1741769dae1a9dd47996192": {
   "balance": "0xfeed1a9d",
   "nonce": "0x1"
  },
  "0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b": {
   "balance": "0x5ffd4878be161d74",
   "nonce": "0xac"
  },
  "0xc94f5374fce5edbc8e2a8697c15331677e6ebf0b": {
   "balance": "0xa410"
  }
 },
 "result": {
  "postState": "0x84208a19bc2b46ada7445180c1db162be5b39b9abc8c0a54b05d32943eae4e13",
  "txHash": "0xe9bd66ea8f932b2b610632074c8b2c10bd3a1de96365a31568b1c776d80779b8",
  "receiptRoot": "0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2",
  "receipts": [
   {
    "root": "0x",
    "status": "0x1",
    "cumulativeGasUsed": "0x5208",
    "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
    "logs": null,
    "transactionHash": "0x0557bacce3375c98d806609b8d5043072f0b6a8bae45ae5a67a00d3a1a18d673",
    "contractAddress": "0x0000000000000000000000000000000000000000",
    "gasUsed": "0x5208",
    "blockHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "transactionIndex": "0x0"
   }
  ],
  "rejected": [
   1
  ]
 }
}
```
### Mining reward

Adding a mining reward: 
```
./statet8n --input.alloc=./testdata/alloc.json --input.txs=./testdata/txs.json --input.env=./testdata/env.json  --output.alloc=stdout
```
Output:
```json
{
 "alloc": {
  "0x8a8eafb1cf62bfbeb1741769dae1a9dd47996192": {
   "balance": "0xfeed1a9d",
   "nonce": "0x1"
  },
  "0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b": {
   "balance": "0x5ffd4878be161d74",
   "nonce": "0xac"
  },
  "0xc94f5374fce5edbc8e2a8697c15331677e6ebf0b": {
   "balance": "0xa410"
  }
 }
}
```
### Future EIPS

It is also possible to experiment with future eips that are not yet defined in a hard fork.
Example, putting EIP-1344 into Frontier: 
```
./statet8n --state.fork=Frontier+1344 --input.pre=./testdata/pre.json --input.txs=./testdata/txs.json --input.env=/testdata/env.json
```

### Chaining

Another thing that can be done, is to chain invocations:
```
./statet8n --input.alloc=./testdata/alloc.json --input.txs=./testdata/txs.json --input.env=./testdata/env.json --output.alloc=stdout | ./statet8n --input.alloc=stdin --input.env=./testdata/env.json --input.txs=./testdata/txs.json

```
What happened here, is that we first applied two identical transactions, so the second one was rejected. 
Then, taking the poststate alloc as the input for the next state, we tried again to include
the same two transactions: this time, both failed due to too low nonce.

In order to meaningfully chain invocations, one would need to provide meaningful new `env`, otherwise the
actual blocknumber (exposed to the EVM) would not increase.



## TODO

This specification and implementation is still draft, and subject to change. Some 
implementation leftovers are: 

- Make traces use different output files, right now it stores all traces in the same file (traces.jsonl)
- Make the machine accept rlp-encoded transactions, 
- Right now, a missing `BLOCKHASH` lead to `panic`, and does not exit with the right exit code
- It's not possible to provide blockhashes yet (not even specified how)