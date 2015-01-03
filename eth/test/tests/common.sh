#!/bin/bash

# launched by run.sh
function peer {
  rm -rf $DIR/$1
  ARGS="-datadir $DIR/$1 -debug debug -seed=false -shh=false -id test$1"
  if [ "" != "$2" ]; then
    chain="chains/$2.chain"
    echo "import chain $chain"
    $ETH $ARGS -loglevel 5 -chain $chain
    # $ETH $ARGS -loglevel 5 -chain $chain | grep CLI |grep import
  fi
  echo "starting test node $1 with extra args ${@:3}"
  $ETH $ARGS -port 303$1 ${@:3} &
  PID=$!
  PIDS="$PIDS $PID"
}
