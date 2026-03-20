# Partial State Devnet Testing Guide

This document describes how to test partial statefulness with a local devnet using 2 geth instances.

## Overview

Partial state nodes:
- Sync all account data (balances, nonces, code hashes)
- Only store storage for tracked contracts
- Process blocks using BAL (Block Access Lists) instead of re-executing transactions

## Prerequisites

- Go 1.22+ installed
- Two terminal windows
- Build geth with partial state support:
  ```bash
  go build ./cmd/geth
  ```

## Setup

### Terminal 1: Full Node (creates blocks in dev mode)

```bash
# Create fresh data directory
rm -rf /tmp/full-node

# Start full node in dev mode
./geth --datadir /tmp/full-node \
    --dev \
    --dev.period 5 \
    --port 30303 \
    --http --http.port 8545 \
    --http.api eth,net,web3,debug,admin \
    --verbosity 3

# Get the enode URL (run in another terminal or use geth attach)
# geth attach /tmp/full-node/geth.ipc --exec admin.nodeInfo.enode
```

### Terminal 2: Partial State Node (receives blocks via P2P)

First, get the enode from the full node:
```bash
ENODE=$(geth attach /tmp/full-node/geth.ipc --exec admin.nodeInfo.enode | tr -d '"')
echo "Full node enode: $ENODE"
```

Then start the partial state node:
```bash
# Create fresh data directory
rm -rf /tmp/partial-node

# Start partial state node
./geth --datadir /tmp/partial-node \
    --port 30304 \
    --http --http.port 8546 \
    --http.api eth,net,web3,debug \
    --partial-state \
    --partial-state.contracts 0xContractAddr1,0xContractAddr2 \
    --bootnodes "$ENODE" \
    --networkid 1337 \
    --verbosity 3
```

Note: Replace `0xContractAddr1,0xContractAddr2` with actual contract addresses you want to track.

## Test Scenarios

### 1. Block Sync Test

Send a transaction on the full node and verify the partial node receives it:

```bash
# On full node (Terminal 1 or new terminal)
geth attach /tmp/full-node/geth.ipc

# In geth console, send a transaction
> eth.sendTransaction({from: eth.coinbase, to: "0x1234567890123456789012345678901234567890", value: web3.toWei(1, "ether")})

# Check block number
> eth.blockNumber
```

Verify on partial node:
```bash
curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    -H "Content-Type: application/json" localhost:8546 | jq
```

### 2. Balance Query Test

Both nodes should return the same balance for any account:

```bash
# Full node
curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x1234567890123456789012345678901234567890","latest"],"id":1}' \
    -H "Content-Type: application/json" localhost:8545 | jq

# Partial node
curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x1234567890123456789012345678901234567890","latest"],"id":1}' \
    -H "Content-Type: application/json" localhost:8546 | jq
```

### 3. Storage Query Test

Deploy a contract and test storage access:

```bash
# Query tracked contract storage (should work)
curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_getStorageAt","params":["0xTrackedContractAddr","0x0","latest"],"id":1}' \
    -H "Content-Type: application/json" localhost:8546 | jq

# Query untracked contract storage (should fail or return empty)
curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_getStorageAt","params":["0xUntrackedContractAddr","0x0","latest"],"id":1}' \
    -H "Content-Type: application/json" localhost:8546 | jq
```

### 4. State Root Verification

Verify both nodes have the same state root:

```bash
# Get latest block from both nodes
FULL_ROOT=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest",false],"id":1}' \
    -H "Content-Type: application/json" localhost:8545 | jq -r '.result.stateRoot')

PARTIAL_ROOT=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest",false],"id":1}' \
    -H "Content-Type: application/json" localhost:8546 | jq -r '.result.stateRoot')

echo "Full node state root: $FULL_ROOT"
echo "Partial node state root: $PARTIAL_ROOT"

if [ "$FULL_ROOT" = "$PARTIAL_ROOT" ]; then
    echo "State roots match!"
else
    echo "State roots DO NOT match!"
fi
```

## Database Size Comparison

After syncing, compare database sizes:

```bash
echo "Full node database size:"
du -sh /tmp/full-node/geth/chaindata

echo "Partial node database size:"
du -sh /tmp/partial-node/geth/chaindata
```

The partial node should have a significantly smaller database size due to skipped storage.

## Cleanup

```bash
# Stop both geth instances (Ctrl+C in each terminal)

# Remove test data
rm -rf /tmp/full-node /tmp/partial-node
```

## Troubleshooting

### Nodes not connecting
- Verify bootnodes enode URL is correct
- Check that network IDs match (dev mode uses 1337)
- Ensure ports are not blocked

### State root mismatch
- This indicates a bug in BAL processing
- Check geth logs for errors during block processing
- Verify the partial node received the BAL with the block

### Storage queries failing
- Verify the contract address is in the tracked contracts list
- Check that the contract was deployed after the partial node started syncing

## Related Documentation

- [EIP-7928: Block Access Lists](https://eips.ethereum.org/EIPS/eip-7928)
- [Partial Statefulness Master Plan](./PARTIAL_STATEFULNESS_PLAN.md)
- [Phase 3 Implementation Plan](./PHASE3_PLAN.md)
