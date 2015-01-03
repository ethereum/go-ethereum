#!/bin/bash
. `dirname $BASH_SOURCE`/common.sh

TIMEOUT=10
ID=01
JSFILE="$DIR/js/$ID.js"

echo $JSFILE
cat > $JSFILE <<EOF
eth.addPeer("localhost:30311");
var now = new Date().getTime();
while(new Date().getTime() < now + 100){}
eth.addPeer("localhost:30310");
var now = new Date().getTime();
while(new Date().getTime() < now + 2000){}
eth.export("$CHAIN_TEST");
EOF

peer 10 01 -loglevel 5
peer 11 02 -loglevel 5
sleep 1
peer $ID "" -loglevel 5 $JSFILE

