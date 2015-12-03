#!/bin/bash

function up { #file port
  echo "Upload file '$1' to node $2 on port 85$2" 1>&2
  key=`bash swarm/cmd/bzzup.sh $1 85$2`
  echo -n $key
}

function down { #key port
  echo "Download hash '$1' from node $2 on port 85$2"
  wget -O- http://localhost:85$2/$1 > /dev/null && echo "got it" || echo "not found"
}

function clean { #index
  echo "Clean up for $1"
  rm -rf ~/tmp/sync/$1/{bzz/*/chunks,bzz/*/requests/,bzz/*/bzz-peers.json,chaindata,nodes}
}

function gethup { #index account
  cp geth geth$1
  echo "start node $1"
  echo "./geth$1 --datadir ~/tmp/sync/$1 --bzzaccount $2 --unlock 0 --password <(echo bzz) --networkid 323 --port 303$1 --vmodule netstore=6,depo=6,forwarding=6,hive=5,dpa=6,dpa=6,http=6,syncb=6,syncer=6,protocol=6,swap=6,chequebook=6 --shh=false --nodiscover --maxpeers 20 --dev --vmdebug=false --verbosity 4 2> sync$1.log &"
  ./geth$1 --datadir ~/tmp/sync/$1 --bzzaccount "$2" --unlock 0 --password <(echo bzz) --networkid 323 --port 303$1 --vmodule netstore=6,depo=6,forwarding=6,hive=5,dpa=6,dpa=6,http=6,syncb=6,syncer=6,protocol=6,swap=6,chequebook=6 --shh=false --nodiscover --maxpeers 20 --dev --vmdebug=false --verbosity 4 2> sync$1.log
}

function gethdown { #index
  killall -INT geth$1
}

wait=5
gethdown 00
gethdown 01

# ./geth --datadir ~/tmp/sync/00 --password <(echo bzz) account new
BZZKEY00=5c6332e46a095feb9da1ed9803af2fa425f96aa6
BZZKEY01=7dafa7436cba4b5b94a0452557b20b01d23bdc05
echo "Fresh start, wipe datadir"
clean 00
clean 01

gethup 00 $BZZKEY00 &
sleep 2
echo "beyond\n"
key=$(up COPYING.LESSER 00)
echo "beyond\n"
down $key 00

echo "beyond\n"
gethup 01 $BZZKEY01 &
sleep $wait
down $key 01
gethdown 01


key=$(up AUTHORS 00)
down $key 00
gethup 01 $BZZKEY01 &
sleep $wait
down $key 01

key=$(up COPYING 00)
down $key 00
down $key 01

key=$(up README.md 01)
down $key 01
sleep $wait
down $key 00

exit 0

gethdown 00
key=$(up README.md 01)
down $key 01
gethup 00 $BZZKEY00 &
sleep $wait
down $key 00

gethdown 00
gethdown 01



