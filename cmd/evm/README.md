## EVM state transition tool

The `evm t8n` tool is a stateless state transition utility. It is a utility
which can

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
```

   --trace                            Output full trace logs to files <txhash>.jsonl
   --trace.nomemory                   Disable full memory dump in traces
   --trace.nostack                    Disable stack output in traces
   --trace.noreturndata               Disable return data output in traces
   --output.basedir value             Specifies where output files are placed. Will be created if it does not exist.
   --output.alloc alloc               Determines where to put the alloc of the post-state.
                                      `stdout` - into the stdout output
                                      `stderr` - into the stderr output
   --output.result result             Determines where to put the result (stateroot, txroot etc) of the post-state.
                                      `stdout` - into the stdout output
                                      `stderr` - into the stderr output
   --state.fork value                 Name of ruleset to use.
   --state.chainid value              ChainID to use (default: 1)
   --state.reward value               Mining reward. Set to -1 to disable (default: 0)

```

### Error codes and output

All logging should happen against the `stderr`.
There are a few (not many) errors that can occur, those are defined below.

#### EVM-based errors (`2` to `9`)

- Other EVM error. Exit code `2`
- Failed configuration: when a non-supported or invalid fork was specified. Exit code `3`.
- Block history is not supplied, but needed for a `BLOCKHASH` operation. If `BLOCKHASH`
  is invoked targeting a block which history has not been provided for, the program will
  exit with code `4`.

#### IO errors (`10`-`20`)

- Invalid input json: the supplied data could not be marshalled.
  The program will exit with code `10`
- IO problems: failure to load or save files, the program will exit with code `11`

## Examples
### Basic usage

