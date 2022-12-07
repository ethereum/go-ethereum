# EVM tool

The EVM tool provides a few useful subcommands to facilitate testing at the EVM
layer.

* transition tool    (`t8n`) : a stateless state transition utility
* transaction tool   (`t9n`) : a transaction validation utility
* block builder tool (`b11r`): a block assembler utility

## State transition tool (`t8n`)


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

### Specification

The idea is to specify the behaviour of this binary very _strict_, so that other
node implementors can build replicas based on their own state-machines, and the
state generators can swap between a \`geth\`-based implementation and a \`parityvm\`-based
implementation.

#### Command line params

Command line params that need to be supported are

```
    --input.alloc value            (default: "alloc.json")
    --input.env value              (default: "env.json")
    --input.txs value              (default: "txs.json")
    --output.alloc value           (default: "alloc.json")
    --output.basedir value        
    --output.body value           
    --output.result value          (default: "result.json")
    --state.chainid value          (default: 1)
    --state.fork value             (default: "GrayGlacier")
    --state.reward value           (default: 0)
    --trace.memory                 (default: false)
    --trace.nomemory               (default: true)
    --trace.noreturndata           (default: true)
    --trace.nostack                (default: false)
    --trace.returndata             (default: false)
```
#### Objects

The transition tool uses JSON objects to read and write data related to the transition operation. The
following object definitions are required.

##### `alloc`

The `alloc` object defines the prestate that transition will begin with.

```go
// Map of address to account definition.
type Alloc map[common.Address]Account
// Genesis account. Each field is optional.
type Account struct {
    Code       []byte                           `json:"code"`
    Storage    map[common.Hash]common.Hash      `json:"storage"`
    Balance    *big.Int                         `json:"balance"`
    Nonce      uint64                           `json:"nonce"`
    SecretKey  []byte                            `json:"secretKey"`
}
```

##### `env`

The `env` object defines the environmental context in which the transition will
take place.

```go
type Env struct {
    // required
    CurrentCoinbase  common.Address      `json:"currentCoinbase"`
    CurrentGasLimit  uint64              `json:"currentGasLimit"`
    CurrentNumber    uint64              `json:"currentNumber"`
    CurrentTimestamp uint64              `json:"currentTimestamp"`
    Withdrawals      []*Withdrawal       `json:"withdrawals"`
    // optional
    CurrentDifficulty *big.Int           `json:"currentDifficuly"`
    CurrentRandom     *big.Int           `json:"currentRandom"`
    CurrentBaseFee    *big.Int           `json:"currentBaseFee"`
    ParentDifficulty  *big.Int           `json:"parentDifficulty"`
    ParentGasUsed     uint64             `json:"parentGasUsed"`
    ParentGasLimit    uint64             `json:"parentGasLimit"`
    ParentTimestamp   uint64             `json:"parentTimestamp"`
    BlockHashes       map[uint64]common.Hash `json:"blockHashes"`
    ParentUncleHash   common.Hash        `json:"parentUncleHash"`
    Ommers            []Ommer            `json:"ommers"`
}
type Ommer struct {
    Delta   uint64         `json:"delta"`
    Address common.Address `json:"address"`
}
type Withdrawal struct {
    Index          uint64         `json:"index"`
    ValidatorIndex uint64         `json:"validatorIndex"`
    Recipient      common.Address `json:"recipient"`
    Amount         *big.Int       `json:"amount"`
}
```

##### `txs`

The `txs` object is an array of any of the transaction types: `LegacyTx`,
`AccessListTx`, or `DynamicFeeTx`.

```go
type LegacyTx struct {
	Nonce     uint64          `json:"nonce"`
	GasPrice  *big.Int        `json:"gasPrice"`
	Gas       uint64          `json:"gas"`
	To        *common.Address `json:"to"`
	Value     *big.Int        `json:"value"`
	Data      []byte          `json:"data"`
	V         *big.Int        `json:"v"`
	R         *big.Int        `json:"r"`
	S         *big.Int        `json:"s"`
    SecretKey *common.Hash    `json:"secretKey"`
}
type AccessList []AccessTuple
type AccessTuple struct {
	Address     common.Address `json:"address"        gencodec:"required"`
	StorageKeys []common.Hash  `json:"storageKeys"    gencodec:"required"`
}
type AccessListTx struct {
	ChainID    *big.Int        `json:"chainId"`
	Nonce      uint64          `json:"nonce"`
	GasPrice   *big.Int        `json:"gasPrice"`
	Gas        uint64          `json:"gas"`
	To         *common.Address `json:"to"`
	Value      *big.Int        `json:"value"`
	Data       []byte          `json:"data"`
	AccessList AccessList      `json:"accessList"`
	V          *big.Int        `json:"v"`
	R          *big.Int        `json:"r"`
	S          *big.Int        `json:"s"`
    SecretKey  *common.Hash     `json:"secretKey"`
}
type DynamicFeeTx struct {
	ChainID    *big.Int        `json:"chainId"`
	Nonce      uint64          `json:"nonce"`
	GasTipCap  *big.Int        `json:"maxPriorityFeePerGas"`
	GasFeeCap  *big.Int        `json:"maxFeePerGas"`
	Gas        uint64          `json:"gas"`
	To         *common.Address `json:"to"`
	Value      *big.Int        `json:"value"`
	Data       []byte          `json:"data"`
	AccessList AccessList      `json:"accessList"`
	V          *big.Int        `json:"v"`
	R          *big.Int        `json:"r"`
	S          *big.Int        `json:"s"`
    SecretKey  *common.Hash     `json:"secretKey"`
}
```

##### `result`

The `result` object is output after a transition is executed. It includes
information about the post-transition environment.

```go
type ExecutionResult struct {
    StateRoot   common.Hash    `json:"stateRoot"`
    TxRoot      common.Hash    `json:"txRoot"`
    ReceiptRoot common.Hash    `json:"receiptsRoot"`
    LogsHash    common.Hash    `json:"logsHash"`
    Bloom       types.Bloom    `json:"logsBloom"`
    Receipts    types.Receipts `json:"receipts"`
    Rejected    []*rejectedTx  `json:"rejected,omitempty"`
    Difficulty  *big.Int       `json:"currentDifficulty"`
    GasUsed     uint64         `json:"gasUsed"`
    BaseFee     *big.Int       `json:"currentBaseFee,omitempty"`
}
```

#### Error codes and output

All logging should happen against the `stderr`.
There are a few (not many) errors that can occur, those are defined below.

##### EVM-based errors (`2` to `9`)

- Other EVM error. Exit code `2`
- Failed configuration: when a non-supported or invalid fork was specified. Exit code `3`.
- Block history is not supplied, but needed for a `BLOCKHASH` operation. If `BLOCKHASH`
  is invoked targeting a block which history has not been provided for, the program will
  exit with code `4`.

##### IO errors (`10`-`20`)

- Invalid input json: the supplied data could not be marshalled.
  The program will exit with code `10`
- IO problems: failure to load or save files, the program will exit with code `11`

```
# This should exit with 3
./evm t8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json --state.fork=Frontier+1346 2>/dev/null
exitcode:3 OK
```
#### Forks
### Basic usage

The chain configuration to be used for a transition is specified via the
`--state.fork` CLI flag. A list of possible values and configurations can be
found in [`tests/init.go`](tests/init.go).

#### Examples
##### Basic usage

Invoking it with the provided example files
```
./evm t8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json --state.fork=Berlin
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
 "receiptsRoot": "0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2",
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
  {
   "index": 1,
   "error": "nonce too low: address 0x8A8eAFb1cf62BfBeb1741769DAE1a9dd47996192, tx: 0 state: 1"
  }
 ],
 "currentDifficulty": "0x20000",
 "gasUsed": "0x5208"
}
```

We can make them spit out the data to e.g. `stdout` like this:
```
./evm t8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json --output.result=stdout --output.alloc=stdout --state.fork=Berlin
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
    "receiptsRoot": "0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2",
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
      {
        "index": 1,
        "error": "nonce too low: address 0x8A8eAFb1cf62BfBeb1741769DAE1a9dd47996192, tx: 0 state: 1"
      }
    ],
    "currentDifficulty": "0x20000",
    "gasUsed": "0x5208"
  }
}
```

#### About Ommers

Mining rewards and ommer rewards might need to be added. This is how those are applied:

- `block_reward` is the block mining reward for the miner (`0xaa`), of a block at height `N`.
- For each ommer (mined by `0xbb`), with blocknumber `N-delta`
   - (where `delta` is the difference between the current block and the ommer)
   - The account `0xbb` (ommer miner) is awarded `(8-delta)/ 8 * block_reward`
   - The account `0xaa` (block miner) is awarded `block_reward / 32`

To make `t8n` apply these, the following inputs are required:

- `--state.reward`
  - For ethash, it is `5000000000000000000` `wei`,
  - If this is not defined, mining rewards are not applied,
  - A value of `0` is valid, and causes accounts to be 'touched'.
- For each ommer, the tool needs to be given an `addres\` and a `delta`. This
  is done via the `ommers` field in `env`.

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
#### Future EIPS

It is also possible to experiment with future eips that are not yet defined in a hard fork.
Example, putting EIP-1344 into Frontier: 
```
./evm t8n --state.fork=Frontier+1344 --input.pre=./testdata/1/pre.json --input.txs=./testdata/1/txs.json --input.env=/testdata/1/env.json
```

#### Block history

The `BLOCKHASH` opcode requires blockhashes to be provided by the caller, inside the `env`.
If a required blockhash is not provided, the exit code should be `4`:
Example where blockhashes are provided: 
```
./evm t8n --input.alloc=./testdata/3/alloc.json --input.txs=./testdata/3/txs.json --input.env=./testdata/3/env.json  --trace --state.fork=Berlin

