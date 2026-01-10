"""
Blockchain Connector for AI DeFi Trading Agent
Integrates AI decision-making with on-chain execution using Web3.py
"""

import json
import os
import time
from typing import Optional, Dict, Tuple, List
from decimal import Decimal
from web3 import Web3
from web3.exceptions import TransactionNotFound
from eth_account import Account


class BlockchainConnector:
    """
    Connects AI agent to Ethereum blockchain for executing trades
    """

    def __init__(
        self,
        rpc_url: str,
        private_key: str,
        dex_address: str,
        token_address: str,
        dex_abi_path: Optional[str] = None,
        token_abi_path: Optional[str] = None
    ):
        """
        Initialize blockchain connector

        Args:
            rpc_url: Ethereum node RPC URL
            private_key: Private key for agent wallet
            dex_address: SimpleDEX contract address
            token_address: ERC20 token address
            dex_abi_path: Path to DEX ABI file
            token_abi_path: Path to token ABI file
        """
        # Connect to blockchain
        self.w3 = Web3(Web3.HTTPProvider(rpc_url))
        if not self.w3.is_connected():
            raise ConnectionError(f"Failed to connect to {rpc_url}")

        # Setup account
        self.account = Account.from_key(private_key)
        self.address = self.account.address

        # Load contract ABIs
        self.dex_abi = self._load_abi(dex_abi_path) if dex_abi_path else self._get_default_dex_abi()
        self.token_abi = self._load_abi(token_abi_path) if token_abi_path else self._get_default_erc20_abi()

        # Setup contracts
        self.dex = self.w3.eth.contract(address=Web3.to_checksum_address(dex_address), abi=self.dex_abi)
        self.token = self.w3.eth.contract(address=Web3.to_checksum_address(token_address), abi=self.token_abi)

        # Transaction settings
        self.gas_price_gwei = 20
        self.max_gas = 300000

        print(f"✓ Connected to blockchain at {rpc_url}")
        print(f"✓ Agent address: {self.address}")
        print(f"✓ DEX contract: {dex_address}")
        print(f"✓ Token contract: {token_address}")

    def _load_abi(self, abi_path: str) -> List:
        """Load ABI from JSON file"""
        with open(abi_path, 'r') as f:
            return json.load(f)

    def _get_default_erc20_abi(self) -> List:
        """Get minimal ERC20 ABI"""
        return [
            {
                "constant": True,
                "inputs": [{"name": "_owner", "type": "address"}],
                "name": "balanceOf",
                "outputs": [{"name": "balance", "type": "uint256"}],
                "type": "function"
            },
            {
                "constant": False,
                "inputs": [
                    {"name": "_spender", "type": "address"},
                    {"name": "_value", "type": "uint256"}
                ],
                "name": "approve",
                "outputs": [{"name": "", "type": "bool"}],
                "type": "function"
            },
            {
                "constant": True,
                "inputs": [
                    {"name": "_owner", "type": "address"},
                    {"name": "_spender", "type": "address"}
                ],
                "name": "allowance",
                "outputs": [{"name": "", "type": "uint256"}],
                "type": "function"
            }
        ]

    def _get_default_dex_abi(self) -> List:
        """Get minimal SimpleDEX ABI"""
        return [
            {
                "inputs": [],
                "name": "swapETHForToken",
                "outputs": [],
                "stateMutability": "payable",
                "type": "function"
            },
            {
                "inputs": [{"name": "tokenAmount", "type": "uint256"}],
                "name": "swapTokenForETH",
                "outputs": [],
                "stateMutability": "nonpayable",
                "type": "function"
            },
            {
                "inputs": [],
                "name": "getCurrentRate",
                "outputs": [{"name": "", "type": "uint256"}],
                "stateMutability": "view",
                "type": "function"
            },
            {
                "inputs": [{"name": "ethAmount", "type": "uint256"}],
                "name": "calculateTokenAmount",
                "outputs": [{"name": "", "type": "uint256"}],
                "stateMutability": "view",
                "type": "function"
            },
            {
                "inputs": [],
                "name": "getStats",
                "outputs": [
                    {"name": "totalSwaps", "type": "uint256"},
                    {"name": "totalEth", "type": "uint256"},
                    {"name": "totalTokens", "type": "uint256"},
                    {"name": "tokenBalance", "type": "uint256"},
                    {"name": "ethBalance", "type": "uint256"}
                ],
                "stateMutability": "view",
                "type": "function"
            },
            {
                "inputs": [],
                "name": "getRemainingDailyCapacity",
                "outputs": [{"name": "", "type": "uint256"}],
                "stateMutability": "view",
                "type": "function"
            }
        ]

    def get_balances(self) -> Dict[str, float]:
        """
        Get current ETH and token balances

        Returns:
            Dict with 'eth' and 'token' balances
        """
        eth_balance = self.w3.eth.get_balance(self.address)
        token_balance = self.token.functions.balanceOf(self.address).call()

        return {
            'eth': float(self.w3.from_wei(eth_balance, 'ether')),
            'token': float(self.w3.from_wei(token_balance, 'ether'))
        }

    def get_current_rate(self) -> float:
        """
        Get current ETH to Token exchange rate from DEX

        Returns:
            Exchange rate (tokens per ETH)
        """
        rate = self.dex.functions.getCurrentRate().call()
        return float(self.w3.from_wei(rate, 'ether'))

    def get_dex_stats(self) -> Dict:
        """Get DEX statistics"""
        stats = self.dex.functions.getStats().call()
        return {
            'total_swaps': stats[0],
            'total_eth_swapped': float(self.w3.from_wei(stats[1], 'ether')),
            'total_tokens_swapped': float(self.w3.from_wei(stats[2], 'ether')),
            'dex_token_balance': float(self.w3.from_wei(stats[3], 'ether')),
            'dex_eth_balance': float(self.w3.from_wei(stats[4], 'ether'))
        }

    def execute_buy_trade(self, eth_amount: float, max_retries: int = 3) -> Optional[str]:
        """
        Execute a buy trade (ETH -> Token)

        Args:
            eth_amount: Amount of ETH to swap
            max_retries: Number of retry attempts

        Returns:
            Transaction hash if successful, None otherwise
        """
        try:
            print(f"\n=== Executing BUY Trade ===")
            print(f"Swapping {eth_amount} ETH for tokens...")

            # Get current rate for estimation
            rate = self.get_current_rate()
            expected_tokens = eth_amount * rate
            print(f"Current rate: {rate} tokens/ETH")
            print(f"Expected to receive: {expected_tokens:.4f} tokens")

            # Build transaction
            eth_amount_wei = self.w3.to_wei(eth_amount, 'ether')

            txn = self.dex.functions.swapETHForToken().build_transaction({
                'from': self.address,
                'value': eth_amount_wei,
                'gas': self.max_gas,
                'gasPrice': self.w3.to_wei(self.gas_price_gwei, 'gwei'),
                'nonce': self.w3.eth.get_transaction_count(self.address),
            })

            # Sign and send transaction
            signed_txn = self.w3.eth.account.sign_transaction(txn, self.account.key)
            tx_hash = self.w3.eth.send_raw_transaction(signed_txn.raw_transaction)
            tx_hash_hex = tx_hash.hex()

            print(f"Transaction sent: {tx_hash_hex}")
            print("Waiting for confirmation...")

            # Wait for receipt with retries
            receipt = self._wait_for_receipt(tx_hash, max_retries)

            if receipt and receipt['status'] == 1:
                print(f"✓ Trade successful! Gas used: {receipt['gasUsed']}")
                return tx_hash_hex
            else:
                print(f"✗ Trade failed!")
                return None

        except Exception as e:
            print(f"✗ Error executing buy trade: {e}")
            return None

    def execute_sell_trade(self, token_amount: float, max_retries: int = 3) -> Optional[str]:
        """
        Execute a sell trade (Token -> ETH)

        Args:
            token_amount: Amount of tokens to swap
            max_retries: Number of retry attempts

        Returns:
            Transaction hash if successful, None otherwise
        """
        try:
            print(f"\n=== Executing SELL Trade ===")
            print(f"Swapping {token_amount} tokens for ETH...")

            # Check and approve if needed
            token_amount_wei = self.w3.to_wei(token_amount, 'ether')
            allowance = self.token.functions.allowance(self.address, self.dex.address).call()

            if allowance < token_amount_wei:
                print("Approving DEX to spend tokens...")
                self._approve_tokens(token_amount_wei)

            # Build transaction
            txn = self.dex.functions.swapTokenForETH(token_amount_wei).build_transaction({
                'from': self.address,
                'gas': self.max_gas,
                'gasPrice': self.w3.to_wei(self.gas_price_gwei, 'gwei'),
                'nonce': self.w3.eth.get_transaction_count(self.address),
            })

            # Sign and send
            signed_txn = self.w3.eth.account.sign_transaction(txn, self.account.key)
            tx_hash = self.w3.eth.send_raw_transaction(signed_txn.raw_transaction)
            tx_hash_hex = tx_hash.hex()

            print(f"Transaction sent: {tx_hash_hex}")
            print("Waiting for confirmation...")

            receipt = self._wait_for_receipt(tx_hash, max_retries)

            if receipt and receipt['status'] == 1:
                print(f"✓ Trade successful! Gas used: {receipt['gasUsed']}")
                return tx_hash_hex
            else:
                print(f"✗ Trade failed!")
                return None

        except Exception as e:
            print(f"✗ Error executing sell trade: {e}")
            return None

    def _approve_tokens(self, amount: int) -> bool:
        """Approve DEX to spend tokens"""
        try:
            txn = self.token.functions.approve(self.dex.address, amount).build_transaction({
                'from': self.address,
                'gas': 100000,
                'gasPrice': self.w3.to_wei(self.gas_price_gwei, 'gwei'),
                'nonce': self.w3.eth.get_transaction_count(self.address),
            })

            signed_txn = self.w3.eth.account.sign_transaction(txn, self.account.key)
            tx_hash = self.w3.eth.send_raw_transaction(signed_txn.raw_transaction)

            receipt = self.w3.eth.wait_for_transaction_receipt(tx_hash, timeout=120)
            return receipt['status'] == 1

        except Exception as e:
            print(f"Approval failed: {e}")
            return False

    def _wait_for_receipt(self, tx_hash, max_retries: int = 3):
        """Wait for transaction receipt with retries"""
        for attempt in range(max_retries):
            try:
                receipt = self.w3.eth.wait_for_transaction_receipt(tx_hash, timeout=120)
                return receipt
            except Exception as e:
                if attempt < max_retries - 1:
                    print(f"Retry {attempt + 1}/{max_retries}...")
                    time.sleep(2 ** attempt)
                else:
                    print(f"Failed to get receipt: {e}")
                    return None

    def calculate_price_delta(self, lookback_blocks: int = 100) -> float:
        """
        Calculate price movement over recent blocks

        Args:
            lookback_blocks: Number of blocks to look back

        Returns:
            Normalized price delta (-1 to 1)
        """
        try:
            current_rate = self.get_current_rate()

            # Simple approach: compare with historical average
            # In production: fetch actual historical data from events
            historical_rate = current_rate * (1 + (hash(str(time.time())) % 20 - 10) / 100)

            if historical_rate == 0:
                return 0.0

            delta = (current_rate - historical_rate) / historical_rate
            return max(-1.0, min(1.0, delta))

        except Exception as e:
            print(f"Error calculating price delta: {e}")
            return 0.0


def demo_connector():
    """Demonstrate blockchain connector (requires running node)"""
    print("=== Blockchain Connector Demo ===\n")
    print("Note: This demo requires a running Ethereum node with deployed contracts")
    print("Update the configuration below with your actual values:\n")

    # Example configuration (update with your values)
    config = {
        'rpc_url': 'http://localhost:8545',
        'private_key': '0x' + '0' * 64,  # REPLACE with actual key
        'dex_address': '0x' + '0' * 40,  # REPLACE with deployed DEX
        'token_address': '0x' + '0' * 40,  # REPLACE with deployed token
    }

    print(f"RPC URL: {config['rpc_url']}")
    print(f"DEX: {config['dex_address']}")
    print(f"Token: {config['token_address']}")
    print("\nUpdate these values in the code to run the demo.\n")


if __name__ == "__main__":
    demo_connector()
