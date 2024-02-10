#!/bin/sh
set -exu

export SIGNER_NODE_ADDRESS=${SIGNER_NODE_ADDRESS:-"0x57949c3552159532c324c6fa8b102696cf4504bc"}
export SIGNER_NODE_PRIVATE_KEY=${SIGNER_NODE_PRIVATE_KEY:-"0x0300633b02bab7305e17a2eabc6477f5caa3bc705994d2e19f55e8427c38536e"}
export SIGNER_NODE_VOLUME=${SIGNER_NODE_VOLUME:-"geth-data-signer"}
export SIGNER_NODE_PORT=${SIGNER_NODE_PORT:-60605}
export SIGNER_NODE_IP=${SIGNER_NODE_IP:-"172.29.0.102"}

docker-compose -f docker-compose-add-signer.yml --profile settlement up -d --build

for var in "$@"
do
    docker exec $var geth attach  --exec "clique.propose(\"$SIGNER_NODE_ADDRESS\", true)" /data/geth.ipc
done
