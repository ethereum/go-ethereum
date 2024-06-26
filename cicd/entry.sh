#!/bin/bash
if test -z "$NETWORK"
then
    echo "NETWORK env Must be set, mainnet/testnet/devnet/local"
    exit 1
fi

echo "Select to run $NETWORK..."
ln -s /usr/bin/XDC-$NETWORK /usr/bin/XDC
cp /work/$NETWORK/* /work

echo "Start Node..."
/work/start.sh