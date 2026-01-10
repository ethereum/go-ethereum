"""
Configuration for AI DeFi Trading Agent
"""

import os
from typing import Dict, Any


class Config:
    """Central configuration for the trading agent"""

    # Blockchain settings
    RPC_URL = os.getenv('ETH_RPC_URL', 'http://localhost:8545')
    PRIVATE_KEY = os.getenv('AGENT_PRIVATE_KEY', '')  # Set via environment variable
    CHAIN_ID = int(os.getenv('CHAIN_ID', '1337'))  # Local chain default

    # Contract addresses (set after deployment)
    DEX_ADDRESS = os.getenv('DEX_ADDRESS', '')
    TOKEN_ADDRESS = os.getenv('TOKEN_ADDRESS', '')

    # ABI file paths (optional)
    DEX_ABI_PATH = os.getenv('DEX_ABI_PATH', None)
    TOKEN_ABI_PATH = os.getenv('TOKEN_ABI_PATH', None)

    # AI Agent parameters
    LEARNING_RATE = 0.001
    GAMMA = 0.99  # Discount factor
    EPSILON = 0.1  # Exploration rate
    MODEL_PATH = 'models/trading_agent.pth'

    # Trading parameters
    MIN_TRADE_AMOUNT_ETH = 0.01  # Minimum ETH per trade
    MAX_TRADE_AMOUNT_ETH = 1.0   # Maximum ETH per trade
    TRADE_AMOUNT_ETH = 0.1       # Default trade size

    # Price monitoring
    PRICE_CHECK_INTERVAL = 60  # Seconds between price checks
    LOOKBACK_BLOCKS = 100      # Blocks for price history

    # Risk management
    MAX_DAILY_TRADES = 10
    STOP_LOSS_THRESHOLD = -0.15  # Stop if portfolio drops 15%
    TAKE_PROFIT_THRESHOLD = 0.25  # Take profit at 25% gain

    # Gas settings
    GAS_PRICE_GWEI = 20
    MAX_GAS = 300000

    # Logging
    LOG_LEVEL = os.getenv('LOG_LEVEL', 'INFO')
    LOG_FILE = 'logs/trading_agent.log'

    @classmethod
    def validate(cls) -> bool:
        """Validate configuration"""
        if not cls.PRIVATE_KEY:
            print("ERROR: AGENT_PRIVATE_KEY not set")
            return False

        if not cls.DEX_ADDRESS:
            print("ERROR: DEX_ADDRESS not set")
            return False

        if not cls.TOKEN_ADDRESS:
            print("ERROR: TOKEN_ADDRESS not set")
            return False

        if len(cls.PRIVATE_KEY) != 66 or not cls.PRIVATE_KEY.startswith('0x'):
            print("ERROR: Invalid private key format")
            return False

        return True

    @classmethod
    def to_dict(cls) -> Dict[str, Any]:
        """Export configuration as dictionary"""
        return {
            'rpc_url': cls.RPC_URL,
            'chain_id': cls.CHAIN_ID,
            'dex_address': cls.DEX_ADDRESS,
            'token_address': cls.TOKEN_ADDRESS,
            'learning_rate': cls.LEARNING_RATE,
            'gamma': cls.GAMMA,
            'epsilon': cls.EPSILON,
            'min_trade_eth': cls.MIN_TRADE_AMOUNT_ETH,
            'max_trade_eth': cls.MAX_TRADE_AMOUNT_ETH,
            'price_check_interval': cls.PRICE_CHECK_INTERVAL,
        }

    @classmethod
    def print_config(cls):
        """Print current configuration"""
        print("\n=== Trading Agent Configuration ===")
        print(f"RPC URL: {cls.RPC_URL}")
        print(f"Chain ID: {cls.CHAIN_ID}")
        print(f"DEX Address: {cls.DEX_ADDRESS}")
        print(f"Token Address: {cls.TOKEN_ADDRESS}")
        print(f"Trade Amount: {cls.TRADE_AMOUNT_ETH} ETH")
        print(f"Price Check Interval: {cls.PRICE_CHECK_INTERVAL}s")
        print(f"Max Daily Trades: {cls.MAX_DAILY_TRADES}")
        print("===================================\n")


# Environment template for .env file
ENV_TEMPLATE = """
# AI DeFi Trading Agent Configuration
# Copy this to .env and fill in your values

# Blockchain Connection
ETH_RPC_URL=http://localhost:8545
CHAIN_ID=1337

# Wallet (NEVER commit this file with real keys!)
AGENT_PRIVATE_KEY=0x0000000000000000000000000000000000000000000000000000000000000000

# Deployed Contracts (fill after deployment)
DEX_ADDRESS=0x0000000000000000000000000000000000000000
TOKEN_ADDRESS=0x0000000000000000000000000000000000000000

# Optional: Custom ABI paths
# DEX_ABI_PATH=contracts/abi/SimpleDEX.json
# TOKEN_ABI_PATH=contracts/abi/MockERC20.json

# Logging
LOG_LEVEL=INFO
"""


if __name__ == "__main__":
    # Generate .env template
    print("Configuration module loaded")
    print("\nTo setup, create a .env file with:")
    print(ENV_TEMPLATE)
