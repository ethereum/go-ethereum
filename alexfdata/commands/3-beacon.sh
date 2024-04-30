#!/bin/sh -e

../beacon_chain --datadir ../beacondata --min-sync-peers 0 --genesis-state ../genesis.ssz --bootstrap-node= --interop-eth1data-votes --chain-config-file ../config.yml --contract-deployment-block 0 --chain-id 1337 --accept-terms-of-use --jwt-secret ../jwt.hex --suggested-fee-recipient 0x123463a4B065722E99115D6c222f267d9cABb524 --minimum-peers-per-subnet 0 --enable-debug-rpc-endpoints --execution-endpoint ../gethdata/geth.ipc
