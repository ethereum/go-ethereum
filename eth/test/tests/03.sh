#!/bin/bash

TIMEOUT=12

cat >> $JSFILE <<EOF
eth.addPeer("localhost:30311");
sleep(10000);
eth.export("$CHAIN_TEST");
EOF

peer 11 12k
sleep 2
test_node $NAME "" -loglevel 5 $JSFILE

