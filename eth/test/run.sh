#!/bin/bash
# bash run.sh (testid0 testid1 ...)
# runs tests tests/testid0.sh tests/testid1.sh ...
# without arguments, it runs all tests

. tests/common.sh

TESTS=

if [ "$#" -eq 0 ]; then
  for NAME in tests/??.sh; do
    i=`basename $NAME .sh`
    TESTS="$TESTS $i"
  done
else
  TESTS=$@
fi

ETH=../../ethereum
DIR="/tmp/eth.test/nodes"
TIMEOUT=10

mkdir -p $DIR/js

echo "running tests $TESTS"
for NAME in $TESTS; do
  PIDS=
  CHAIN="tests/$NAME.chain"
  JSFILE="$DIR/js/$NAME.js"
  CHAIN_TEST="$DIR/$NAME/chain"

  echo "RUN: test $NAME"
  cat tests/common.js > $JSFILE
  . tests/$NAME.sh
  sleep $TIMEOUT
  echo "timeout after $TIMEOUT seconds: killing $PIDS"
  kill $PIDS
  if [ -r "$CHAIN" ]; then
    if diff $CHAIN $CHAIN_TEST >/dev/null ; then
      echo "chain ok: $CHAIN=$CHAIN_TEST"
    else
      echo "FAIL: chains differ: expected $CHAIN ; got $CHAIN_TEST"
      continue
    fi
  fi
  ERRORS=$DIR/errors
  if [ -r "$ERRORS" ]; then
    echo "FAIL: "
    cat $ERRORS
  else
    echo PASS
  fi
done