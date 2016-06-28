#!/bin/bash
echo "TEST sync/01:"
echo " two nodes that do not have any funds"
echo " can still sync content with each other"

dir=`dirname $0`
source $dir/../test.sh
key=/tmp/key

long=/tmp/10M
randomfile 10000 > $long
ls -l $long


swarm init 2
sleep $wait
swarm up 00 $long |tail -n1 > $key
sleep $wait
swarm stop 01

swarm start 01
swarm needs 01 $key $long

swarm stop all