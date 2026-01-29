# XDC Network Synchronization Guide

## Overview

This guide covers the various synchronization modes available for the XDC Network and how to optimize sync performance.

## Sync Modes

### 1. Full Sync
Downloads and verifies every block from genesis.

```bash
./XDC --syncmode full
```

**Pros:**
- Complete blockchain history
- Can serve historical data
- Most secure

**Cons:**
- Longest sync time
- Highest disk usage

### 2. Fast Sync (Default)
Downloads blocks and state at a recent checkpoint.

```bash
./XDC --syncmode fast
```

**Pros:**
- Faster initial sync
- Lower disk usage during sync
- Good for most users

**Cons:**
- No historical state
- Depends on checkpoint validity

### 3. Snap Sync
Uses snapshot data for rapid synchronization.

```bash
./XDC --xdc.snapsync --xdc.snapshot.block 50000000
```

**Pros:**
- Fastest sync method
- Minimal bandwidth
- Ideal for new nodes

**Cons:**
- Requires trusted snapshot
- No pre-snapshot history

## XDC-Specific Sync Features

### Checkpoint Verification
XDC uses checkpoints for additional security.

```bash
./XDC --xdc.checkpoint.interval 900
```

### Validator Set Sync
Synchronizes the current validator set.

```go
// Automatically syncs validator set at epoch boundaries
```

### Trading State Sync (XDCx)
Syncs order book state if XDCx is enabled.

```bash
./XDC --xdcx
```

### Lending State Sync
Syncs lending state if XDCxLending is enabled.

```bash
./XDC --xdcxlending
```

## Performance Optimization

### Hardware Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 4 cores | 8+ cores |
| RAM | 16 GB | 32+ GB |
| Disk | 500 GB SSD | 1+ TB NVMe |
| Network | 25 Mbps | 100+ Mbps |

### Configuration Tuning

#### Cache Settings
```bash
./XDC --cache 4096 --cache.gc 50
```

#### Database Settings
```bash
./XDC --db.engine pebble
```

#### Network Settings
```bash
./XDC --maxpeers 50 --maxpendpeers 25
```

## Sync Progress Monitoring

### RPC Methods
```javascript
// Get sync status
eth.syncing

// Get current block
eth.blockNumber

// Get peer count
net.peerCount
```

### Log Monitoring
```bash
tail -f /var/log/xdc/node.log | grep -i sync
```

### Expected Sync Times

| Mode | Mainnet | Testnet |
|------|---------|---------|
| Full | 3-7 days | 1-2 days |
| Fast | 6-24 hours | 2-6 hours |
| Snap | 1-4 hours | 30-60 min |

## Troubleshooting

### Sync Stuck

1. Check peer connections
```bash
./XDC attach --exec "admin.peers.length"
```

2. Add bootnodes
```bash
./XDC --bootnodes "enode://..."
```

3. Clear bad peers
```bash
./XDC attach --exec "admin.peers"
```

### Low Peer Count

1. Check firewall rules
2. Open P2P ports (30303)
3. Enable UPnP

```bash
./XDC --nat extip:YOUR_IP
```

### Disk Space Issues

1. Enable pruning
```bash
./XDC --gcmode archive
```

2. Use snap sync
3. Resize disk

### Memory Issues

1. Reduce cache
```bash
./XDC --cache 2048
```

2. Limit peers
```bash
./XDC --maxpeers 25
```

## Network-Specific Settings

### Mainnet
```bash
./XDC --xdc.mainnet \
  --datadir /data/xdc/mainnet \
  --syncmode fast
```

### Testnet (Apothem)
```bash
./XDC --xdc.testnet \
  --datadir /data/xdc/testnet \
  --syncmode fast
```

### Devnet
```bash
./XDC --xdc.devnet \
  --datadir /data/xdc/devnet \
  --syncmode full
```

## State Pruning

### Enable Pruning
```bash
./XDC --gcmode full --state.gc.percent 25
```

### Manual Pruning
```bash
./XDC snapshot prune-state --datadir /data/xdc
```

## Backup and Recovery

### Create Backup
```bash
# Stop node first
./XDC copydb /data/xdc /backup/xdc
```

### Restore Backup
```bash
cp -r /backup/xdc /data/xdc
./XDC --datadir /data/xdc
```

## Monitoring Tools

### Prometheus Metrics
```bash
./XDC --metrics --metrics.addr 0.0.0.0 --metrics.port 6060
```

### Key Metrics
- `chain_head_block`: Current block number
- `p2p_peers`: Connected peer count
- `chain_sync_mode`: Current sync mode

## Best Practices

1. **Use SSDs**: Much faster than HDDs
2. **Adequate RAM**: Prevents swap usage
3. **Stable Network**: Reduces peer churn
4. **Regular Backups**: Protect against corruption
5. **Monitor Disk**: Prevent space exhaustion

## Further Reading

- [Node Deployment Guide](./DEPLOYMENT.md)
- [Network Configuration](./NETWORK.md)
- [Performance Tuning](./PERFORMANCE.md)
