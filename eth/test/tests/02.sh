#!/bin/bash

TIMEOUT=6

cat >> $JSFILE <<EOF
eth.addPeer("localhost:30311");
sleep(200);
eth.addPeer("localhost:30312");
sleep(3000);
eth.export("$CHAIN_TEST");
EOF

peer 11 01
peer 12 02
P12ID=$PID
test_node $NAME "" -loglevel 5 $JSFILE
sleep 0.3
kill $P12ID

