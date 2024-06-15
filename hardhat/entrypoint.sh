#!/bin/bash

# Deploy the contracts using Hardhat Ignition
npx hardhat ignition deploy /usr/app/ignition/modules/Apollo.js

# Run the tests
npx hardhat test
