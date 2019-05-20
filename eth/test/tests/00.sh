#!/bin/bash

TIMEOUT=4

cat >> $JSFILE <<EOF
eth.addPeer("localhost:30311");
sleep(1000)
eth.export("$CHAIN_TEST");
EOF

peer 11 01
test_node $NAME "" -loglevel 5 $JSFILE

