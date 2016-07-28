#!/bin/bash

# Startup script to initialize and boot a go-ethereum instance as a swarm node.
#
# This script assumes the following files:
#  - `geth` binary is located in the filesystem root
#  - `genesis.json` file is located in the filesystem root

# Immediately abort the script on any error encountered
set -e

if [ "$SWARM_NETWORK_ID" = "" ]; then export SWARM_NETWORK_ID=322; fi

if [ ! -f "/swarm/data/nodekey" ]; then
    # First run
    echo "Initializing swarm node..."
    mkdir -p /swarm/data
    mkdir -p /swarm/enodes
    mkdir -p /swarm/pids
    /geth --datadir=/swarm/data --password=<(echo -n) account new
fi

/geth   --dev \
        --maxpeers=40 \
        --shh=false \
        --networkid=$SWARM_NETWORK_ID \
        --bzznoswap \
        --verbosity=6 \
        --vmodule=swarm/*=5,discover=5 \
        --datadir=/swarm/data \
        --bzzaccount=0 \
        --unlock=0 \
        --port=30303 \
        --bzzport=32200 \
        --nat=none \
        --rpc \
        --rpcaddr="0.0.0.0" \
        --rpccorsdomain="*" \
        --password=<(echo -n) \
        $*
