# XDPoS Consensus Mechanism

## Overview

XDPoS (XinFin Delegated Proof of Stake) is a consensus mechanism designed for the XDC Network. It combines the benefits of Delegated Proof of Stake (DPoS) with practical Byzantine Fault Tolerance (pBFT) to achieve high throughput, low latency, and energy efficiency.

## Key Features

### 1. Validator Selection
- 108 Masternodes participate in block validation
- Validators are selected based on stake and voting
- Minimum stake requirement: 10,000,000 XDC
- Epoch-based validator set updates (900 blocks)

### 2. Block Production
- 2-second block time
- Round-robin block production among validators
- Block finality achieved through validator signatures

### 3. Consensus Versions

#### XDPoS v1 (Legacy)
- Simple round-robin block production
- Snapshot-based validator management
- Used until block X (network upgrade)

#### XDPoS v2 (Current)
- Enhanced BFT-style consensus
- Gap blocks for improved finality
- Vote aggregation for efficiency
- Penalty mechanism for misbehavior

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    XDPoS Engine                         │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │  Engine V1  │  │  Engine V2  │  │  Snapshot   │     │
│  │  (Legacy)   │  │  (Current)  │  │  Manager    │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │  Validator  │  │   Reward    │  │  Penalty    │     │
│  │  Selection  │  │  Calculator │  │  Handler    │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
└─────────────────────────────────────────────────────────┘
```

## Configuration

### Chain Config Parameters

```go
type XDPoSConfig struct {
    Period uint64 // Block time in seconds (default: 2)
    Epoch  uint64 // Epoch length in blocks (default: 900)
    Gap    uint64 // Gap for snapshots (default: 450)
    V2     *XDPoSV2Config
}
```

### Command Line Flags

```bash
--xdpos.rewards      Enable block rewards
--xdpos.slashing     Enable slashing for misbehavior
--xdpos.epoch        Epoch length (default: 900)
--xdpos.gap          Gap for snapshots (default: 450)
```

## Validator Operations

### Becoming a Masternode

1. Stake minimum 10,000,000 XDC
2. Run a full node with masternode configuration
3. Register through the validator contract

### Voting

- XDC holders can vote for validators
- Votes are weighted by stake
- Vote changes take effect at epoch boundaries

### Rewards

- Block rewards distributed to validators
- Rewards proportional to stake and participation
- Foundation wallet receives protocol fees

## Penalty System

### Penalties Applied For:
- Missing block production slot
- Double signing
- Invalid block proposals
- Extended downtime

### Penalty Amounts:
- Minor offense: Warning
- Repeated offense: Stake slash
- Major offense: Removal from validator set

## Network Synchronization

### Fast Sync
1. Download block headers
2. Verify validator signatures
3. Download state at checkpoint
4. Continue from checkpoint

### Full Sync
1. Download all blocks
2. Verify all transactions
3. Replay all state changes

## API Reference

### RPC Methods

```javascript
// Get current validators
xdpos.getValidators()

// Get validator info
xdpos.getValidatorInfo(address)

// Get snapshot at block
xdpos.getSnapshot(blockNumber)

// Get rewards for epoch
xdpos.getEpochRewards(epoch)
```

## Smart Contracts

### Validator Contract
- Address: `0x0000000000000000000000000000000000000088`
- Functions: register, vote, withdraw, getValidators

### Block Signer Contract
- Address: `0x0000000000000000000000000000000000000089`
- Functions: sign, getSigners, getRewardPercent

## Security Considerations

1. **Validator Key Security**: Keep masternode keys secure
2. **Network Security**: Use firewalls, restrict P2P ports
3. **Update Regularly**: Apply security patches promptly
4. **Monitor**: Watch for unusual block production patterns

## Troubleshooting

### Common Issues

1. **Not producing blocks**
   - Check if node is synced
   - Verify validator registration
   - Check stake requirement

2. **Sync issues**
   - Verify peer connections
   - Check network connectivity
   - Try different bootnodes

3. **Penalties received**
   - Check node uptime
   - Verify clock synchronization
   - Review logs for errors

## Further Reading

- [XDC Network Documentation](https://docs.xinfin.org)
- [Masternode Setup Guide](https://docs.xinfin.org/docs/masternode)
- [XDPoS Technical Paper](https://xinfin.org/xdpos)
