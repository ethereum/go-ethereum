# AI Agent for Autonomous DeFi Trading

A production-inspired prototype that combines **off-chain AI intelligence** with **on-chain execution** for autonomous DeFi trading. This hybrid architecture uses reinforcement learning (Q-learning) for decision-making while maintaining transparency and security through smart contract execution.

## Overview

This project demonstrates a practical implementation of an AI-powered trading agent that:
- 📊 **Monitors** token prices via on-chain exchange rates (simulated oracle)
- 🤖 **Decides** on trades using a Q-learning neural network
- ⚡ **Executes** swaps transparently on a custom DEX smart contract
- 🔒 **Manages risk** with configurable limits and circuit breakers

### Hybrid Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     AI DeFi Trading Agent                    │
└─────────────────────────────────────────────────────────────┘
         │                                          │
         ▼                                          ▼
┌──────────────────────┐              ┌──────────────────────┐
│   OFF-CHAIN (AI)     │              │   ON-CHAIN (Smart   │
│                      │              │     Contracts)      │
│  • Q-Learning Model  │              │                     │
│  • Price Analysis    │◄────────────►│  • SimpleDEX        │
│  • Decision Making   │   Web3.py    │  • ERC20 Token      │
│  • Risk Management   │              │  • Liquidity Pools  │
└──────────────────────┘              └──────────────────────┘
   Python + PyTorch                      Solidity + EVM
```

**Why Hybrid?**
- **Flexibility**: AI logic can be updated without contract redeployment
- **Transparency**: All trades executed on-chain with full auditability
- **Security**: Smart contracts enforce limits and prevent manipulation
- **Efficiency**: Complex ML computations happen off-chain to save gas

## Features

### ✨ Core Capabilities

- **Reinforcement Learning**: Q-learning agent that learns optimal trading strategies
- **Multi-Mode Operation**: Monitor (watch-only), Paper (simulate), Live (execute)
- **Risk Management**: Daily limits, stop-loss, take-profit, and sanity checks
- **Real-Time Monitoring**: Continuous price tracking and portfolio analysis
- **Production-Ready Contracts**: SimpleDEX with security features and gas optimization
- **Comprehensive Logging**: Track all decisions, trades, and performance metrics

### 🛡️ Security Features

- Daily trading volume limits
- Min/max swap amount constraints
- Rate change restrictions (prevents oracle manipulation)
- Emergency withdrawal mechanism
- Configurable gas limits
- Private key environment isolation

## Project Structure

```
ai-defi-agent/
├── README.md                    # This file
├── DEPLOYMENT.md                # Deployment guide
├── requirements.txt             # Python dependencies
├── .env.example                 # Environment configuration template
├── .gitignore                   # Git ignore rules
│
├── contracts/                   # Solidity smart contracts
│   ├── SimpleDEX.sol           # DEX for token swaps
│   └── MockERC20.sol           # Test token (mock USDC)
│
├── src/                         # Python source code
│   ├── ai_agent.py             # Q-learning trading agent
│   ├── blockchain_connector.py # Web3.py integration
│   ├── config.py               # Configuration management
│   └── trading_agent.py        # Main agent orchestrator
│
├── tests/                       # Test files (future)
│   └── test_agent.py
│
├── models/                      # Trained AI models (created at runtime)
│   └── trading_agent.pth
│
└── logs/                        # Application logs (created at runtime)
    └── trading_agent.log
```

## Quick Start

### 1. Prerequisites

- **Ethereum Node**: Running go-ethereum node (local or testnet)
- **Python**: 3.8 or higher
- **Wallet**: Funded with ETH for gas fees

### 2. Installation

```bash
# Navigate to the project directory
cd ai-defi-agent

# Create and activate virtual environment
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt
```

### 3. Configuration

```bash
# Copy environment template
cp .env.example .env

# Edit with your values
nano .env
```

Required variables:
- `ETH_RPC_URL`: Your Ethereum node URL (e.g., `http://localhost:8545`)
- `AGENT_PRIVATE_KEY`: Private key for the agent wallet
- `DEX_ADDRESS`: SimpleDEX contract address (after deployment)
- `TOKEN_ADDRESS`: MockERC20 token address (after deployment)

### 4. Deploy Contracts

See [DEPLOYMENT.md](DEPLOYMENT.md) for detailed deployment instructions.

Quick deploy using Remix:
1. Open https://remix.ethereum.org
2. Deploy `MockERC20.sol` (1M tokens, 18 decimals)
3. Deploy `SimpleDEX.sol` (with token address, rate 1000)
4. Add liquidity to DEX
5. Update `.env` with deployed addresses

### 5. Run the Agent

