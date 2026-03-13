#!/bin/sh
# Node 1: Whitelisted PoW miner
set -e

# Initialize with PoW genesis
geth --datadir /app/node1 init /app/docker/genesis.json

# For Ethash mining we don't need local keys; we can mine directly to the etherbase.
geth --datadir /app/node1 \
  --networkid 1234 --nodiscover \
  --http --http.addr 0.0.0.0 --http.port 8545 \
  --port 30303 \
  --http.api eth,net,web3,admin,miner \
  --mine --miner.etherbase 0xca6b49ee60cdd276ab503fbd6fb80a3cfbc06ffc