#!/usr/bin/env bash
#
# start_partial_sync.sh - Start a partial state sync on bal-devnet-2.
#
# This script builds geth, initializes the genesis (if needed), and starts
# geth in partial state mode tracking active devnet contracts.
# After starting geth, you must also start Lighthouse (see start_lighthouse.sh).
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GETH_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
DATADIR="$HOME/.bal-devnet-2-partial"
CONTRACTS_FILE="$SCRIPT_DIR/contracts.json"
GENESIS_FILE="$SCRIPT_DIR/bal-devnet-2/genesis.json"
ENODES_FILE="$SCRIPT_DIR/bal-devnet-2/enodes.txt"
JWT_FILE="$DATADIR/jwt.hex"
LOG_FILE="$DATADIR/geth.log"
NETWORK_ID=7033429093

echo "=== Partial State Sync Setup (bal-devnet-2) ==="
echo "Geth source:     $GETH_DIR"
echo "Data directory:  $DATADIR"
echo "Contracts file:  $CONTRACTS_FILE"
echo "Genesis file:    $GENESIS_FILE"
echo "Network ID:      $NETWORK_ID"
echo ""

# Step 1: Always rebuild geth from current source to ensure fixes are included
echo "Building geth from source at $GETH_DIR ..."
cd "$GETH_DIR"
go build -o build/bin/geth ./cmd/geth
GETH="$GETH_DIR/build/bin/geth"
echo "Built: $GETH"
echo "Binary hash: $(shasum -a 256 "$GETH" | cut -d' ' -f1)"
echo ""

# Step 2: Create datadir if needed
mkdir -p "$DATADIR"

# Step 3: Generate JWT secret (if not exists)
if [ ! -f "$JWT_FILE" ]; then
    echo "Generating JWT secret..."
    openssl rand -hex 32 > "$JWT_FILE"
    echo "JWT secret: $JWT_FILE"
else
    echo "JWT secret already exists: $JWT_FILE"
fi
echo ""

# Step 4: Initialize genesis (if chaindata doesn't exist yet)
if [ ! -d "$DATADIR/geth/chaindata" ]; then
    echo "Initializing genesis from $GENESIS_FILE ..."
    "$GETH" init --datadir "$DATADIR" "$GENESIS_FILE"
    echo "Genesis initialized."
else
    echo "Chaindata already exists, skipping genesis init."
fi
echo ""

# Step 5: Verify contracts file exists
if [ ! -f "$CONTRACTS_FILE" ]; then
    echo "ERROR: Contracts file not found: $CONTRACTS_FILE"
    exit 1
fi
echo "Tracked contracts:"
cat "$CONTRACTS_FILE" | python3 -c "
import json, sys
data = json.load(sys.stdin)
for c in data['contracts']:
    print(f\"  {c['name']:20s} {c['address']}\")
" 2>/dev/null || cat "$CONTRACTS_FILE"
echo ""

# Step 6: Read bootnodes from enodes.txt
BOOTNODES=""
if [ -f "$ENODES_FILE" ]; then
    BOOTNODES=$(cat "$ENODES_FILE" | tr '\n' ',' | sed 's/,$//')
    echo "Bootnodes loaded: $(echo "$BOOTNODES" | tr ',' '\n' | wc -l | tr -d ' ') nodes"
else
    echo "WARNING: No enodes file found at $ENODES_FILE"
fi
echo ""

# Step 7: Start geth
echo "Starting geth in partial state mode..."
echo "Log file: $LOG_FILE"
echo ""

"$GETH" \
    --networkid "$NETWORK_ID" \
    --syncmode snap \
    --partial-state \
    --partial-state.contracts-file "$CONTRACTS_FILE" \
    --partial-state.bal-retention 256 \
    --partial-state.chain-retention 1024 \
    --history.logs.disable \
    --datadir "$DATADIR" \
    --authrpc.jwtsecret "$JWT_FILE" \
    --bootnodes "$BOOTNODES" \
    --http \
    --http.api eth,net,web3,debug \
    --http.addr 127.0.0.1 \
    --http.port 8545 \
    --authrpc.addr 127.0.0.1 \
    --authrpc.port 8551 \
    --verbosity 4 \
    --nat upnp \
    --log.file "$LOG_FILE" \
    &

GETH_PID=$!
echo "Geth started (PID: $GETH_PID)"
echo ""

cat <<INSTRUCTIONS
========================================
  NEXT STEP: Start Lighthouse
========================================

Geth (Execution Layer) is running. Now start Lighthouse in a new terminal:

  ./scripts/partial-sync/start_lighthouse.sh

Monitor sync progress:
  tail -f $LOG_FILE | grep -iE "partial|syncing|sync stats|Advanced|BAL|newPayload"

Check sync status via RPC:
  curl -s -X POST http://localhost:8545 \\
    -H "Content-Type: application/json" \\
    -d '{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}' | jq

========================================
INSTRUCTIONS

# Wait for geth process
wait $GETH_PID
