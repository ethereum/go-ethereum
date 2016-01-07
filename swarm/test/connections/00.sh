#!/bin/bash

dir=`dirname $0`
source $dir/../../cmd/swarm/test.sh

swarm init 4
echo "expect each node to have 3 peers"
cmd="'net.peerCount'"
sleep 5
swarm attach 00 --exec "$cmd"|tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"
swarm attach 01 --exec "$cmd"|tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"
swarm attach 02 --exec "$cmd"|tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"
swarm attach 03 --exec "$cmd"|tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"

swarm stop all

echo "after static nodes is deleted, connections are recovered from kaddb in bzz-peers.json"
# echo rm -rf $DATA_ROOT/enodes\*
# echo rm -rf $DATA_ROOT/data/\*/static-nodes.json
rm -rf $DATA_ROOT/enodes*
rm -rf $DATA_ROOT/data/*/static-nodes.json

swarm cluster 4
echo "expect each node to have 3 peers"
cmd="'net.peerCount'"
sleep 10
swarm attach 00 --exec "$cmd" |tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"
swarm attach 01 --exec "$cmd"|tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"
swarm attach 02 --exec "$cmd"|tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"
swarm attach 03 --exec "$cmd"|tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"

swarm stop all

