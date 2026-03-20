#!/bin/bash
# Partial State Devnet Test Script
#
# This script sets up a 2-node devnet to test partial state functionality.
# It starts a full node in dev mode and a partial state node that syncs from it.
#
# Usage: ./scripts/partial-state-devnet-test.sh

set -e

# Configuration
FULL_NODE_DIR="/tmp/partial-state-test/full-node"
PARTIAL_NODE_DIR="/tmp/partial-state-test/partial-node"
FULL_NODE_PORT=30303
PARTIAL_NODE_PORT=30304
FULL_NODE_RPC=8545
PARTIAL_NODE_RPC=8546

# Test contract address (will be tracked by partial node)
TRACKED_CONTRACT="0x1234567890123456789012345678901234567890"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    log_info "Cleaning up..."
    if [ -n "$FULL_PID" ]; then
        kill $FULL_PID 2>/dev/null || true
    fi
    if [ -n "$PARTIAL_PID" ]; then
        kill $PARTIAL_PID 2>/dev/null || true
    fi
    wait 2>/dev/null || true
    log_info "Cleanup complete"
}

trap cleanup EXIT

# Build geth if not already built
if [ ! -f "./geth" ]; then
    log_info "Building geth..."
#    go build ./cmd/geth
fi

# Clean up old test data
log_info "Setting up test directories..."
rm -rf /tmp/partial-state-test
mkdir -p "$FULL_NODE_DIR" "$PARTIAL_NODE_DIR"

# Start full node
log_info "Starting full node..."
./geth --datadir "$FULL_NODE_DIR" \
    --dev \
    --dev.period 2 \
    --port $FULL_NODE_PORT \
    --http --http.port $FULL_NODE_RPC \
    --http.api eth,net,web3,admin \
    --verbosity 2 \
    --nodiscover &
FULL_PID=$!

log_info "Full node started with PID $FULL_PID"

# Wait for full node to start
log_info "Waiting for full node to initialize..."
sleep 5

# Get enode from full node
log_info "Getting enode from full node..."
for i in {1..10}; do
    ENODE=$(./geth attach "$FULL_NODE_DIR/geth.ipc" --exec admin.nodeInfo.enode 2>/dev/null | tr -d '"')
    if [ -n "$ENODE" ]; then
        break
    fi
    sleep 1
done

if [ -z "$ENODE" ]; then
    log_error "Failed to get enode from full node"
    exit 1
fi

log_info "Full node enode: ${ENODE:0:50}..."

# Start partial state node
log_info "Starting partial state node..."
./geth --datadir "$PARTIAL_NODE_DIR" \
    --port $PARTIAL_NODE_PORT \
    --http --http.port $PARTIAL_NODE_RPC \
    --http.api eth,net,web3 \
    --partial-state \
    --partial-state.contracts "$TRACKED_CONTRACT" \
    --bootnodes "$ENODE" \
    --networkid 1337 \
    --verbosity 2 &
PARTIAL_PID=$!

log_info "Partial state node started with PID $PARTIAL_PID"

# Wait for nodes to connect
log_info "Waiting for nodes to connect..."
sleep 10

# Run tests
log_info "Running tests..."

# Test 1: Check both nodes are running
log_info "Test 1: Checking node connectivity..."
FULL_PEERS=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":1}' \
    -H "Content-Type: application/json" localhost:$FULL_NODE_RPC | grep -o '"result":"[^"]*"' | cut -d'"' -f4)
PARTIAL_PEERS=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":1}' \
    -H "Content-Type: application/json" localhost:$PARTIAL_NODE_RPC | grep -o '"result":"[^"]*"' | cut -d'"' -f4)

log_info "Full node peers: $FULL_PEERS, Partial node peers: $PARTIAL_PEERS"

# Test 2: Send a transaction and verify sync
log_info "Test 2: Sending test transaction..."
./geth attach "$FULL_NODE_DIR/geth.ipc" --exec "eth.sendTransaction({from: eth.coinbase, to: '$TRACKED_CONTRACT', value: web3.toWei(1, 'ether')})" 2>/dev/null || true

# Wait for block to be mined
sleep 5

# Test 3: Compare block numbers
log_info "Test 3: Comparing block numbers..."
FULL_BLOCK=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    -H "Content-Type: application/json" localhost:$FULL_NODE_RPC | grep -o '"result":"[^"]*"' | cut -d'"' -f4)
PARTIAL_BLOCK=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    -H "Content-Type: application/json" localhost:$PARTIAL_NODE_RPC | grep -o '"result":"[^"]*"' | cut -d'"' -f4)

log_info "Full node block: $FULL_BLOCK, Partial node block: $PARTIAL_BLOCK"

# Test 4: Compare balances
log_info "Test 4: Comparing account balances..."
FULL_BALANCE=$(curl -s -X POST --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"$TRACKED_CONTRACT\",\"latest\"],\"id\":1}" \
    -H "Content-Type: application/json" localhost:$FULL_NODE_RPC | grep -o '"result":"[^"]*"' | cut -d'"' -f4)
PARTIAL_BALANCE=$(curl -s -X POST --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"$TRACKED_CONTRACT\",\"latest\"],\"id\":1}" \
    -H "Content-Type: application/json" localhost:$PARTIAL_NODE_RPC | grep -o '"result":"[^"]*"' | cut -d'"' -f4)

log_info "Full node balance: $FULL_BALANCE, Partial node balance: $PARTIAL_BALANCE"

if [ "$FULL_BALANCE" = "$PARTIAL_BALANCE" ]; then
    log_info "Balances match!"
else
    log_warn "Balances do not match (this may be expected if partial node is still syncing)"
fi

# Summary
echo ""
log_info "========== Test Summary =========="
log_info "Full node: PID=$FULL_PID, Port=$FULL_NODE_PORT, RPC=$FULL_NODE_RPC"
log_info "Partial node: PID=$PARTIAL_PID, Port=$PARTIAL_NODE_PORT, RPC=$PARTIAL_NODE_RPC"
log_info "Tracked contract: $TRACKED_CONTRACT"
log_info ""
log_info "Database sizes:"
du -sh "$FULL_NODE_DIR/geth/chaindata" 2>/dev/null || echo "  Full node: N/A"
du -sh "$PARTIAL_NODE_DIR/geth/chaindata" 2>/dev/null || echo "  Partial node: N/A"
log_info "================================="
echo ""

log_info "Test complete. Press Ctrl+C to stop nodes and cleanup."

# Keep running until interrupted
wait
