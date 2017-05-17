#!/bin/bash

# Working directory
cd /tmp

# Preparation
DATADIR=/tmp/BZZ/`date +%s`
mkdir -p $DATADIR
read -s -p "Enter Password. It will be stored in $DATADIR/my-password: " MYPASSWORD && echo $MYPASSWORD > $DATADIR/my-password
echo
BZZKEY=$($GOPATH/bin/geth --datadir $DATADIR --password $DATADIR/my-password account new | awk -F"{|}" '{print $2}')

echo "Your account is ready: "$BZZKEY

# Run geth in the background
nohup $GOPATH/bin/geth --datadir $DATADIR \
	--unlock 0 \
	--password <(cat $DATADIR/my-password) \
	--verbosity 6 \
	--networkid 322 \
	--nodiscover \
	--maxpeers 0 \
	2>> $DATADIR/geth.log &

echo "geth is running in the background, you can check its logs at "$DATADIR"/geth.log"

# Now run swarm in the background
$GOPATH/bin/swarm \
	--bzzaccount $BZZKEY \
	--datadir $DATADIR \
	--ethapi $DATADIR/geth.ipc \
	--verbosity 6 \
	--maxpeers 0 \
	--bzznetworkid 322 \
	&> $DATADIR/swarm.log < <(cat $DATADIR/my-password) &


echo "swarm is running in the background, you can check its logs at "$DATADIR"/swarm.log"

# Cleaning up
# You need to perform this feature manually
# USE THESE COMMANDS AT YOUR OWN RISK!
##
# kill -9 $(ps aux | grep swarm | grep bzzaccount | awk '{print $2}')
# kill -9 $(ps aux | grep geth | grep datadir | awk '{print $2}')
# rm -rf /tmp/BZZ
