#!/usr/bin/env bash
#
# start_partial_sync.sh - Start a partial state sync on Ethereum mainnet.
#
# This script builds geth, generates a JWT secret, and starts geth in partial
# state mode tracking only WETH and DAI contracts. After starting geth, you
# must also start a Consensus Layer client (instructions printed at the end).
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GETH_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
DATADIR="$HOME/.ethereum-partial-test"
CONTRACTS_FILE="$SCRIPT_DIR/contracts.json"
JWT_FILE="$DATADIR/jwt.hex"
LOG_FILE="$DATADIR/geth.log"

echo "=== Partial State Sync Setup ==="
echo "Geth source:     $GETH_DIR"
echo "Data directory:  $DATADIR"
echo "Contracts file:  $CONTRACTS_FILE"
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

# Step 4: Verify contracts file exists
if [ ! -f "$CONTRACTS_FILE" ]; then
    echo "ERROR: Contracts file not found: $CONTRACTS_FILE"
    exit 1
fi
echo "Tracked contracts:"
cat "$CONTRACTS_FILE" | python3 -c "
import json, sys
data = json.load(sys.stdin)
for c in data['contracts']:
    print(f\"  {c['name']:10s} {c['address']}\")
" 2>/dev/null || cat "$CONTRACTS_FILE"
echo ""

# Step 5: Start geth
echo "Starting geth in partial state mode..."
echo "Log file: $LOG_FILE"
echo ""

"$GETH" \
    --mainnet \
    --syncmode snap \
    --partial-state \
    --partial-state.contracts-file "$CONTRACTS_FILE" \
    --partial-state.bal-retention 256 \
    --partial-state.chain-retention 1024 \
    --history.logs.disable \
    --datadir "$DATADIR" \
    --authrpc.jwtsecret "$JWT_FILE" \
    --http \
    --http.api eth,net,web3,debug \
    --http.addr 127.0.0.1 \
    --http.port 8545 \
    --authrpc.addr 127.0.0.1 \
    --authrpc.port 8551 \
    --verbosity 3 \
    --log.file "$LOG_FILE" \
    &

GETH_PID=$!
echo "Geth started (PID: $GETH_PID)"
echo ""

# Step 6: Print CL instructions
cat <<'INSTRUCTIONS'
========================================
  NEXT STEP: Start a Consensus Layer client
========================================

Geth (Execution Layer) is running. You now need a Consensus Layer client.
Lighthouse is recommended. Install it from:

  https://lighthouse-book.sigmaprime.io/installation.html

Then run (in a new terminal):

INSTRUCTIONS

echo "  lighthouse bn \\"
echo "    --network mainnet \\"
echo "    --checkpoint-sync-url https://mainnet.checkpoint.sigp.io \\"
echo "    --execution-endpoint http://localhost:8551 \\"
echo "    --execution-jwt $JWT_FILE \\"
echo "    --datadir $HOME/.lighthouse-partial-test \\"
echo "    --slots-per-restore-point 8192 \\"
echo "    --disable-deposit-contract-sync \\"
echo "    --prune-blobs true \\"
echo "    --disable-backfill-rate-limiting \\"
echo "    --disable-optimistic-finalized-sync"

cat <<'INSTRUCTIONS'

Monitor sync progress:
  tail -f ~/.ethereum-partial-test/geth.log | grep -i "partial\|syncing\|sync stats"

Check sync status via RPC:
  curl -s -X POST http://localhost:8545 \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}' | jq

When sync completes, run verification:
  ./scripts/partial-sync/verify_partial_sync.sh

========================================
INSTRUCTIONS

# Wait for geth process
wait $GETH_PID
