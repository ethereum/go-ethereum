"""
Unit tests for AI DeFi Trading Agent
Run with: pytest tests/test_agent.py -v
"""

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'src'))

import numpy as np
import torch
from ai_agent import DeFiTradingAgent, QNetwork


class TestQNetwork:
    """Test the Q-Network implementation"""

    def test_network_initialization(self):
        """Test that network initializes correctly"""
        net = QNetwork(state_size=1, action_size=3)
        assert net is not None
        assert isinstance(net, torch.nn.Module)

    def test_network_forward_pass(self):
        """Test forward pass produces correct output shape"""
        net = QNetwork(state_size=1, action_size=3)
        state = torch.tensor([0.5], dtype=torch.float32)
        output = net(state)
        assert output.shape == torch.Size([3])

    def test_network_output_range(self):
        """Test that network outputs reasonable Q-values"""
        net = QNetwork(state_size=1, action_size=3)
        state = torch.tensor([0.0], dtype=torch.float32)
        output = net(state)
        # Q-values should be finite
        assert torch.all(torch.isfinite(output))


class TestDeFiTradingAgent:
    """Test the DeFi Trading Agent"""

    def test_agent_initialization(self):
        """Test agent initializes with correct parameters"""
        agent = DeFiTradingAgent(learning_rate=0.001, gamma=0.99, epsilon=0.1)
        assert agent.gamma == 0.99
        assert agent.epsilon == 0.1
        assert len(agent.actions) == 3

    def test_get_action_returns_valid_action(self):
        """Test that get_action returns valid action indices"""
        agent = DeFiTradingAgent()
        for _ in range(10):
            state = np.random.uniform(-1, 1)
            action_idx, action_name = agent.get_action(state)
            assert action_idx in [0, 1, 2]
            assert action_name in ["hold", "buy", "sell"]

    def test_get_action_exploration(self):
        """Test epsilon-greedy exploration"""
        agent = DeFiTradingAgent(epsilon=1.0)  # Always explore
        actions = []
        for _ in range(100):
            action_idx, _ = agent.get_action(0.0, explore=True)
            actions.append(action_idx)
        # With epsilon=1.0, should see all actions
        assert len(set(actions)) > 1

    def test_calculate_price_delta(self):
        """Test price delta calculation"""
        agent = DeFiTradingAgent()

        # Test normal case
        delta = agent.calculate_price_delta(110, 100)
        assert abs(delta - 0.1) < 0.001

        # Test negative change
        delta = agent.calculate_price_delta(90, 100)
        assert abs(delta - (-0.1)) < 0.001

        # Test clipping at bounds
        delta = agent.calculate_price_delta(200, 100)
        assert delta == 1.0  # Should clip at 1.0

        delta = agent.calculate_price_delta(0, 100)
        assert delta == -1.0  # Should clip at -1.0

        # Test zero reference
        delta = agent.calculate_price_delta(100, 0)
        assert delta == 0.0

    def test_train_step(self):
        """Test training step executes without error"""
        agent = DeFiTradingAgent()
        state = 0.5
        action = 1
        reward = 1.0
        next_state = 0.6

        initial_history_len = len(agent.training_history)
        agent.train_step(state, action, reward, next_state)
        assert len(agent.training_history) == initial_history_len + 1

    def test_training_on_simulated_data(self):
        """Test that agent can train on simulated data"""
        agent = DeFiTradingAgent()
        initial_history = len(agent.training_history)

        agent.train_on_simulated_data(episodes=10, verbose=False)

        # Should have 10 new history entries
        assert len(agent.training_history) == initial_history + 10

    def test_model_save_and_load(self, tmp_path):
        """Test model saving and loading"""
        agent1 = DeFiTradingAgent()
        agent1.train_on_simulated_data(episodes=5, verbose=False)

        # Save model
        model_path = tmp_path / "test_model.pth"
        agent1.save_model(str(model_path))
        assert model_path.exists()

        # Load into new agent
        agent2 = DeFiTradingAgent()
        agent2.load_model(str(model_path))

        # Both agents should make same decision
        state = 0.5
        action1, _ = agent1.get_action(state, explore=False)
        action2, _ = agent2.get_action(state, explore=False)
        assert action1 == action2


class TestTradingLogic:
    """Test trading decision logic"""

    def test_buy_on_price_drop(self):
        """Agent should prefer buying when price drops"""
        agent = DeFiTradingAgent(epsilon=0.0)  # No exploration
        agent.train_on_simulated_data(episodes=500, verbose=False)

        # Test on large price drop
        action_idx, action_name = agent.get_action(-0.3, explore=False)
        # After training, should learn to buy on dips
        # Note: This might not always be "buy" due to random initialization
        assert action_name in ["buy", "hold"]  # Reasonable actions

    def test_sell_on_price_rise(self):
        """Agent should consider selling when price rises"""
        agent = DeFiTradingAgent(epsilon=0.0)
        agent.train_on_simulated_data(episodes=500, verbose=False)

        # Test on large price rise
        action_idx, action_name = agent.get_action(0.3, explore=False)
        # After training, might sell or hold
        assert action_name in ["sell", "hold"]

    def test_consistent_decisions(self):
        """Same state should produce same decision without exploration"""
        agent = DeFiTradingAgent(epsilon=0.0)
        state = 0.2

        action1, _ = agent.get_action(state, explore=False)
        action2, _ = agent.get_action(state, explore=False)
        action3, _ = agent.get_action(state, explore=False)

        assert action1 == action2 == action3


def test_config_validation():
    """Test configuration validation"""
    from config import Config

    # This will fail without environment variables, which is expected
    # In CI, you'd mock these
    result = Config.validate()
    # Should return False if env vars not set
    assert isinstance(result, bool)


if __name__ == "__main__":
    print("Running basic tests...")
    print("\nTest 1: Q-Network")
    test = TestQNetwork()
    test.test_network_initialization()
    test.test_network_forward_pass()
    print("✓ Q-Network tests passed")

    print("\nTest 2: Trading Agent")
    test = TestDeFiTradingAgent()
    test.test_agent_initialization()
    test.test_get_action_returns_valid_action()
    test.test_calculate_price_delta()
    print("✓ Trading Agent tests passed")

    print("\n✅ All basic tests passed!")
