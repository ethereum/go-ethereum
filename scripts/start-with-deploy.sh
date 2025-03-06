#!/bin/sh

# Start geth in the background with the provided arguments
# First initialize with genesis block
geth --datadir /app/data init /app/genesis.json

# Then start geth with API options
geth --dev --http --http.addr=0.0.0.0 --http.api=eth,net,web3,txpool,debug,admin --dev.period 1 --datadir /app/data  &
GETH_PID=$!

echo "Waiting for Geth RPC to be ready..."
until curl --silent --fail http://localhost:8545 > /dev/null; do
  sleep 1
done
echo "Geth RPC is available"

# Deploy contracts if needed
cd /app/hardhat

# yes "y" | npx hardhat run scripts/deploy.js --network localhost
# sleep 10
yes "y" | npx hardhat ignition deploy ignition/modules/Lock.js --network localhost

# Wait for the geth process to finish
wait $GETH_PID