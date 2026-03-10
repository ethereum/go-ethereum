#!/bin/bash

case "$NETWORK" in
    "")
        echo "NETWORK environment variable must be set. Allowed values: mainnet/testnet/devnet/local"
        exit 1
        ;;
    mainnet|testnet|devnet|local)
        ;;
    *)
        echo "Invalid NETWORK: $NETWORK. Allowed: mainnet/testnet/devnet/local"
        exit 1
        ;;
esac

echo "Select to run $NETWORK..."
cp -n /work/"$NETWORK"/* /work

echo "Start Node..."
/work/start.sh
