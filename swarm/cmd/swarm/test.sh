#!/bin/bash

TEST_DIR=`dirname $0`
TEST_NAME=`basename $0 .sh`

export SWARM_BIN=$TEST_DIR/../../cmd/swarm
export GETH=$SWARM_BIN/../../../geth
export NETWORKID=322$TEST_NAME
export TMPDIR=~/BZZ/swarm-test
export DATA_ROOT=$TMPDIR/$NETWORKID
# alias swarm='bash $SWARM_BIN/swarm.sh $DATA_ROOT $NETWORKID'
EXTRA_ARGS=$*

wait=1

function swarm {
  # echo bash $SWARM_BIN/swarm.sh $TMPDIR $NETWORKID  $* $EXTRA_ARGS
  bash $SWARM_BIN/swarm.sh $TMPDIR $NETWORKID $* $EXTRA_ARGS
}


function randomfile {
  dd if=/dev/urandom of=/dev/stdout bs=1024 count=$1 2>/dev/null
}