# Testing XDC Part4 Node

## Quick Start

### 1. Build the node

```bash
cd /path/to/xdpos-part4
go build -o build/bin/geth ./cmd/geth
```

### 2. Initialize with XDC mainnet genesis

```bash
./build/bin/geth init --datadir ~/.xdc-part4 genesis/xdc_mainnet.json
```

Expected output:
```
Successfully wrote genesis state  database=chaindata hash=4a9d74..42d6b1
```

### 3. Start the node

Using the script:
```bash
chmod +x scripts/run-xdc-mainnet.sh
./scripts/run-xdc-mainnet.sh
```

Or manually:
```bash
./build/bin/geth \
    --datadir ~/.xdc-part4 \
    --networkid 50 \
    --port 30304 \
    --bootnodes "enode://91e59fa1b034ae35e9f4e8a99cc6621f09d74e76a6220abb6c93b29ed41a9e1fc4e5b70e2c5fc43f883cffbdcd6f4f6cbc1d23af077f28c2aecc22403355d4b1@209.126.0.250:30304,enode://7524db6718828c2c7663e6585a5b1e066457b8b0235034b69358b36e584fea776666d36ed4fc43d0f8bf2a5c3b2a960b5600689b6c8f0c207e5a76f8b0ca432d@157.173.120.219:30304" \
    --syncmode full \
    --http \
    --http.addr "0.0.0.0" \
    --http.port 8545 \
    --http.api "eth,net,web3,txpool,debug,admin" \
    --http.corsdomain "*" \
    --verbosity 3
```

## Configuration Options

| Variable | Default | Description |
|----------|---------|-------------|
| DATADIR | ~/.xdc-part4 | Data directory |
| PORT | 30304 | P2P port |
| HTTP_PORT | 8545 | HTTP RPC port |
| WS_PORT | 8546 | WebSocket port |
| SYNCMODE | full | Sync mode (full/snap) |
| VERBOSITY | 3 | Log verbosity (0-5) |

Example with custom settings:
```bash
DATADIR=/data/xdc PORT=30305 HTTP_PORT=8546 ./scripts/run-xdc-mainnet.sh
```

## Verify Node Status

### Check peer count
```bash
curl -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":1}' \
    http://localhost:8545
```

### Check sync status
```bash
curl -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}' \
    http://localhost:8545
```

### Check block number
```bash
curl -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    http://localhost:8545
```

### Get node info
```bash
curl -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"admin_nodeInfo","params":[],"id":1}' \
    http://localhost:8545
```

### List peers
```bash
curl -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"admin_peers","params":[],"id":1}' \
    http://localhost:8545
```

## Using Console

Attach to running node:
```bash
./build/bin/geth attach ~/.xdc-part4/geth.ipc
```

In console:
```javascript
// Check peers
admin.peers.length

// Check sync status  
eth.syncing

// Check block number
eth.blockNumber

// Node info
admin.nodeInfo
```

## XDC Mainnet Details

| Parameter | Value |
|-----------|-------|
| Network ID | 50 |
| Chain ID | 50 |
| Genesis Hash | 0x4a9d748bd78a8d0385b67788c2435dcdb914f98a96250b68863a1f8b7642d6b1 |
| State Root | 0x49be235b0098b048f9805aed38a279d8c189b469ff9ba307b39c7ad3a3bc55ae |
| P2P Port | 30304 |
| Consensus | XDPoS |

## Troubleshooting

### No peers connecting
- Check firewall allows port 30304 (TCP/UDP)
- Verify bootnodes are reachable: `nc -vz 209.126.0.250 30304`
- Check logs for peer discovery messages

### Genesis hash mismatch
- Re-initialize with correct genesis file
- Delete datadir and start fresh: `rm -rf ~/.xdc-part4`

### Node crashes on startup
- Check disk space
- Increase verbosity: `VERBOSITY=4 ./scripts/run-xdc-mainnet.sh`
- Check system logs: `dmesg | tail`

## Development

Run tests:
```bash
go test ./core -run TestXDC -v
```

Build with race detector:
```bash
go build -race -o build/bin/geth ./cmd/geth
```
