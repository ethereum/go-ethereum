#!/bin/sh
set -exu

if [ "$#" -ne 5 ]; then
    echo "Usage: $0 <containerId> <chaindata> <exportdata> <startBlock> <endBlock>"
    exit 1
fi

CONTAINER_ID=$1
CHAIN_DATA=$2
EXPORT_DATA=$3
START_BLOCK=$4
END_BLOCK=$5
docker stop "${CONTAINER_ID}"
geth --datadir "${CHAIN_DATA}" export "${EXPORT_DATA}" ${START_BLOCK} ${END_BLOCK}
docker start "${CONTAINER_ID}"
