# Testing Whitelist Extra Rewards

This guide shows you how to verify that whitelisted validators receive **double block rewards**.

## Prerequisites

You can test this in two ways:

1. **Manual devnet setup (recommended for functional testing):**  
   Run two local `geth` processes (node1 + node2) with block production enabled.
2. **Docker setup (for structure / integration):**  
   Spin up two nodes via `docker-compose`.

### Manual Devnet Setup (no Docker)

These commands assume you are in a separate checkout of geth **v1.11.6**
in a directory like `~/Desktop/go-ethereum-v1.11`, and that you have a
`genesis.json` configured with the three accounts used in this test.

#### 0. Clone and build geth v1.11.6 (one-time)

```bash
cd ~/Desktop
git clone https://github.com/ethereum/go-ethereum.git go-ethereum-v1.11
cd go-ethereum-v1.11
git checkout v1.11.6
make geth
```

#### 1. Start node1 (whitelisted validator)

```bash
./build/bin/geth --datadir node1 init genesis.json

./build/bin/geth --datadir node1 \
  --networkid 1234 --nodiscover \
  --http --http.addr 0.0.0.0 --http.port 8545 \
  --port 30303 \
  --http.api eth,net,web3,admin,miner \
  --mine --miner.threads 1 \
  --miner.etherbase 0xca6b49ee60cdd276ab503fbd6fb80a3cfbc06ffc
```

#### 2. Start node2 (non-whitelisted validator)

```bash
./build/bin/geth --datadir node2 init genesis.json

./build/bin/geth --datadir node2 \
  --networkid 1234 --nodiscover \
  --http --http.addr 0.0.0.0 --http.port 8546 \
  --port 30304 \
  --http.api eth,net,web3,admin,miner \
  --mine --miner.threads 1 \
  --miner.etherbase 0xab52b2c71f61cd9447a932c0cb55d1752571dab8
```

#### 3. Verify blocks are being produced

```bash
./build/bin/geth attach --exec "eth.blockNumber" http://localhost:8545
```

The block number should increase over time.

#### 4. (Optional) Connect the two nodes explicitly

Get node1's enode:

```bash
./build/bin/geth attach --exec "admin.nodeInfo.enode" http://localhost:8545
```

Then on node2:

```bash
./build/bin/geth attach --exec "admin.addPeer('ENODE_HERE')" http://localhost:8546
```

Verify from node1:

```bash
./build/bin/geth attach --exec "admin.peers" http://localhost:8545
```

Once blocks are moving, you can run the Go test scripts from this repo (see below).

## Step-by-Step Testing

### Step 1: Check Initial State

First, let's see the current balances **before** whitelisting:

```bash
# Run the test script (this will show balances and wait for blocks)
go run ./test/test_rewards/test_rewards.go
```

This will show:
- Current block number
- Miner1 (Node1) balance
- Miner2 (Node2) balance
- Then wait for 10 blocks and show the results

**Expected before whitelisting**: Both miners should earn roughly the same per-block reward.

### Step 2: Whitelist Node1's Miner

Now, add Node1's miner address to the whitelist:

```bash
go run ./test/whitelist_validator/whitelist_tx.go
```

You should see:
```
Whitelisting transaction sent: 0x...
```

Wait for this transaction to be mined (check logs or wait ~15 seconds).

### Step 3: Verify Extra Rewards

Run the test script again:

```bash
go run ./test/test_rewards/test_rewards.go
```

**Expected after whitelisting**:
- **Miner1 (whitelisted)** should earn **~2x** the reward per block
- **Miner2 (not whitelisted)** should earn normal reward
- The ratio should be approximately **2.0x** (or close to it)

### Step 4: Manual Verification (Alternative Method)

If you want to verify manually using `geth attach`:

```bash
# Attach to node1
docker exec -it geth-node1 geth attach http://localhost:8545
```

In the JavaScript console:

```javascript
// Miner addresses
var miner1 = "0x26357d0353bEA3f89B654b14ccdc610720753F5E"; // Node1
var miner2 = "0xa6d864bb0D1F25EDD958c48E202F7a51b3E93424"; // Node2

// Check current balances
web3.fromWei(eth.getBalance(miner1), "ether");
web3.fromWei(eth.getBalance(miner2), "ether");

// Get current block number
eth.blockNumber;

// Wait for 5-10 blocks, then check again
// Miner1 should have increased by ~2x more than Miner2
```

### Step 5: Check Block Miners

To see which node mined which blocks:

```javascript
// Check last 10 blocks
for (var i = 0; i < 10; i++) {
    var block = eth.getBlock(eth.blockNumber - i);
    console.log("Block " + block.number + ": mined by " + block.miner);
}
```

## Expected Results

### Before Whitelisting:
- Miner1 per-block reward: ~5 ETH (Frontier reward)
- Miner2 per-block reward: ~5 ETH (Frontier reward)
- Ratio: ~1.0x

### After Whitelisting:
- Miner1 per-block reward: ~10 ETH (Frontier reward × 2)
- Miner2 per-block reward: ~5 ETH (Frontier reward)
- Ratio: ~2.0x ✅

## Troubleshooting

### If Miner1 is not getting extra rewards:

1. **Check if whitelist transaction was mined:**
   ```bash
   docker logs geth-node1 --tail 50 | grep -i "whitelist\|0x0100"
   ```

2. **Verify the precompile is registered:**
   - Check `core/vm/contracts.go` - should have whitelist precompile at `0x0100`

3. **Check consensus code:**
   - Verify `consensus/ethash/consensus.go` has the whitelist check in `accumulateRewards`

4. **Rebuild Docker images:**
   ```bash
   docker compose build --no-cache
   docker compose down
   docker compose up -d
   ```

### If both miners get the same reward:

- Make sure you ran `go run ./test/whitelist_validator/whitelist_tx.go` **after** starting the nodes
- Check that the transaction was successfully mined (look for the tx hash in logs)
- Verify Node1 is actually mining blocks (check `docker logs geth-node1`)

## Understanding the Reward Logic

The extra reward logic is in `consensus/ethash/consensus.go`:

1. During block creation, `accumulateRewards()` is called
2. It checks if the block's `coinbase` (miner) is whitelisted by calling the precompile at `0x0100`
3. If whitelisted, it adds **one extra block reward** on top of the normal reward
4. Result: **whitelisted miners get 2× the base block reward**
