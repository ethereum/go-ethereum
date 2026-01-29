#!/bin/bash
# XDC Network Sync Test
# Tests synchronization with XDC mainnet/testnet

set -e

# Configuration
DATA_DIR="${DATA_DIR:-/tmp/xdc-sync-test}"
NETWORK="${NETWORK:-apothem}"  # mainnet, apothem, or devnet
SYNC_MODE="${SYNC_MODE:-full}"  # full, fast, or snap
LOG_LEVEL="${LOG_LEVEL:-3}"
SYNC_DURATION="${SYNC_DURATION:-300}"  # seconds to sync

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}XDC Network Sync Test${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Determine network ID and bootnodes
case $NETWORK in
    mainnet)
        NETWORK_ID=50
        GENESIS="genesis/xdc_mainnet.json"
        echo -e "${YELLOW}Testing mainnet sync (NetworkID: 50)${NC}"
        ;;
    apothem)
        NETWORK_ID=51
        GENESIS="genesis/xdc_apothem.json"
        echo -e "${YELLOW}Testing apothem testnet sync (NetworkID: 51)${NC}"
        ;;
    devnet)
        NETWORK_ID=551
        GENESIS="genesis/devnet.json"
        echo -e "${YELLOW}Testing devnet sync (NetworkID: 551)${NC}"
        ;;
    *)
        echo -e "${RED}Unknown network: $NETWORK${NC}"
        exit 1
        ;;
esac

# Clean data directory
echo "Cleaning data directory: $DATA_DIR"
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR"

# Find XDC binary
XDC_BIN="./build/bin/XDC"
if [ ! -f "$XDC_BIN" ]; then
    echo "Building XDC..."
    make XDC
fi

echo -e "${YELLOW}Starting sync test...${NC}"
echo "  Network: $NETWORK"
echo "  Sync Mode: $SYNC_MODE"
echo "  Duration: ${SYNC_DURATION}s"
echo "  Data Dir: $DATA_DIR"
echo ""

# Initialize genesis if exists
if [ -f "$GENESIS" ]; then
    echo "Initializing genesis..."
    $XDC_BIN init --datadir "$DATA_DIR" "$GENESIS"
fi

# Start XDC node
echo "Starting XDC node..."
$XDC_BIN \
    --datadir "$DATA_DIR" \
    --networkid $NETWORK_ID \
    --syncmode "$SYNC_MODE" \
    --verbosity $LOG_LEVEL \
    --maxpeers 50 \
    --cache 1024 \
    --http \
    --http.addr "127.0.0.1" \
    --http.port 8545 \
    --http.api "eth,net,web3,xdc" \
    2>&1 | tee "$DATA_DIR/sync.log" &

XDC_PID=$!

# Wait for RPC to be ready
echo "Waiting for RPC..."
sleep 10

# Monitor sync progress
echo -e "${YELLOW}Monitoring sync progress...${NC}"
START_TIME=$(date +%s)
LAST_BLOCK=0

while true; do
    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))
    
    if [ $ELAPSED -ge $SYNC_DURATION ]; then
        echo ""
        echo "Sync duration reached"
        break
    fi
    
    # Get current block
    RESULT=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
        http://127.0.0.1:8545 2>/dev/null || echo '{"result":"0x0"}')
    
    BLOCK_HEX=$(echo $RESULT | grep -o '"result":"[^"]*"' | cut -d'"' -f4)
    
    if [ -n "$BLOCK_HEX" ]; then
        BLOCK=$((16#${BLOCK_HEX#0x}))
        BLOCKS_PER_SEC=$(( (BLOCK - LAST_BLOCK) ))
        LAST_BLOCK=$BLOCK
        
        echo -ne "\rBlock: $BLOCK | Elapsed: ${ELAPSED}s | Speed: ~${BLOCKS_PER_SEC} blocks/sec    "
    fi
    
    sleep 1
done

# Check sync status
echo ""
echo -e "${YELLOW}Checking sync status...${NC}"

SYNC_RESULT=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}' \
    http://127.0.0.1:8545 2>/dev/null)

echo "Sync status: $SYNC_RESULT"

# Get peer count
PEERS_RESULT=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":1}' \
    http://127.0.0.1:8545 2>/dev/null)

PEERS_HEX=$(echo $PEERS_RESULT | grep -o '"result":"[^"]*"' | cut -d'"' -f4)
PEERS=$((16#${PEERS_HEX#0x}))
echo "Connected peers: $PEERS"

# Stop node
echo ""
echo "Stopping XDC node..."
kill $XDC_PID 2>/dev/null || true
wait $XDC_PID 2>/dev/null || true

# Check final block count
FINAL_BLOCK=$LAST_BLOCK

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Sync Test Results${NC}"
echo -e "${GREEN}========================================${NC}"
echo "  Network: $NETWORK"
echo "  Final Block: $FINAL_BLOCK"
echo "  Duration: ${SYNC_DURATION}s"
echo "  Avg Speed: $(( FINAL_BLOCK / SYNC_DURATION )) blocks/sec"
echo "  Peers: $PEERS"

if [ $FINAL_BLOCK -gt 0 ] && [ $PEERS -gt 0 ]; then
    echo -e "${GREEN}✓ Sync test PASSED${NC}"
    exit 0
else
    echo -e "${RED}✗ Sync test FAILED${NC}"
    exit 1
fi
