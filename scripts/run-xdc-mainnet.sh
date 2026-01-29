#!/bin/bash
# XDC Mainnet Node Startup Script
# Part4 Implementation (go-ethereum v1.15 based)

set -e

# Configuration
DATADIR="${DATADIR:-$HOME/.xdc-part4}"
NETWORK_ID=50
PORT="${PORT:-30304}"
HTTP_PORT="${HTTP_PORT:-8545}"
WS_PORT="${WS_PORT:-8546}"
VERBOSITY="${VERBOSITY:-3}"
SYNCMODE="${SYNCMODE:-full}"

# XDC Mainnet Bootnodes
BOOTNODES="enode://91e59fa1b034ae35e9f4e8a99cc6621f09d74e76a6220abb6c93b29ed41a9e1fc4e5b70e2c5fc43f883cffbdcd6f4f6cbc1d23af077f28c2aecc22403355d4b1@209.126.0.250:30304,enode://91e59fa1b034ae35e9f4e8a99cc6621f09d74e76a6220abb6c93b29ed41a9e1fc4e5b70e2c5fc43f883cffbdcd6f4f6cbc1d23af077f28c2aecc22403355d4b1@209.126.4.150:30304,enode://91e59fa1b034ae35e9f4e8a99cc6621f09d74e76a6220abb6c93b29ed41a9e1fc4e5b70e2c5fc43f883cffbdcd6f4f6cbc1d23af077f28c2aecc22403355d4b1@144.126.150.58:30304,enode://91e59fa1b034ae35e9f4e8a99cc6621f09d74e76a6220abb6c93b29ed41a9e1fc4e5b70e2c5fc43f883cffbdcd6f4f6cbc1d23af077f28c2aecc22403355d4b1@162.250.190.246:30304,enode://7524db6718828c2c7663e6585a5b1e066457b8b0235034b69358b36e584fea776666d36ed4fc43d0f8bf2a5c3b2a960b5600689b6c8f0c207e5a76f8b0ca432d@157.173.120.219:30304"

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
GENESIS_FILE="$PROJECT_ROOT/genesis/xdc_mainnet.json"

# Build if binary doesn't exist
GETH_BIN="$PROJECT_ROOT/build/bin/geth"
if [ ! -f "$GETH_BIN" ]; then
    echo "Building geth..."
    cd "$PROJECT_ROOT"
    go build -o build/bin/geth ./cmd/geth
fi

# Initialize datadir if needed
if [ ! -d "$DATADIR/geth/chaindata" ]; then
    echo "Initializing datadir with XDC mainnet genesis..."
    mkdir -p "$DATADIR"
    "$GETH_BIN" init --datadir "$DATADIR" "$GENESIS_FILE"
    echo ""
    echo "Genesis initialized. Expected hash: 0x4a9d748bd78a8d0385b67788c2435dcdb914f98a96250b68863a1f8b7642d6b1"
fi

echo ""
echo "========================================="
echo "Starting XDC Mainnet Node (Part4)"
echo "========================================="
echo "Datadir:    $DATADIR"
echo "Network ID: $NETWORK_ID"
echo "P2P Port:   $PORT"
echo "HTTP Port:  $HTTP_PORT"
echo "WS Port:    $WS_PORT"
echo "Sync Mode:  $SYNCMODE"
echo "========================================="
echo ""

# Start the node
exec "$GETH_BIN" \
    --datadir "$DATADIR" \
    --networkid $NETWORK_ID \
    --port $PORT \
    --bootnodes "$BOOTNODES" \
    --syncmode "$SYNCMODE" \
    --http \
    --http.addr "0.0.0.0" \
    --http.port $HTTP_PORT \
    --http.api "eth,net,web3,txpool,debug,admin" \
    --http.corsdomain "*" \
    --http.vhosts "*" \
    --ws \
    --ws.addr "0.0.0.0" \
    --ws.port $WS_PORT \
    --ws.api "eth,net,web3,txpool" \
    --ws.origins "*" \
    --verbosity $VERBOSITY \
    --nat "any" \
    "$@"
