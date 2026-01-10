#!/usr/bin/env python3
"""
Main Trading Agent
Integrates AI decision-making with blockchain execution
"""

import os
import sys
import time
import signal
from datetime import datetime
from typing import Optional

from ai_agent import DeFiTradingAgent
from blockchain_connector import BlockchainConnector
from config import Config


class TradingAgent:
    """
    Main autonomous trading agent that combines AI and blockchain
    """

    def __init__(self, config: Config):
        """Initialize trading agent with configuration"""
        self.config = config
        self.running = False
        self.trades_today = 0
        self.start_time = datetime.now()

        # Initialize AI agent
        print("Initializing AI agent...")
        self.ai_agent = DeFiTradingAgent(
            learning_rate=config.LEARNING_RATE,
            gamma=config.GAMMA,
            epsilon=config.EPSILON
        )

        # Load existing model if available
        if os.path.exists(config.MODEL_PATH):
            self.ai_agent.load_model(config.MODEL_PATH)
            print(f"✓ Loaded trained model from {config.MODEL_PATH}")
        else:
            print("No existing model found, training new agent...")
            self.ai_agent.train_on_simulated_data(episodes=1000, verbose=False)
            os.makedirs(os.path.dirname(config.MODEL_PATH), exist_ok=True)
            self.ai_agent.save_model(config.MODEL_PATH)

        # Initialize blockchain connector
        print("\nInitializing blockchain connector...")
        self.blockchain = BlockchainConnector(
            rpc_url=config.RPC_URL,
            private_key=config.PRIVATE_KEY,
            dex_address=config.DEX_ADDRESS,
            token_address=config.TOKEN_ADDRESS,
            dex_abi_path=config.DEX_ABI_PATH,
            token_abi_path=config.TOKEN_ABI_PATH
        )

        # Set gas parameters
        self.blockchain.gas_price_gwei = config.GAS_PRICE_GWEI
        self.blockchain.max_gas = config.MAX_GAS

        # Trading state
        self.initial_balances = None
        self.trade_history = []

        print("\n✓ Trading agent initialized successfully!")

    def start(self, mode: str = 'monitor'):
        """
        Start the trading agent

        Args:
            mode: 'monitor' (watch only), 'paper' (simulate trades), or 'live' (execute trades)
        """
        self.running = True
        self.mode = mode

        print(f"\n{'=' * 60}")
        print(f"AI DeFi Trading Agent Started - Mode: {mode.upper()}")
        print(f"{'=' * 60}\n")

        # Get initial state
        self.initial_balances = self.blockchain.get_balances()
        print(f"Initial balances:")
        print(f"  ETH: {self.initial_balances['eth']:.4f}")
        print(f"  Token: {self.initial_balances['token']:.4f}\n")

        # Setup signal handler for graceful shutdown
        signal.signal(signal.SIGINT, self._signal_handler)

        # Main trading loop
        try:
            self._trading_loop()
        except Exception as e:
            print(f"\n✗ Error in trading loop: {e}")
        finally:
            self._shutdown()

    def _trading_loop(self):
        """Main loop for monitoring and trading"""
        print(f"Starting trading loop (checking every {self.config.PRICE_CHECK_INTERVAL}s)...\n")

        iteration = 0
        while self.running:
            iteration += 1
            print(f"--- Iteration {iteration} @ {datetime.now().strftime('%H:%M:%S')} ---")

            try:
                # 1. Get current market state
                balances = self.blockchain.get_balances()
                current_rate = self.blockchain.get_current_rate()
                price_delta = self.blockchain.calculate_price_delta(
                    lookback_blocks=self.config.LOOKBACK_BLOCKS
                )

                print(f"Current rate: {current_rate:.2f} tokens/ETH")
                print(f"Price delta: {price_delta:.4f} ({price_delta * 100:.2f}%)")
                print(f"Balances - ETH: {balances['eth']:.4f}, Token: {balances['token']:.4f}")

                # 2. Get AI decision
                action_idx, action_name = self.ai_agent.get_action(price_delta, explore=False)
                print(f"AI Decision: {action_name.upper()}")

                # 3. Execute action based on mode
                if action_name == "buy" and balances['eth'] >= self.config.MIN_TRADE_AMOUNT_ETH:
                    self._execute_buy(price_delta)
                elif action_name == "sell" and balances['token'] > 0:
                    self._execute_sell(price_delta)
                else:
                    print(f"Action: {action_name.upper()} (holding position)")

                # 4. Check risk limits
                self._check_risk_limits()

                # 5. Display performance
                if iteration % 5 == 0:
                    self._display_performance()

            except Exception as e:
                print(f"✗ Error in iteration {iteration}: {e}")

            # Sleep until next check
            print()
            time.sleep(self.config.PRICE_CHECK_INTERVAL)

    def _execute_buy(self, price_delta: float):
        """Execute buy trade"""
        if self.trades_today >= self.config.MAX_DAILY_TRADES:
            print("⚠ Daily trade limit reached, skipping trade")
            return

        trade_amount = min(
            self.config.TRADE_AMOUNT_ETH,
            self.config.MAX_TRADE_AMOUNT_ETH
        )

        if self.mode == 'live':
            print(f"→ EXECUTING BUY: {trade_amount} ETH")
            tx_hash = self.blockchain.execute_buy_trade(trade_amount)
            if tx_hash:
                self.trades_today += 1
                self._record_trade('buy', trade_amount, price_delta, tx_hash)
        elif self.mode == 'paper':
            print(f"→ SIMULATED BUY: {trade_amount} ETH")
            self.trades_today += 1
            self._record_trade('buy', trade_amount, price_delta, 'simulated')
        else:
            print(f"→ WOULD BUY: {trade_amount} ETH (monitor mode)")

    def _execute_sell(self, price_delta: float):
        """Execute sell trade"""
        if self.trades_today >= self.config.MAX_DAILY_TRADES:
            print("⚠ Daily trade limit reached, skipping trade")
            return

        balances = self.blockchain.get_balances()
        sell_amount = min(balances['token'], balances['token'] * 0.5)  # Sell up to 50%

        if self.mode == 'live':
            print(f"→ EXECUTING SELL: {sell_amount:.4f} tokens")
            tx_hash = self.blockchain.execute_sell_trade(sell_amount)
            if tx_hash:
                self.trades_today += 1
                self._record_trade('sell', sell_amount, price_delta, tx_hash)
        elif self.mode == 'paper':
            print(f"→ SIMULATED SELL: {sell_amount:.4f} tokens")
            self.trades_today += 1
            self._record_trade('sell', sell_amount, price_delta, 'simulated')
        else:
            print(f"→ WOULD SELL: {sell_amount:.4f} tokens (monitor mode)")

    def _record_trade(self, trade_type: str, amount: float, price_delta: float, tx_hash: str):
        """Record trade in history"""
        trade = {
            'timestamp': datetime.now().isoformat(),
            'type': trade_type,
            'amount': amount,
            'price_delta': price_delta,
            'tx_hash': tx_hash
        }
        self.trade_history.append(trade)

    def _check_risk_limits(self):
        """Check stop-loss and take-profit conditions"""
        if not self.initial_balances:
            return

        current = self.blockchain.get_balances()
        current_rate = self.blockchain.get_current_rate()

        # Calculate portfolio value in ETH
        initial_value = self.initial_balances['eth'] + self.initial_balances['token'] / current_rate
        current_value = current['eth'] + current['token'] / current_rate
        pnl_pct = (current_value - initial_value) / initial_value if initial_value > 0 else 0

        if pnl_pct <= self.config.STOP_LOSS_THRESHOLD:
            print(f"\n⚠ STOP LOSS TRIGGERED: {pnl_pct * 100:.2f}% loss")
            self.running = False

        elif pnl_pct >= self.config.TAKE_PROFIT_THRESHOLD:
            print(f"\n✓ TAKE PROFIT TRIGGERED: {pnl_pct * 100:.2f}% gain")
            self.running = False

    def _display_performance(self):
        """Display current performance metrics"""
        if not self.initial_balances:
            return

        print("\n" + "=" * 60)
        print("PERFORMANCE SUMMARY")
        print("=" * 60)

        current = self.blockchain.get_balances()
        current_rate = self.blockchain.get_current_rate()

        initial_value = self.initial_balances['eth'] + self.initial_balances['token'] / current_rate
        current_value = current['eth'] + current['token'] / current_rate
        pnl = current_value - initial_value
        pnl_pct = (pnl / initial_value * 100) if initial_value > 0 else 0

        print(f"Trades today: {self.trades_today}/{self.config.MAX_DAILY_TRADES}")
        print(f"Initial value: {initial_value:.4f} ETH")
        print(f"Current value: {current_value:.4f} ETH")
        print(f"P&L: {pnl:+.4f} ETH ({pnl_pct:+.2f}%)")
        print(f"Runtime: {(datetime.now() - self.start_time).total_seconds() / 60:.1f} minutes")
        print("=" * 60 + "\n")

    def _signal_handler(self, signum, frame):
        """Handle shutdown signal"""
        print("\n\nReceived shutdown signal...")
        self.running = False

    def _shutdown(self):
        """Graceful shutdown"""
        print("\n" + "=" * 60)
        print("SHUTTING DOWN")
        print("=" * 60)

        # Display final performance
        self._display_performance()

        # Display trade history
        if self.trade_history:
            print("\nTrade History:")
            for i, trade in enumerate(self.trade_history, 1):
                print(f"{i}. {trade['type'].upper()} - {trade['amount']:.4f} @ {trade['timestamp']}")

        # Get final DEX stats
        try:
            stats = self.blockchain.get_dex_stats()
            print("\nDEX Statistics:")
            print(f"  Total swaps: {stats['total_swaps']}")
            print(f"  Total ETH traded: {stats['total_eth_swapped']:.4f}")
            print(f"  Total tokens traded: {stats['total_tokens_swapped']:.4f}")
        except Exception as e:
            print(f"Could not fetch DEX stats: {e}")

        print("\n✓ Agent shutdown complete")