```

```
cat trace-0-0x72fadbef39cd251a437eea619cfeda752271a5faaaa2147df012e112159ffb81.jsonl | grep BLOCKHASH -C2
```
```
{"pc":0,"op":96,"gas":"0x5f58ef8","gasCost":"0x3","memSize":0,"stack":[],"depth":1,"refund":0,"opName":"PUSH1"}
{"pc":2,"op":64,"gas":"0x5f58ef5","gasCost":"0x14","memSize":0,"stack":["0x1"],"depth":1,"refund":0,"opName":"BLOCKHASH"}
{"pc":3,"op":0,"gas":"0x5f58ee1","gasCost":"0x0","memSize":0,"stack":["0xdac58aa524e50956d0c0bae7f3f8bb9d35381365d07804dd5b48a5a297c06af4"],"depth":1,"refund":0,"opName":"STOP"}
{"output":"","gasUsed":"0x17"}
```

In this example, the caller has not provided the required blockhash:
```
./evm t8n --input.alloc=./testdata/4/alloc.json --input.txs=./testdata/4/txs.json --input.env=./testdata/4/env.json --trace --state.fork=Berlin
ERROR(4): getHash(3) invoked, blockhash for that block not provided
```
Error code: 4

#### Chaining

Another thing that can be done, is to chain invocations:
```
./evm t8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json --state.fork=Berlin --output.alloc=stdout | ./evm t8n --input.alloc=stdin --input.env=./testdata/1/env.json --input.txs=./testdata/1/txs.json --state.fork=Berlin