Invoking it with the provided example files
```
./evm t8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json
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
 "stateRoot": "0x84208a19bc2b46ada7445180c1db162be5b39b9abc8c0a54b05d32943eae4e13",
 "txRoot": "0xc4761fd7b87ff2364c7c60b6c5c8d02e522e815328aaea3f20e3b7b7ef52c42d",
 "receiptRoot": "0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2",
 "logsHash": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
 "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
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
./evm t8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json --output.result=stdout --output.alloc=stdout
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
  "stateRoot": "0x84208a19bc2b46ada7445180c1db162be5b39b9abc8c0a54b05d32943eae4e13",
  "txRoot": "0xc4761fd7b87ff2364c7c60b6c5c8d02e522e815328aaea3f20e3b7b7ef52c42d",
  "receiptRoot": "0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2",
  "logsHash": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
  "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
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

## About Ommers

Mining rewards and ommer rewards might need to be added. This is how those are applied:

- `block_reward` is the block mining reward for the miner (`0xaa`), of a block at height `N`.
- For each ommer (mined by `0xbb`), with blocknumber `N-delta`
   - (where `delta` is the difference between the current block and the ommer)
   - The account `0xbb` (ommer miner) is awarded `(8-delta)/ 8 * block_reward`
   - The account `0xaa` (block miner) is awarded `block_reward / 32`

To make `state_t8n` apply these, the following inputs are required:

- `state.reward`
  - For ethash, it is `5000000000000000000` `wei`,
  - If this is not defined, mining rewards are not applied,
  - A value of `0` is valid, and causes accounts to be 'touched'.
- For each ommer, the tool needs to be given an `address` and a `delta`. This
  is done via the `env`.

Note: the tool does not verify that e.g. the normal uncle rules apply,
and allows e.g two uncles at the same height, or the uncle-distance. This means that
the tool allows for negative uncle reward (distance > 8)

Example:
`./testdata/5/env.json`:
```json
{
  "currentCoinbase": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "currentDifficulty": "0x20000",
  "currentGasLimit": "0x750a163df65e8a",
  "currentNumber": "1",
  "currentTimestamp": "1000",
  "ommers": [
    {"delta":  1, "address": "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" },
    {"delta":  2, "address": "0xcccccccccccccccccccccccccccccccccccccccc" }
  ]
}
```
When applying this, using a reward of `0x08`
Output:
```json
{
 "alloc": {
  "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa": {
   "balance": "0x88"
  },
  "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb": {
   "balance": "0x70"
  },
  "0xcccccccccccccccccccccccccccccccccccccccc": {
   "balance": "0x60"
  }
 }
}
```
### Future EIPS

It is also possible to experiment with future eips that are not yet defined in a hard fork.
Example, putting EIP-1344 into Frontier: 
```
./evm t8n --state.fork=Frontier+1344 --input.pre=./testdata/1/pre.json --input.txs=./testdata/1/txs.json --input.env=/testdata/1/env.json
```

### Block history

The `BLOCKHASH` opcode requires blockhashes to be provided by the caller, inside the `env`.
If a required blockhash is not provided, the exit code should be `4`:
Example where blockhashes are provided: 
```
./evm t8n --input.alloc=./testdata/3/alloc.json --input.txs=./testdata/3/txs.json --input.env=./testdata/3/env.json --trace
```
```
cat trace-0-0x72fadbef39cd251a437eea619cfeda752271a5faaaa2147df012e112159ffb81.jsonl | grep BLOCKHASH -C2
```
```
{"pc":0,"op":96,"gas":"0x5f58ef8","gasCost":"0x3","memory":"0x","memSize":0,"stack":[],"returnStack":[],"returnData":"0x","depth":1,"refund":0,"opName":"PUSH1","error":""}
{"pc":2,"op":64,"gas":"0x5f58ef5","gasCost":"0x14","memory":"0x","memSize":0,"stack":["0x1"],"returnStack":[],"returnData":"0x","depth":1,"refund":0,"opName":"BLOCKHASH","error":""}
{"pc":3,"op":0,"gas":"0x5f58ee1","gasCost":"0x0","memory":"0x","memSize":0,"stack":["0xdac58aa524e50956d0c0bae7f3f8bb9d35381365d07804dd5b48a5a297c06af4"],"returnStack":[],"returnData":"0x","depth":1,"refund":0,"opName":"STOP","error":""}
{"output":"","gasUsed":"0x17","time":142709}
```

In this example, the caller has not provided the required blockhash:
```
./evm t8n --input.alloc=./testdata/4/alloc.json --input.txs=./testdata/4/txs.json --input.env=./testdata/4/env.json --trace
```
```
ERROR(4): getHash(3) invoked, blockhash for that block not provided
```
Error code: 4
### Chaining

Another thing that can be done, is to chain invocations:
```
./evm t8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json --output.alloc=stdout | ./evm t8n --input.alloc=stdin --input.env=./testdata/1/env.json --input.txs=./testdata/1/txs.json
INFO [01-21|22:41:22.963] rejected tx                              index=1 hash=0557ba..18d673 from=0x8A8eAFb1cf62BfBeb1741769DAE1a9dd47996192 error="nonce too low: address 0x8A8eAFb1cf62BfBeb1741769DAE1a9dd47996192, tx: 0 state: 1"
INFO [01-21|22:41:22.966] rejected tx                              index=0 hash=0557ba..18d673 from=0x8A8eAFb1cf62BfBeb1741769DAE1a9dd47996192 error="nonce too low: address 0x8A8eAFb1cf62BfBeb1741769DAE1a9dd47996192, tx: 0 state: 1"
INFO [01-21|22:41:22.967] rejected tx                              index=1 hash=0557ba..18d673 from=0x8A8eAFb1cf62BfBeb1741769DAE1a9dd47996192 error="nonce too low: address 0x8A8eAFb1cf62BfBeb1741769DAE1a9dd47996192, tx: 0 state: 1"

```
What happened here, is that we first applied two identical transactions, so the second one was rejected. 
Then, taking the poststate alloc as the input for the next state, we tried again to include
the same two transactions: this time, both failed due to too low nonce.

In order to meaningfully chain invocations, one would need to provide meaningful new `env`, otherwise the
actual blocknumber (exposed to the EVM) would not increase.
