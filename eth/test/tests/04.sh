#!/bin/bash

TIMEOUT=15

cat >> $JSFILE <<EOF
eth.addPeer("localhost:30311");
sleep(200);
eth.addPeer("localhost:30312");
sleep(13000);
eth.export("$CHAIN_TEST");
EOF

peer 11 01 -mine
peer 12 02
test_node $NAME "" -loglevel 5 $JSFILE
sleep 6
cat $DIR/$NAME/debug.log | grep 'best peer'
