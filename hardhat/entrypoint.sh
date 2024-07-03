#!/bin/sh

# Change to the correct directory
cd /usr/src/app/hardhat;

# Start hardhat node as a background process
nohup npx hardhat node &

# Wait for hardhat node to initialize and then deploy contracts
npx wait-on http://127.0.0.1:8545 && nohup npm run deploy;

# The hardhat node process never completes
# Waiting prevents the container from pausing
wait $!