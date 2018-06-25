#!/bin/sh

Datadir0="~/.mainnode"
Datadir1="~/.peernode"

#build/bin/geth --datadir $Datadir0 removedb
#build/bin/geth --datadir $Datadir1 removedb

rm -rf ~/.ethash
rm -rf $Datadir0
rm -rf $Datadir1

echo "node datadir cleared"

echo "main node init"
build/bin/geth --datadir $Datadir0 init genesis.json
build/bin/geth --datadir $Datadir0 --port 30303 --networkid 9876 --exec 'loadScript("yapen/mainnode.js")' console &

sleep 4s

echo "peer node init"
build/bin/geth --datadir $Datadir1 init genesis.json
build/bin/geth --datadir $Datadir1 --port 30304 --networkid 9876 --exec 'loadScript("yapen/peernode.js")' console

echo "all command executed"

