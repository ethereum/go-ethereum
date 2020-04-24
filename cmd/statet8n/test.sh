#!/bin/bash
ticks="\`\`\`"

function showjson(){
  echo "\`$1\`:"
  echo "${ticks}json"
  cat $1
  echo ""
  echo "$ticks"
}


# This should exit with 3
./statet8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json --state.fork=Frontier+1346 2>/dev/null
if [ $? !=  3 ]; then
	echo "Failed, exitcode should be 3"
else
	echo "Ok"
fi
echo "## Examples"
echo "### Basic usage"
echo ""
echo "Invoking it with the provided example files"
cmd="./statet8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json"
echo "$ticks"
echo "$cmd"
echo "$ticks"
$cmd 2>/dev/null
echo "Two resulting files:"
showjson alloc.json
showjson result.json
echo ""

echo "We can make them spit out the data to e.g. \`stdout\` like this:"
cmd="./statet8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json --output.result=stdout --output.alloc=stdout"
echo "$ticks"
echo "$cmd"
echo "$ticks"
output=`$cmd 2>/dev/null`
echo "Output:"
echo "${ticks}json"
echo "$output"
echo "$ticks"

echo "### Mining reward"
echo ""
echo "Adding a mining reward: "
cmd="./statet8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json  --output.alloc=stdout"
echo "$ticks"
echo "$cmd"
echo "$ticks"
output=`$cmd 2>/dev/null`
echo "Output:"
echo "${ticks}json"
echo "$output"
echo "$ticks"

echo "### Future EIPS"
echo ""
echo "It is also possible to experiment with future eips that are not yet defined in a hard fork."
echo "Example, putting EIP-1344 into Frontier: "
cmd="./statet8n --state.fork=Frontier+1344 --input.pre=./testdata/1/pre.json --input.txs=./testdata/1/txs.json --input.env=/testdata/1/env.json"
echo "$ticks"
echo "$cmd"
echo "$ticks"
echo ""
echo "### Chaining"
echo ""
echo "Another thing that can be done, is to chain invocations:"
cmd1="./statet8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json --output.alloc=stdout"
cmd2="./statet8n --input.alloc=stdin --input.env=./testdata/1/env.json --input.txs=./testdata/1/txs.json"
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