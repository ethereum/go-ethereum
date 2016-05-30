#!/bin/bash

swarm init 4
echo "expect each node to have 3 peers"
cmd="net.peerCount"
sleep 5
swarm execute 00  "$cmd"|tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"
swarm execute 01  "$cmd"|tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"
swarm execute 02  "$cmd"|tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"
swarm execute 03  "$cmd"|tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"

swarm stop all

echo "connections are recovered from kaddb in bzz-peers.json"

swarm cluster 4
echo "expect each node to have 3 peers"
sleep 5
swarm execute 00  "$cmd"|tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"
swarm execute 01  "$cmd"|tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"
swarm execute 02  "$cmd"|tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"
swarm execute 03  "$cmd"|tail -n1|grep -ql 3&& echo "PASS"||echo "FAIL"

swarm stop all

