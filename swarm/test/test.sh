#!/bin/bash

TEST_DIR=`dirname $0`
TEST_NAME=`basename $0 .sh`
TEST_TYPE=`basename $TEST_DIR`
export IP_ADDR="[::]"


export SWARM_NETWORK_ID=322$TEST_NAME
export SWARM_DIR=~/bzz/test/$TEST_TYPE

rm -rf $SWARM_DIR/$SWARM_NETWORK_ID

wait=1



function randomfile {
  dd if=/dev/urandom of=/dev/stdout bs=1024 count=$1 2>/dev/null
}