#!/bin/bash
# bash ./mine.sh node_id timeout(sec) [basechain]
ETH="../ethereum -datadir tmp/nodes/$1 -seed=false -port '' -id test$1"
rm -rf tmp/nodes/$1
if [[ "" !=  "$3" ]]; then
  CHAIN="chains/$3.chain"
  CHAINARG="-chain $CHAIN"
  echo "import chain '$CHAIN'"
  $ETH -mine $CHAINARG
fi
$ETH -mine &
PID=$!
sleep $2
echo "killing $PID"
kill $PID
$ETH <(echo "eth.export(\"chains/$1.chain\")") &
PID=$!
sleep 1
echo "killing $PID"
kill $PID
