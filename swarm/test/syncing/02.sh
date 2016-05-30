#!/bin/bash

echo "TEST sync/02:"
echo " two nodes that sync (no swap and do not have any funds)"
echo " can sync content with each other even with intermittent network connection"

dir=`dirname $0`
source $dir/../test.sh

long=/tmp/10M
key=/tmp/key
randomfile 100000 > $long
ls -l $long

swarm init 2 --vmodule='swarm/*=5'
swarm up 00 $long |tail -n1 > $key &
sleep 1
swarm execute 01 'bzz.blockNetworkRead(true)'
sleep 3
swarm execute 01 'bzz.blockNetworkRead(false)'
# sleep $wait
# swarm attach 01 -exec "'bzz.blockNetworkRead(true)'"
# sleep $wait
swarm stop 01

# swarm start 01
# sleep $wait
# swarm needs 01 $key $long
# sleep 3
swarm stop all