**Monitor Mode** (watch only, no trades):
```bash
cd src
python trading_agent.py monitor
```

**Paper Trading Mode** (simulate trades):
```bash
python trading_agent.py paper
```

**Live Trading Mode** (execute real trades):
```bash
python trading_agent.py live
```

## How It Works

### 1. Perception (Price Monitoring)

The agent continuously monitors the DEX exchange rate:
```python
current_rate = blockchain.get_current_rate()  # e.g., 1000 tokens/ETH
price_delta = calculate_price_delta(current_rate, historical_avg)
```

### 2. Planning (AI Decision Making)

A Q-learning neural network processes the price delta and outputs an action:
```python
state = normalize_price_delta(price_delta)  # -1 to 1
action = ai_agent.get_action(state)  # 0=hold, 1=buy, 2=sell
```

**Q-Learning Model**:
- **State**: Normalized price change (-1 to 1)
- **Actions**: Hold, Buy, Sell
- **Reward**: Positive for profitable trades, negative for losses
- **Network**: 2-layer MLP with ReLU activation

### 3. Action (On-Chain Execution)

Based on the AI decision, the agent executes trades:
```python
if action == "buy":
    tx_hash = blockchain.execute_buy_trade(amount_eth)
elif action == "sell":
    tx_hash = blockchain.execute_sell_trade(amount_tokens)
```

### 4. Learning (Continuous Improvement)

The agent can be retrained with actual trading outcomes:
```python
reward = calculate_reward(trade_outcome)
ai_agent.train_step(state, action, reward, next_state)
```

## Configuration

Edit `src/config.py` or set environment variables:

### Trading Parameters
```python
MIN_TRADE_AMOUNT_ETH = 0.01   # Minimum trade size
MAX_TRADE_AMOUNT_ETH = 1.0    # Maximum trade size
TRADE_AMOUNT_ETH = 0.1        # Default trade size
```

### Risk Management
```python
MAX_DAILY_TRADES = 10              # Trades per day limit
STOP_LOSS_THRESHOLD = -0.15        # Stop at 15% loss
TAKE_PROFIT_THRESHOLD = 0.25       # Take profit at 25% gain
```

### AI Learning
```python
LEARNING_RATE = 0.001   # Neural network learning rate
GAMMA = 0.99            # Discount factor for future rewards
EPSILON = 0.1           # Exploration rate (10% random actions)
```

### Monitoring
```python
PRICE_CHECK_INTERVAL = 60   # Seconds between price checks
LOOKBACK_BLOCKS = 100       # Historical blocks for analysis
```

## Smart Contracts

### SimpleDEX

A secure DEX for ETH ↔ Token swaps with:
- Fixed exchange rate (upgradeable for oracle integration)
- Daily volume limits
- Min/max swap constraints
- Liquidity management
- Emergency controls

**Key Functions**:
```solidity
swapETHForToken() payable          // Buy tokens with ETH
swapTokenForETH(uint256)           // Sell tokens for ETH
getCurrentRate() view              // Get exchange rate
updateRate(uint256)                // Update rate (owner only)
addLiquidity(uint256)              // Add token liquidity
```

### MockERC20

Standard ERC20 token for testing:
- 18 decimals
- Minting capability
- Full ERC20 compliance

## Real-World Inspirations

This prototype draws from production patterns:

1. **Hybrid Architecture**: Similar to Coinbase's AgentKit [[1]](https://pub.towardsai.net/coinbases-agentkit-revolutionizing-crypto-agents-with-tool-based-frameworks-2b6fa748f0bd), separates AI planning from execution
2. **Tool-Use Pattern**: AI decides, smart contracts execute (inspired by agent frameworks)
3. **Security First**: Multi-sig ready, oracle-compatible (Chainlink integration path)
4. **Production Scaling**: Can integrate with keeper networks (Gelato, Chainlink Automation)

## Security Considerations

⚠️ **Important**: This is a prototype for learning and experimentation.

### Before Production Use:
1. **Audit Smart Contracts**: Professional security audit required
2. **Oracle Integration**: Replace simulated oracle with Chainlink or similar
3. **Multi-Sig Wallet**: Use Gnosis Safe or ERC-4337 account abstraction
4. **Formal Verification**: Verify critical contract logic
5. **MEV Protection**: Implement flashbot bundles or private RPCs
6. **Monitoring**: 24/7 alerting and circuit breakers
7. **Insurance**: Consider DeFi insurance protocols

### Current Limitations:
- Simulated oracle (use Chainlink in production)
- Single-signature wallet (add multi-sig)
- Basic price analysis (enhance with TWAP, volatility metrics)
- No MEV protection
- Fixed trading strategy (add strategy patterns)

