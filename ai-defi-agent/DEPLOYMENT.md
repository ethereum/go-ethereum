# Deployment Guide for AI DeFi Trading Agent

This guide walks you through deploying and running the AI DeFi Trading Agent on your cloned Ethereum setup.

## Prerequisites

1. **Running Ethereum Node**: Your cloned go-ethereum node must be running
   ```bash
   # Start your local Ethereum node (from go-ethereum root)
   ./build/bin/geth --dev --http --http.api eth,web3,personal,net --http.corsdomain "*"
   ```

2. **Python Environment**: Python 3.8+ with pip
   ```bash
   python3 --version  # Should be 3.8 or higher
   ```

3. **Funded Wallet**: A wallet with some ETH for gas fees and trading

## Step 1: Environment Setup

### 1.1 Create Python Virtual Environment

```bash
cd ai-defi-agent
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
```

### 1.2 Install Dependencies

```bash
pip install -r requirements.txt
```

### 1.3 Configure Environment Variables

```bash
# Copy the example environment file
cp .env.example .env

# Edit .env with your values
nano .env  # or use your preferred editor
```

**Important**: Set at least these values in `.env`:
- `ETH_RPC_URL`: Your node URL (e.g., `http://localhost:8545`)
- `AGENT_PRIVATE_KEY`: Your wallet's private key (with 0x prefix)

## Step 2: Deploy Smart Contracts

You can deploy using Remix, Hardhat, or Foundry. Here's how to do it with Remix (easiest for testing):

### 2.1 Using Remix IDE

1. **Open Remix**: Go to https://remix.ethereum.org

2. **Create Files**:
   - Create `MockERC20.sol` and paste content from `contracts/MockERC20.sol`
   - Create `SimpleDEX.sol` and paste content from `contracts/SimpleDEX.sol`

3. **Compile**:
   - Select Solidity Compiler (0.8.x)
   - Compile both contracts

4. **Deploy MockERC20**:
   - Go to "Deploy & Run Transactions"
   - Select "Injected Provider - MetaMask" or "Web3 Provider" (http://localhost:8545)
   - Deploy `MockERC20` with parameters:
     - `_name`: "Mock USDC"
     - `_symbol`: "mUSDC"
     - `_decimals`: 18
     - `_initialSupply`: 1000000000000000000000000 (1 million tokens)
   - **Copy the deployed token address**

5. **Deploy SimpleDEX**:
   - Deploy `SimpleDEX` with parameters:
     - `_token`: [paste MockERC20 address]
     - `_initialRate`: 1000000000000000000000 (1000 tokens per ETH)
   - **Copy the deployed DEX address**

6. **Add Liquidity**:
   - Call `approve` on MockERC20 with:
     - `spender`: [SimpleDEX address]
     - `amount`: 500000000000000000000000 (500k tokens)
   - Call `addLiquidity` on SimpleDEX with:
     - `tokenAmount`: 500000000000000000000000 (500k tokens)
   - Send some ETH to SimpleDEX using the "Value" field and "Transact" button

### 2.2 Update Configuration

Update your `.env` file with the deployed addresses:

```bash
DEX_ADDRESS=0x... # SimpleDEX address from Remix
TOKEN_ADDRESS=0x... # MockERC20 address from Remix
```

## Step 3: Train the AI Agent (Optional)

The agent will auto-train on first run, but you can pre-train:

```bash
cd src
python ai_agent.py
```

This creates a trained model in `models/trading_agent.pth`.

## Step 4: Run the Agent

The agent has three modes:

### Monitor Mode (Read-Only)
Just watches prices and shows what it would do:
```bash
cd src
python trading_agent.py monitor
```

### Paper Trading Mode
Simulates trades without executing them:
```bash
python trading_agent.py paper
```

### Live Trading Mode
⚠️ **Executes real trades on-chain**:
```bash
python trading_agent.py live
```

## Step 5: Monitor Performance

The agent will display:
- Current market state (price, rate)
- AI decisions (buy/hold/sell)
- Trade execution results
- Performance metrics every 5 iterations

Example output:
```
--- Iteration 1 @ 14:30:25 ---
Current rate: 1000.00 tokens/ETH
Price delta: -0.0231 (-2.31%)
Balances - ETH: 10.0000, Token: 0.0000
AI Decision: BUY

=== Executing BUY Trade ===
Swapping 0.1 ETH for tokens...
Current rate: 1000.0 tokens/ETH
Expected to receive: 100.0000 tokens
Transaction sent: 0xabc123...
✓ Trade successful!
```

## Troubleshooting

### Connection Issues
```bash
# Check if your node is running
curl http://localhost:8545 -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

### Contract Deployment Fails
- Ensure your wallet has enough ETH
- Check that you're connected to the right network
- Verify Solidity compiler version (0.8.x)

### Transaction Failures
- Check gas prices in `config.py`
- Verify DEX has enough liquidity
- Ensure agent wallet has ETH for gas

### Import Errors
```bash
# Reinstall dependencies
pip install -r requirements.txt --upgrade
```

## Security Considerations

1. **Private Keys**: Never commit `.env` or share private keys
2. **Test First**: Always test on a local/testnet before mainnet
3. **Limited Funds**: Use a dedicated wallet with limited funds
4. **Monitor**: Watch the agent closely during initial runs
5. **Risk Limits**: Set appropriate `STOP_LOSS` and `MAX_TRADE_AMOUNT`

## Advanced Configuration

Edit `src/config.py` to customize:
- Trade amounts and limits
- Risk management thresholds
- AI learning parameters
- Gas price settings
- Monitoring intervals

## Next Steps

1. **Monitor Performance**: Run in paper mode for 24 hours
2. **Optimize Parameters**: Adjust thresholds based on results
3. **Enhance AI**: Add more sophisticated features (LSTM, sentiment analysis)
4. **Production Setup**:
   - Use Chainlink oracles for real price feeds
   - Add multi-sig for security
   - Implement ERC-4337 account abstraction
   - Deploy monitoring/alerting

## Useful Commands

```bash
# Check balances
python -c "from blockchain_connector import BlockchainConnector; \
from config import Config; \
bc = BlockchainConnector(Config.RPC_URL, Config.PRIVATE_KEY, \
Config.DEX_ADDRESS, Config.TOKEN_ADDRESS); \
print(bc.get_balances())"

# Get DEX stats
python -c "from blockchain_connector import BlockchainConnector; \
from config import Config; \
bc = BlockchainConnector(Config.RPC_URL, Config.PRIVATE_KEY, \
Config.DEX_ADDRESS, Config.TOKEN_ADDRESS); \
print(bc.get_dex_stats())"

# Check current rate
python -c "from blockchain_connector import BlockchainConnector; \
from config import Config; \
bc = BlockchainConnector(Config.RPC_URL, Config.PRIVATE_KEY, \
Config.DEX_ADDRESS, Config.TOKEN_ADDRESS); \
print(f'Rate: {bc.get_current_rate()} tokens/ETH')"
```

## Support

For issues:
1. Check the logs in `logs/trading_agent.log`
2. Review contract transactions on your block explorer
3. Verify configuration in `.env` and `config.py`
