#!/bin/bash

TIMEOUT=5

cat >> $JSFILE <<EOF
eth.addPeer("localhost:30311");
log("added peer localhost:30311");
sleep(1000);
log("added peer localhost:30312");
eth.addPeer("localhost:30312");
sleep(3000);
eth.export("$CHAIN_TEST");
EOF

peer 11 01
peer 12 02
test_node $NAME "" -loglevel 5 $JSFILE

