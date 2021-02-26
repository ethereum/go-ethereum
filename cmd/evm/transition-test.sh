#!/bin/bash
ticks="\`\`\`"

function showjson(){
  echo "\`$1\`:"
  echo "${ticks}json"
  cat $1
  echo ""
  echo "$ticks"
}
function demo(){
  echo "$ticks"
  echo "$1"
  echo "$ticks"
  echo ""
}
function tick(){
  echo "$ticks"
}

cat << EOF
## EVM state transition tool

The \`evm t8n\` tool is a stateless state transition utility. It is a utility
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
state generators can swap between a \`geth\`-based implementation and a \`parityvm\`-based
implementation.

### Command line params

Command line params that has to be supported are
$(tick)

` ./evm t8n -h | grep "trace\|output\|state\."`

$(tick)

### Error codes and output

All logging should happen against the \`stderr\`.
There are a few (not many) errors that can occur, those are defined below.

#### EVM-based errors (\`2\` to \`9\`)

- Other EVM error. Exit code \`2\`
- Failed configuration: when a non-supported or invalid fork was specified. Exit code \`3\`.
- Block history is not supplied, but needed for a \`BLOCKHASH\` operation. If \`BLOCKHASH\`
  is invoked targeting a block which history has not been provided for, the program will
  exit with code \`4\`.

#### IO errors (\`10\`-\`20\`)

- Invalid input json: the supplied data could not be marshalled.
  The program will exit with code \`10\`
- IO problems: failure to load or save files, the program will exit with code \`11\`

EOF

# This should exit with 3
./evm t8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json --state.fork=Frontier+1346 2>/dev/null
if [ $? !=  3 ]; then
	echo "Failed, exitcode should be 3"
fi
cat << EOF
## Examples
### Basic usage

Invoking it with the provided example files
EOF
cmd="./evm t8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json"
tick;echo "$cmd"; tick
$cmd 2>/dev/null
echo "Two resulting files:"
echo ""
showjson alloc.json
showjson result.json
echo ""

echo "We can make them spit out the data to e.g. \`stdout\` like this:"
cmd="./evm t8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json --output.result=stdout --output.alloc=stdout"
tick;echo "$cmd"; tick
output=`$cmd 2>/dev/null`
echo "Output:"
echo "${ticks}json"
echo "$output"
echo "$ticks"

cat << EOF

## About Ommers

Mining rewards and ommer rewards might need to be added. This is how those are applied:

- \`block_reward\` is the block mining reward for the miner (\`0xaa\`), of a block at height \`N\`.
- For each ommer (mined by \`0xbb\`), with blocknumber \`N-delta\`
   - (where \`delta\` is the difference between the current block and the ommer)
   - The account \`0xbb\` (ommer miner) is awarded \`(8-delta)/ 8 * block_reward\`
   - The account \`0xaa\` (block miner) is awarded \`block_reward / 32\`

To make \`state_t8n\` apply these, the following inputs are required:

- \`state.reward\`
  - For ethash, it is \`5000000000000000000\` \`wei\`,
  - If this is not defined, mining rewards are not applied,
  - A value of \`0\` is valid, and causes accounts to be 'touched'.
- For each ommer, the tool needs to be given an \`address\` and a \`delta\`. This
  is done via the \`env\`.

Note: the tool does not verify that e.g. the normal uncle rules apply,
and allows e.g two uncles at the same height, or the uncle-distance. This means that
the tool allows for negative uncle reward (distance > 8)

Example:
EOF

showjson ./testdata/5/env.json

echo "When applying this, using a reward of \`0x08\`"
cmd="./evm t8n --input.alloc=./testdata/5/alloc.json -input.txs=./testdata/5/txs.json --input.env=./testdata/5/env.json  --output.alloc=stdout --state.reward=0x80"
output=`$cmd 2>/dev/null`
echo "Output:"
echo "${ticks}json"
echo "$output"
echo "$ticks"

echo "### Future EIPS"
echo ""
echo "It is also possible to experiment with future eips that are not yet defined in a hard fork."
echo "Example, putting EIP-1344 into Frontier: "
cmd="./evm t8n --state.fork=Frontier+1344 --input.pre=./testdata/1/pre.json --input.txs=./testdata/1/txs.json --input.env=/testdata/1/env.json"
tick;echo "$cmd"; tick
echo ""

echo "### Block history"
echo ""
echo "The \`BLOCKHASH\` opcode requires blockhashes to be provided by the caller, inside the \`env\`."
echo "If a required blockhash is not provided, the exit code should be \`4\`:"
echo "Example where blockhashes are provided: "
cmd="./evm t8n --input.alloc=./testdata/3/alloc.json --input.txs=./testdata/3/txs.json --input.env=./testdata/3/env.json  --trace"
tick && echo $cmd && tick
$cmd 2>&1 >/dev/null
cmd="cat trace-0-0x72fadbef39cd251a437eea619cfeda752271a5faaaa2147df012e112159ffb81.jsonl | grep BLOCKHASH -C2"
tick && echo $cmd && tick
echo "$ticks"
cat trace-0-0x72fadbef39cd251a437eea619cfeda752271a5faaaa2147df012e112159ffb81.jsonl | grep BLOCKHASH -C2
echo "$ticks"
echo ""

echo "In this example, the caller has not provided the required blockhash:"
cmd="./evm t8n --input.alloc=./testdata/4/alloc.json --input.txs=./testdata/4/txs.json --input.env=./testdata/4/env.json  --trace"
tick && echo $cmd && tick
tick
$cmd
errc=$?
tick
echo "Error code: $errc"


echo "### Chaining"
echo ""
echo "Another thing that can be done, is to chain invocations:"
cmd1="./evm t8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json --output.alloc=stdout"
cmd2="./evm t8n --input.alloc=stdin --input.env=./testdata/1/env.json --input.txs=./testdata/1/txs.json"
echo "$ticks"
echo "$cmd1 | $cmd2"
output=$($cmd1 | $cmd2 )
echo $output
echo "$ticks"
echo "What happened here, is that we first applied two identical transactions, so the second one was rejected. "
echo "Then, taking the poststate alloc as the input for the next state, we tried again to include"
echo "the same two transactions: this time, both failed due to too low nonce."
echo ""
echo "In order to meaningfully chain invocations, one would need to provide meaningful new \`env\`, otherwise the"
echo "actual blocknumber (exposed to the EVM) would not increase."
echo ""
