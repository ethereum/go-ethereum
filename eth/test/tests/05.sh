#!/bin/bash

TIMEOUT=60

cat >> $JSFILE <<EOF
eth.addPeer("localhost:30311");
sleep(200);
eth.addPeer("localhost:30312");
eth.addPeer("localhost:30313");
eth.addPeer("localhost:30314");
sleep(3000);
eth.export("$CHAIN_TEST");
EOF

peer 11 01 -mine
peer 12 02 -mine
peer 13 03
peer 14 04
test_node $NAME "" -loglevel 5 $JSFILE

