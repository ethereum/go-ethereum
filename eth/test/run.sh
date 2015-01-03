#!/bin/bash
# bash run.sh (testid0 testid1 ...)
# runs tests tests/testid0.sh tests/testid1.sh ...
# without arguments, it runs all tests

if [ "$#" -eq 0 ]; then
  for file in tests/*.sh; do
    i=`basename $file .sh`
    TESTS="$TESTS $i"
  done
else
  TESTS=$@
fi

ETH=../../ethereum
DIR="/tmp/eth.test/nodes"
TIMEOUT=10

mkdir -p $DIR
mkdir -p $DIR/js

echo "running tests $TESTS"
for i in $TESTS; do
  PIDS=
  CHAIN="tests/$i.chain"
  JSFILE="$DIR/$i/js"
  CHAIN_TEST="$DIR/$i/chain"

  # OUT="$DIR/out"
  echo "RUN: test $i"
  . tests/$i.sh
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


