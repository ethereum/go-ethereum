#!/bin/sh
# Node 2: Not whitelisted PoW miner
set -e

# Initialize with PoW genesis
geth --datadir /app/node2 init /app/docker/genesis.json

# For Ethash mining we don't need local keys; we can mine directly to the etherbase.
geth --datadir /app/node2 \
  --networkid 1234 --nodiscover \
  --http --http.addr 0.0.0.0 --http.port 8546 \
  --port 30304 \
  --http.api eth,net,web3,admin,miner \
  --mine --miner.etherbase 0xab52b2c71f61cd9447a932c0cb55d1752571dab8