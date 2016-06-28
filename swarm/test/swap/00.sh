#!/bin/bash
echo "TEST swap/00:"
echo " two nodes that do not sync and do not have any funds"
echo " cannot retrieve content from each other"

dir=`dirname $0`
source $dir/../test.sh

FILE_00=/tmp/1K.0
randomfile 1 > $FILE_00
ls -l $FILE_00
mininginterval=50
key=/tmp/key

swarm init 2 --bzznosync
sleep $wait
swarm up 00 $FILE_00|tail -n1 > $key
swarm needs 00 $key $FILE_00
echo -n "node 01 cannot download file 0: "
swarm needs 01 $key $FILE_00 | tail -1| grep -ql "PASS" && echo "FAIL" || echo "PASS"

FILE_01=/tmp/1K.1
randomfile 1 > $FILE_01
swarm up 01 $FILE_01|tail -1 > $key
swarm needs 01 $key $FILE_01
echo -n "node 00 cannot download file 1: "
swarm needs 00 $key $FILE_01 | tail -1| grep -ql "PASS" && echo "FAIL" || echo "PASS"

swarm stop all