## Extending the Agent

### Add Chainlink Oracle

Replace simulated price with real oracle:
```solidity
import "@chainlink/contracts/src/v0.8/interfaces/AggregatorV3Interface.sol";

function getLatestPrice() public view returns (int) {
    (,int price,,,) = priceFeed.latestRoundData();
    return price;
}
```

### Enhance AI Model

Upgrade to LSTM for time-series:
```python
class LSTMTradingAgent(nn.Module):
    def __init__(self, input_size, hidden_size, num_layers):
        super().__init__()
        self.lstm = nn.LSTM(input_size, hidden_size, num_layers)
        self.fc = nn.Linear(hidden_size, 3)  # 3 actions
```

### Add Multi-Agent System

Create specialized agents:
- **Data Agent**: Aggregates price feeds
- **Analysis Agent**: Technical/sentiment analysis
- **Execution Agent**: Trade execution and monitoring

### Integrate ERC-4337

Use account abstraction for gas-less trades:
```solidity
// Use account abstraction for bundled transactions
// See: https://eips.ethereum.org/EIPS/eip-4337
```

## Testing

### Unit Tests (Future)
```bash
pytest tests/test_agent.py -v
```

### Manual Testing Workflow
1. Deploy contracts on local testnet
2. Run agent in monitor mode for 1 hour
3. Run in paper mode for 24 hours
4. Analyze trade decisions and performance
5. Only then consider live mode with minimal funds

## Performance Metrics

The agent tracks:
- **Total Trades**: Number of executed trades
- **Win Rate**: Percentage of profitable trades
- **P&L**: Profit and loss in ETH and percentage
- **Sharpe Ratio**: Risk-adjusted returns (future)
- **Gas Costs**: Total gas spent on trades
- **Uptime**: Agent runtime and availability

## Troubleshooting

### Common Issues

**"Connection refused"**
```bash
# Check if your Ethereum node is running
curl http://localhost:8545 -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

**"Insufficient balance"**
- Ensure wallet has ETH for gas
- Check DEX has enough liquidity
- Verify trade amount within limits

**"Transaction reverted"**
- Check daily limits not exceeded
- Verify rate hasn't been manipulated
- Ensure approvals are set correctly

See [DEPLOYMENT.md](DEPLOYMENT.md) for more troubleshooting.

## Roadmap

### Phase 1: Foundation (Current)
- ✅ Basic Q-learning agent
- ✅ SimpleDEX implementation
- ✅ Web3 integration
- ✅ Multi-mode operation

### Phase 2: Enhanced AI
- [ ] LSTM price prediction
- [ ] Sentiment analysis integration
- [ ] Multi-factor decision making
- [ ] Reinforcement learning from real trades

### Phase 3: Production Features
- [ ] Chainlink oracle integration
- [ ] Multi-sig wallet support
- [ ] Flashbot/MEV protection
- [ ] Advanced risk analytics

### Phase 4: Scaling
- [ ] Multi-DEX arbitrage
- [ ] Yield farming strategies
- [ ] Portfolio rebalancing
- [ ] DAO governance integration

## Contributing

Contributions welcome! Areas of interest:
- Enhanced AI models (LSTM, Transformers)
- Additional security features
- Integration with other DeFi protocols
- Testing and documentation

## License

MIT License - See LICENSE file

## Disclaimer

This software is provided for educational and research purposes only. Cryptocurrency trading involves significant risk. The authors are not responsible for any financial losses. Always test thoroughly on testnets before any production use. Never invest more than you can afford to lose.

## References

1. Hybrid AI-Blockchain Architectures: [LinkedIn - On-Chain Agents](https://www.linkedin.com/pulse/agents-onchain-myles-oneill)
2. Coinbase AgentKit: [Towards AI - AgentKit Framework](https://pub.towardsai.net/coinbases-agentkit-revolutionizing-crypto-agents-with-tool-based-frameworks-2b6fa748f0bd)
3. Keeper Networks: [Gelato Network Documentation](https://docs.gelato.network/)
4. ERC-4337 Account Abstraction: [Ethereum EIP-4337](https://eips.ethereum.org/EIPS/eip-4337)
5. Chainlink Price Feeds: [Chainlink Documentation](https://docs.chain.link/)

## Acknowledgments

Built for the go-ethereum ecosystem as a demonstration of hybrid AI-blockchain systems. Inspired by production DeFi agents and academic research in reinforcement learning for trading.

---

**Built with**: Python, PyTorch, Web3.py, Solidity, OpenZeppelin

**For questions or support**: See DEPLOYMENT.md or open an issue

Happy Trading! 🚀
