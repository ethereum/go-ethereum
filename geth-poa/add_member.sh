#!/bin/sh
set -exu

export MEMBER_NODE_VOLUME=${MEMBER_NODE_VOLUME:-"geth-data-member"}
export MEMBER_NODE_PORT=${MEMBER_NODE_PORT:-60605}
export SIGNER_NODE_IP=${SIGNER_NODE_IP:-"172.29.0.103"}

docker-compose -f docker-compose-add-member.yml --profile settlement up -d --build
