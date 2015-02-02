#!/bin/bash
# bash ./mine.sh node_id timeout(sec) [basechain]
ETH=../../ethereum
MINE="$ETH -datadir tmp/nodes/$1 -seed=false -port '' -shh=false -id test$1"
rm -rf tmp/nodes/$1
echo "Creating chain $1..."
if [[ "" !=  "$3" ]]; then
  CHAIN="chains/$3.chain"
  CHAINARG="-chain $CHAIN"
  $MINE -mine $CHAINARG -loglevel 3 | grep 'importing'
fi
$MINE -mine -loglevel 0 &
PID=$!
sleep $2
kill $PID
$MINE -loglevel 3 <(echo "eth.export(\"chains/$1.chain\")") > /tmp/eth.test/mine.tmp &
PID=$!
sleep 1
kill $PID
cat /tmp/eth.test/mine.tmp | grep 'exporting'
