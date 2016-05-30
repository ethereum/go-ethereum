#!/bin/bash
echo "TEST sync/00:"
echo " two nodes that sync (no swap and do not have any funds)"
echo " can be in sync content with each other"

dir=`dirname $0`
source $dir/../test.sh

mkdir -p /tmp/swarm-test-files
FILE_00=/tmp/swarm-test-files/00
FILE_01=/tmp/swarm-test-files/01
FILE_02=/tmp/swarm-test-files/02
FILE_03=/tmp/swarm-test-files/03
FILE_04=/tmp/swarm-test-files/04

for f in $FILE_00 $FILE_01 $FILE_02 $FILE_03 $FILE_04; do
  randomfile 20 > $f
done
# options="--verbosity=0 --vmodule=swarm/network/*=6,common/chequebook/*=6,common/swap/*=6,common/kademlia/*=5"

key=/tmp/key
swarm init 2 $options
swarm info 00
swarm info 01
swarm up 00 $FILE_00|tail -n1 > $key
swarm needs 00 $key $FILE_00
# sleep $wait
swarm needs 01 $key $FILE_00
swarm stop 01


swarm up 00 $FILE_01|tail -n1 > $key
swarm needs 00 $key $FILE_01
swarm start 01 $options
swarm needs 01 $key $FILE_01

swarm up 00 $FILE_02|tail -n1 > $key
swarm needs 00 $key $FILE_02
swarm needs 01 $key $FILE_02

swarm up 01 $FILE_03|tail -n1 > $key
swarm needs 01 $key $FILE_03
swarm needs 00 $key $FILE_03

swarm stop 00
swarm up 01 $FILE_04|tail -n1 > $key
swarm needs 01 $key $FILE_04
swarm start 00
sleep $wait
swarm needs 00 $key $FILE_04

swarm stop all



