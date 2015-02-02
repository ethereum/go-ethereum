#!/bin/bash

# launched by run.sh
function test_node {
  rm -rf $DIR/$1
  ARGS="-datadir $DIR/$1 -debug debug -seed=false -shh=false -id test$1 -port 303$1"
  if [ "" != "$2" ]; then
    chain="chains/$2.chain"
    echo "import chain $chain"
    $ETH $ARGS -loglevel 3 -chain $chain | grep CLI |grep import
  fi
  echo "starting test node $1 with args $ARGS ${@:3}"
  $ETH $ARGS  ${@:3} &
  PID=$!
  PIDS="$PIDS $PID"
}

function peer {
  test_node $@ -loglevel 5 -logfile debug.log -maxpeer 1 -dial=false
}