def main():
    """Main entry point"""
    print("\n" + "=" * 60)
    print("AI DeFi Trading Agent")
    print("Hybrid Architecture: AI Planning + On-Chain Execution")
    print("=" * 60 + "\n")

    # Validate configuration
    if not Config.validate():
        print("\n✗ Configuration validation failed!")
        print("\nPlease set the following environment variables:")
        print("  - AGENT_PRIVATE_KEY")
        print("  - DEX_ADDRESS")
        print("  - TOKEN_ADDRESS")
        print("\nOr create a .env file (see config.py for template)")
        sys.exit(1)

    Config.print_config()

    # Get mode from command line
    mode = sys.argv[1] if len(sys.argv) > 1 else 'monitor'
    if mode not in ['monitor', 'paper', 'live']:
        print(f"Invalid mode: {mode}")
        print("Usage: python trading_agent.py [monitor|paper|live]")
        sys.exit(1)

    if mode == 'live':
        confirm = input("\n⚠ WARNING: Running in LIVE mode will execute real trades!\nType 'yes' to continue: ")
        if confirm.lower() != 'yes':
            print("Aborted.")
            sys.exit(0)

    # Create and start agent
    try:
        agent = TradingAgent(Config)
        agent.start(mode=mode)
    except Exception as e:
        print(f"\n✗ Fatal error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