```
What happened here, is that we first applied two identical transactions, so the second one was rejected. 
Then, taking the poststate alloc as the input for the next state, we tried again to include
the same two transactions: this time, both failed due to too low nonce.

In order to meaningfully chain invocations, one would need to provide meaningful new `env`, otherwise the
actual blocknumber (exposed to the EVM) would not increase.

#### Transactions in RLP form

It is possible to provide already-signed transactions as input to, using an `input.txs` which ends with the `rlp` suffix.
The input format for RLP-form transactions is _identical_ to the _output_ format for block bodies. Therefore, it's fully possible
to use the evm to go from `json` input to `rlp` input.

The following command takes **json** the transactions in `./testdata/13/txs.json` and signs them. After execution, they are output to `signed_txs.rlp`.:
```
./evm t8n --state.fork=London --input.alloc=./testdata/13/alloc.json --input.txs=./testdata/13/txs.json --input.env=./testdata/13/env.json --output.result=alloc_jsontx.json --output.body=signed_txs.rlp
INFO [12-07|04:30:12.380] Trie dumping started                     root=e4b924..6aef61
INFO [12-07|04:30:12.380] Trie dumping complete                    accounts=3 elapsed="85.765µs"
INFO [12-07|04:30:12.380] Wrote file                               file=alloc.json
INFO [12-07|04:30:12.380] Wrote file                               file=alloc_jsontx.json
INFO [12-07|04:30:12.380] Wrote file                               file=signed_txs.rlp
```

The `output.body` is the rlp-list of transactions, encoded in hex and placed in a string a'la `json` encoding rules:
```
cat signed_txs.rlp
"0xf8d2b86702f864010180820fa08284d09411111111111111111111111111111111111111118080c001a0b7dfab36232379bb3d1497a4f91c1966b1f932eae3ade107bf5d723b9cb474e0a06261c359a10f2132f126d250485b90cf20f30340801244a08ef6142ab33d1904b86702f864010280820fa08284d09411111111111111111111111111111111111111118080c080a0d4ec563b6568cd42d998fc4134b36933c6568d01533b5adf08769270243c6c7fa072bf7c21eac6bbeae5143371eef26d5e279637f3bd73482b55979d76d935b1e9"
```

We can use `rlpdump` to check what the contents are: 
```
rlpdump -hex $(cat signed_txs.rlp | jq -r )
[
  02f864010180820fa08284d09411111111111111111111111111111111111111118080c001a0b7dfab36232379bb3d1497a4f91c1966b1f932eae3ade107bf5d723b9cb474e0a06261c359a10f2132f126d250485b90cf20f30340801244a08ef6142ab33d1904,
  02f864010280820fa08284d09411111111111111111111111111111111111111118080c080a0d4ec563b6568cd42d998fc4134b36933c6568d01533b5adf08769270243c6c7fa072bf7c21eac6bbeae5143371eef26d5e279637f3bd73482b55979d76d935b1e9,
]
```
Now, we can now use those (or any other already signed transactions), as input, like so: 
```
./evm t8n --state.fork=London --input.alloc=./testdata/13/alloc.json --input.txs=./signed_txs.rlp --input.env=./testdata/13/env.json --output.result=alloc_rlptx.json
INFO [12-07|04:30:12.425] Trie dumping started                     root=e4b924..6aef61
INFO [12-07|04:30:12.425] Trie dumping complete                    accounts=3 elapsed="70.684µs"
INFO [12-07|04:30:12.425] Wrote file                               file=alloc.json
INFO [12-07|04:30:12.425] Wrote file                               file=alloc_rlptx.json
```
You might have noticed that the results from these two invocations were stored in two separate files. 
And we can now finally check that they match.
```
cat alloc_jsontx.json | jq .stateRoot && cat alloc_rlptx.json | jq .stateRoot
"0xe4b924a6adb5959fccf769d5b7bb2f6359e26d1e76a2443c5a91a36d826aef61"
"0xe4b924a6adb5959fccf769d5b7bb2f6359e26d1e76a2443c5a91a36d826aef61"
```
