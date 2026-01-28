# XDPoS Consensus Testing Guide

This document describes how to test the XDPoS consensus implementation for syncing with XDC Mainnet and Apothem Testnet.

## Prerequisites

- Go 1.21 or higher
- Git
- At least 500GB disk space for full sync (mainnet)
- Good network connection

## Building

```bash
# Clone the repository
git clone https://github.com/AnilChinchawale/go-ethereum.git
cd go-ethereum
git checkout feature/xdpos-consensus

# Build the XDC binary
make XDC

# Or build all tools
make all
```

## Network Configuration

### XDC Mainnet
- **Chain ID:** 50
- **Network ID:** 50
- **Block Time:** 2 seconds
- **Epoch:** 900 blocks (~30 minutes)
- **Consensus:** XDPoS (Delegated Proof of Stake)
- **P2P Port:** 30304 (default for XDC)

### XDC Apothem Testnet
- **Chain ID:** 51
- **Network ID:** 51
- **Block Time:** 2 seconds
- **Epoch:** 900 blocks (~30 minutes)

## Running a Node

### Full Sync (XDC Mainnet)

```bash
# Initialize with genesis (first time only)
./build/bin/XDC init genesis/xdc_mainnet.json --datadir ~/.xdc

# Start sync
./build/bin/XDC \
  --networkid 50 \
  --datadir ~/.xdc \
  --syncmode full \
  --port 30304 \
  --bootnodes "enode://91e59fa1b034ae35e9f4e8a99cc6621f09d74e76a6220abb6c93b29ed41a9e1fc4e5b70e2c5fc43f883cffbdcd6f4f6cbc1d23af077f28c2aecc22403355d4b1@81.0.220.137:30304,enode://91e59fa1b034ae35e9f4e8a99cc6621f09d74e76a6220abb6c93b29ed41a9e1fc4e5b70e2c5fc43f883cffbdcd6f4f6cbc1d23af077f28c2aecc22403355d4b1@5.189.144.192:30304,enode://91e59fa1b034ae35e9f4e8a99cc6621f09d74e76a6220abb6c93b29ed41a9e1fc4e5b70e2c5fc43f883cffbdcd6f4f6cbc1d23af077f28c2aecc22403355d4b1@154.53.42.5:30304"
```

### Light Sync

```bash
./build/bin/XDC \
  --networkid 50 \
  --datadir ~/.xdc-light \
  --syncmode light \
  --port 30305 \
  --bootnodes "enode://91e59fa1b034ae35e9f4e8a99cc6621f09d74e76a6220abb6c93b29ed41a9e1fc4e5b70e2c5fc43f883cffbdcd6f4f6cbc1d23af077f28c2aecc22403355d4b1@81.0.220.137:30304"
```

### Apothem Testnet

```bash
# Initialize with genesis
./build/bin/XDC init genesis/xdc_apothem.json --datadir ~/.xdc-testnet

# Start sync
./build/bin/XDC \
  --networkid 51 \
  --datadir ~/.xdc-testnet \
  --syncmode full \
  --port 30304
```

## Verifying Sync

### Check Sync Status

```bash
# Attach to running node
./build/bin/XDC attach ~/.xdc/XDC.ipc

# In console
> eth.syncing
> eth.blockNumber
> admin.peers
```

### Expected Output

A syncing node should show:
```javascript
> eth.syncing
{
  currentBlock: 1234567,
  highestBlock: 9876543,
  knownStates: 0,
  pulledStates: 0,
  startingBlock: 0
}

> admin.peers.length
5  // Should have several peers
```

### Check Consensus

```javascript
// Get current block
> eth.getBlock("latest")

// Verify XDPoS fields
> eth.getBlock("latest").difficulty
2  // Should be 1 or 2 (in-turn/out-of-turn)

// Check block time
> eth.getBlock("latest").timestamp - eth.getBlock(eth.blockNumber - 1).timestamp
2  // Should be ~2 seconds
```

## RPC Endpoints

Enable RPC for external access:

```bash
./build/bin/XDC \
  --networkid 50 \
  --datadir ~/.xdc \
  --http \
  --http.addr "0.0.0.0" \
  --http.port 8545 \
  --http.api "eth,net,web3,xdpos" \
  --http.corsdomain "*" \
  --ws \
  --ws.addr "0.0.0.0" \
  --ws.port 8546 \
  --ws.api "eth,net,web3,xdpos"
```

## XDPoS-Specific APIs

The XDPoS consensus engine exposes additional RPC methods:

```javascript
// Get current masternodes
> xdpos.getMasternodes()

// Get snapshot at specific block
> xdpos.getSnapshot(blockNumber)

// Check if address is a masternode
> xdpos.isValidMasternode(address)

// Get signers for a specific block
> xdpos.getSigners(blockNumber)
```

## Troubleshooting

### No Peers

1. Check firewall allows port 30304 (TCP/UDP)
2. Verify bootnodes are reachable: `nc -vz 81.0.220.137 30304`
3. Try adding more bootnodes from the network

### Sync Stalled

1. Check disk space: `df -h`
2. Check memory usage: `free -m`
3. Restart with `--cache 4096` for more caching

### Genesis Mismatch

If you see "genesis block mismatch" errors:
```bash
# Remove old data and reinitialize
rm -rf ~/.xdc/XDC
./build/bin/XDC init genesis/xdc_mainnet.json --datadir ~/.xdc
```

### Block Validation Errors

Check logs for specific errors:
```bash
./build/bin/XDC --verbosity 4 ...  # Enable debug logging
```

Common issues:
- **"unauthorized"**: Block signer not in masternode list
- **"invalid difficulty"**: Difficulty calculation mismatch
- **"invalid timestamp"**: Block time too close to parent

## Known Limitations

### Current Implementation Status

1. **Reward Distribution**: Basic implementation - distributes rewards at epoch checkpoints
2. **Penalty System**: Tracks missed blocks, penalizes inactive masternodes
3. **Contract Integration**: Hook system in place for validator contract calls
4. **Double Validation**: Signature verification implemented

### What's Working

- ✅ Basic block validation
- ✅ Signature recovery
- ✅ Epoch transitions
- ✅ Masternode list from checkpoint headers
- ✅ Difficulty calculation
- ✅ Block sealing

### Needs Live Network Testing

- ⚠️ Smart contract state sync (0x88 validator contract)
- ⚠️ Full reward distribution with voter rewards
- ⚠️ Cross-epoch penalty persistence
- ⚠️ Network sync with production bootnodes

## Performance Tuning

For production deployments:

```bash
./build/bin/XDC \
  --networkid 50 \
  --datadir /data/xdc \
  --syncmode full \
  --cache 8192 \
  --maxpeers 100 \
  --txpool.globalslots 16384 \
  --txpool.globalqueue 4096
```

## Security Considerations

1. **RPC Access**: Never expose RPC without authentication in production
2. **Key Storage**: Use hardware wallets or encrypted keystores for masternodes
3. **Firewall**: Only allow necessary ports (30304 for P2P, 8545/8546 for RPC if needed)

## Resources

- [XDC Network Documentation](https://docs.xdc.org/)
- [XDPoSChain Reference Implementation](https://github.com/XinFinOrg/XDPoSChain)
- [XDC Block Explorer](https://explorer.xinfin.network/)
- [Apothem Faucet](https://faucet.apothem.network/)

## Reporting Issues

If you encounter issues:

1. Enable debug logging: `--verbosity 5`
2. Capture logs around the error
3. Note the block number where it failed
4. Open an issue with logs and system info
