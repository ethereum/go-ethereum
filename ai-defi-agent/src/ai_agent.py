"""
AI Agent for Autonomous DeFi Trading
Uses Q-learning to make trading decisions based on price movements
"""

import numpy as np
import torch
import torch.nn as nn
import torch.optim as optim
import pickle
import os
from typing import Tuple, Optional


class QNetwork(nn.Module):
    """
    Q-Network for trading decisions
    States: normalized price delta
    Actions: 0=hold, 1=buy, 2=sell
    """
    def __init__(self, state_size: int = 1, action_size: int = 3, hidden_size: int = 64):
        super(QNetwork, self).__init__()
        self.fc1 = nn.Linear(state_size, hidden_size)
        self.fc2 = nn.Linear(hidden_size, hidden_size)
        self.fc3 = nn.Linear(hidden_size, action_size)

    def forward(self, x):
        x = torch.relu(self.fc1(x))
        x = torch.relu(self.fc2(x))
        return self.fc3(x)


class DeFiTradingAgent:
    """
    Autonomous DeFi Trading Agent using Q-learning
    """
    def __init__(self, learning_rate: float = 0.001, gamma: float = 0.99,
                 epsilon: float = 0.1, state_size: int = 1, action_size: int = 3):
        self.state_size = state_size
        self.action_size = action_size
        self.gamma = gamma
        self.epsilon = epsilon

        self.q_net = QNetwork(state_size, action_size)
        self.optimizer = optim.Adam(self.q_net.parameters(), lr=learning_rate)
        self.criterion = nn.MSELoss()

        self.actions = ["hold", "buy", "sell"]
        self.training_history = []

    def get_action(self, state: float, explore: bool = False) -> Tuple[int, str]:
        """
        Get trading action based on current state

        Args:
            state: Normalized price delta (-1 to 1)
            explore: Whether to use epsilon-greedy exploration

        Returns:
            Tuple of (action_index, action_name)
        """
        state_tensor = torch.tensor([state], dtype=torch.float32)

        # Epsilon-greedy action selection
        if explore and np.random.rand() < self.epsilon:
            action = np.random.randint(0, self.action_size)
        else:
            with torch.no_grad():
                q_values = self.q_net(state_tensor)
                action = torch.argmax(q_values).item()

        return action, self.actions[action]

    def train_step(self, state: float, action: int, reward: float, next_state: float):
        """
        Perform one training step using Q-learning update rule

        Args:
            state: Current state (price delta)
            action: Action taken
            reward: Reward received
            next_state: Next state after action
        """
        state_tensor = torch.tensor([state], dtype=torch.float32)
        next_state_tensor = torch.tensor([next_state], dtype=torch.float32)

        # Compute target Q-value
        with torch.no_grad():
            next_q = self.q_net(next_state_tensor).max()
        target = reward + self.gamma * next_q

        # Compute current Q-value and loss
        q_values = self.q_net(state_tensor)
        loss = self.criterion(q_values[0, action], target)

        # Update network
        self.optimizer.zero_grad()
        loss.backward()
        self.optimizer.step()

        self.training_history.append({
            'state': state,
            'action': action,
            'reward': reward,
            'loss': loss.item()
        })

    def train_on_simulated_data(self, episodes: int = 1000, verbose: bool = True):
        """
        Train the agent on simulated price data

        Args:
            episodes: Number of training episodes
            verbose: Whether to print progress
        """
        for episode in range(episodes):
            # Simulate state (price delta between -1 and 1)
            state = np.random.uniform(-1, 1)

            # Get action with exploration
            action, action_name = self.get_action(state, explore=True)

            # Simulate reward (positive for buying low, selling high)
            if action == 1 and state < -0.2:  # Buy when price dropped significantly
                reward = 1.0
            elif action == 2 and state > 0.2:  # Sell when price rose significantly
                reward = 1.0
            elif action == 0:  # Neutral for holding
                reward = 0.1
            else:  # Penalty for bad decisions
                reward = -1.0

            # Simulate next state
            next_state = state + np.random.uniform(-0.1, 0.1)
            next_state = np.clip(next_state, -1, 1)

            # Train
            self.train_step(state, action, reward, next_state)

            if verbose and (episode + 1) % 100 == 0:
                avg_loss = np.mean([h['loss'] for h in self.training_history[-100:]])
                print(f"Episode {episode + 1}/{episodes}, Avg Loss: {avg_loss:.4f}")

    def calculate_price_delta(self, current_price: float, reference_price: float) -> float:
        """
        Calculate normalized price delta for state representation

        Args:
            current_price: Current token price
            reference_price: Reference price (e.g., moving average)

        Returns:
            Normalized delta between -1 and 1
        """
        if reference_price == 0:
            return 0.0

        delta = (current_price - reference_price) / reference_price
        # Clip to [-1, 1] range
        return np.clip(delta, -1, 1)

    def save_model(self, filepath: str):
        """Save model weights to file"""
        torch.save({
            'model_state_dict': self.q_net.state_dict(),
            'optimizer_state_dict': self.optimizer.state_dict(),
            'training_history': self.training_history
        }, filepath)
        print(f"Model saved to {filepath}")

    def load_model(self, filepath: str):
        """Load model weights from file"""
        if os.path.exists(filepath):
            checkpoint = torch.load(filepath)
            self.q_net.load_state_dict(checkpoint['model_state_dict'])
            self.optimizer.load_state_dict(checkpoint['optimizer_state_dict'])
            self.training_history = checkpoint.get('training_history', [])
            print(f"Model loaded from {filepath}")
        else:
            print(f"No saved model found at {filepath}")


def demo_agent():
    """Demonstrate the AI agent with simulated data"""
    print("=== AI DeFi Trading Agent Demo ===\n")

    # Initialize agent
    agent = DeFiTradingAgent(learning_rate=0.001, gamma=0.99, epsilon=0.1)

    # Train on simulated data
    print("Training agent on simulated price data...")
    agent.train_on_simulated_data(episodes=1000, verbose=True)

    # Test decisions
    print("\n=== Testing Trained Agent ===")
    test_scenarios = [
        (-0.3, "Price dropped 30%"),
        (0.3, "Price rose 30%"),
        (-0.1, "Price dropped 10%"),
        (0.05, "Price rose 5%"),
        (0.0, "Price unchanged")
    ]

    for price_delta, description in test_scenarios:
        action_idx, action_name = agent.get_action(price_delta, explore=False)
        print(f"{description}: Agent decision = {action_name.upper()}")

    # Save model
    model_path = "../models/trading_agent.pth"
    os.makedirs("../models", exist_ok=True)
    agent.save_model(model_path)

    return agent


if __name__ == "__main__":
    demo_agent()
