#!/bin/sh
set -exu

docker-compose -f docker-compose-add-signer.yml --profile settlement down

for var in "$@"
do
    docker exec $var geth attach  --exec "clique.propose(\"$SIGNER_NODE_ADDRESS\", false)" /data/geth.ipc
done
