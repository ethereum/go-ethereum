#!/bin/bash

# XDC Genesis Initialization Script
# Initializes a node with the appropriate genesis file

set -e

# Configuration
DATADIR="${DATADIR:-$HOME/.xdc}"
NETWORK="${NETWORK:-mainnet}"
FORCE="${FORCE:-false}"

# Find script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GENESIS_DIR="$(dirname "$SCRIPT_DIR")/genesis"

# Select genesis file
case "$NETWORK" in
    mainnet)
        GENESIS_FILE="$GENESIS_DIR/xdc_mainnet.json"
        ;;
    testnet|apothem)
        GENESIS_FILE="$GENESIS_DIR/xdc_apothem.json"
        ;;
    devnet)
        GENESIS_FILE="$GENESIS_DIR/devnet.json"
        ;;
    *)
        echo "Unknown network: $NETWORK"
        echo "Supported networks: mainnet, testnet, apothem, devnet"
        exit 1
        ;;
esac

# Check if genesis file exists
if [ ! -f "$GENESIS_FILE" ]; then
    echo "Genesis file not found: $GENESIS_FILE"
    exit 1
fi

# Check if already initialized
if [ -d "$DATADIR/XDC/chaindata" ] && [ "$FORCE" != "true" ]; then
    echo "Node already initialized at $DATADIR"
    echo "Use FORCE=true to reinitialize (WARNING: This will delete existing data)"
    exit 1
fi

# Remove existing data if force is set
if [ "$FORCE" = "true" ] && [ -d "$DATADIR/XDC" ]; then
    echo "Removing existing data..."
    rm -rf "$DATADIR/XDC"
fi

# Create data directory
mkdir -p "$DATADIR"

# Initialize genesis
echo "Initializing $NETWORK genesis..."
XDC init --datadir "$DATADIR" "$GENESIS_FILE"

echo "Genesis initialization complete!"
echo "Data directory: $DATADIR"
echo "Network: $NETWORK"
