#!/usr/bin/env python3
"""
Quick Demo of AI DeFi Trading Agent Components
Run this to test the AI agent without needing deployed contracts
"""

import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from ai_agent import DeFiTradingAgent


def demo_ai_agent():
    """Demonstrate the AI trading agent capabilities"""
    print("=" * 70)
    print("AI DeFi Trading Agent - Quick Demo")
    print("=" * 70)
    print("\nThis demo shows the AI agent's decision-making without blockchain")
    print("For full functionality, see DEPLOYMENT.md\n")

    # 1. Initialize agent
    print("📊 Step 1: Initializing AI Agent")
    print("-" * 70)
    agent = DeFiTradingAgent(
        learning_rate=0.001,
        gamma=0.99,
        epsilon=0.1
    )
    print(f"✓ Agent created with {agent.action_size} possible actions: {agent.actions}")
    print()

    # 2. Train on simulated data
    print("🎓 Step 2: Training on Simulated Market Data")
    print("-" * 70)
    print("Training Q-learning model on 1000 simulated episodes...")
    agent.train_on_simulated_data(episodes=1000, verbose=True)
    print("✓ Training complete!")
    print()

    # 3. Test decisions
    print("🤖 Step 3: Testing AI Decisions")
    print("-" * 70)
    print("Testing how the agent responds to different market conditions:\n")

    test_scenarios = [
        (-0.5, "🔴 MAJOR PRICE DROP: -50%"),
        (-0.3, "🟠 LARGE PRICE DROP: -30%"),
        (-0.1, "🟡 SMALL PRICE DROP: -10%"),
        (0.0, "⚪ PRICE UNCHANGED"),
        (0.1, "🟡 SMALL PRICE RISE: +10%"),
        (0.3, "🟠 LARGE PRICE RISE: +30%"),
        (0.5, "🟢 MAJOR PRICE RISE: +50%"),
    ]

    for price_delta, description in test_scenarios:
        action_idx, action_name = agent.get_action(price_delta, explore=False)

        # Format action with emoji
        if action_name == "buy":
            action_display = "💰 BUY"
        elif action_name == "sell":
            action_display = "💸 SELL"
        else:
            action_display = "🤝 HOLD"

        print(f"  {description:35} → Decision: {action_display}")

    print()

    # 4. Show decision consistency
    print("🔄 Step 4: Testing Decision Consistency")
    print("-" * 70)
    state = -0.2  # 20% price drop
    actions = []
    for i in range(5):
        action_idx, action_name = agent.get_action(state, explore=False)
        actions.append(action_name)

    if len(set(actions)) == 1:
        print(f"✓ Agent makes consistent decisions: {actions[0].upper()} (tested 5 times)")
    else:
        print(f"⚠ Agent decisions varied: {actions}")

    print()

    # 5. Show exploration vs exploitation
    print("🎲 Step 5: Exploration vs Exploitation")
    print("-" * 70)
    state = -0.2

    exploited_action, _ = agent.get_action(state, explore=False)
    print(f"Without exploration (pure AI): {exploited_action.upper()}")

    actions_with_exploration = []
    for _ in range(10):
        action_idx, action_name = agent.get_action(state, explore=True)
        actions_with_exploration.append(action_name)

    unique_actions = set(actions_with_exploration)
    print(f"With exploration (ε={agent.epsilon}): {unique_actions}")
    print(f"Exploration adds variety for learning (tested 10 times)")
    print()

    # 6. Price delta calculation
    print("📈 Step 6: Price Delta Calculation")
    print("-" * 70)
    price_examples = [
        (100, 100, "No change"),
        (110, 100, "10% increase"),
        (90, 100, "10% decrease"),
        (200, 100, "100% increase (clipped to +1.0)"),
        (0, 100, "100% decrease (clipped to -1.0)"),
    ]

    for current, reference, description in price_examples:
        delta = agent.calculate_price_delta(current, reference)
        print(f"  Price: ${current:6.2f} vs ${reference:6.2f} ({description:30}) → Delta: {delta:+.3f}")

    print()

    # 7. Training history
    print("📊 Step 7: Training Statistics")
    print("-" * 70)
    if agent.training_history:
        recent_losses = [h['loss'] for h in agent.training_history[-100:]]
        avg_loss = sum(recent_losses) / len(recent_losses)
        print(f"Total training episodes: {len(agent.training_history)}")
        print(f"Average loss (last 100): {avg_loss:.6f}")

        # Show reward distribution
        rewards = [h['reward'] for h in agent.training_history[-100:]]
        positive_rewards = sum(1 for r in rewards if r > 0)
        print(f"Positive rewards: {positive_rewards}/100 ({positive_rewards}%)")
    print()

    # 8. Save model
    print("💾 Step 8: Saving Model")
    print("-" * 70)
    os.makedirs("models", exist_ok=True)
    model_path = "models/demo_agent.pth"
    agent.save_model(model_path)
    print(f"✓ Model saved to {model_path}")
    print(f"  File size: {os.path.getsize(model_path)} bytes")
    print()

    # 9. Summary
    print("=" * 70)
    print("📝 Demo Summary")
    print("=" * 70)
    print("\n✅ Successfully demonstrated:")
    print("  • AI agent initialization")
    print("  • Training on simulated data")
    print("  • Decision-making for various market conditions")
    print("  • Consistent behavior (no exploration)")
    print("  • Exploration for learning")
    print("  • Price delta normalization")
    print("  • Model persistence (save/load)")
    print("\n🚀 Next Steps:")
    print("  1. Deploy smart contracts (see DEPLOYMENT.md)")
    print("  2. Configure .env with your blockchain connection")
    print("  3. Run: python src/trading_agent.py monitor")
    print("  4. Test with paper trading mode")
    print("  5. Only then consider live trading\n")
    print("=" * 70)


def quick_decision_test():
    """Quick interactive decision test"""
    print("\n" + "=" * 70)
    print("🎮 Interactive Decision Test")
    print("=" * 70 + "\n")

    agent = DeFiTradingAgent(epsilon=0.0)  # No randomness
    print("Training agent...")
    agent.train_on_simulated_data(episodes=500, verbose=False)
    print("✓ Training complete\n")

    print("Enter price changes to see AI decisions (or 'q' to quit):")
    print("Example: -0.2 for -20%, 0.3 for +30%\n")

    while True:
        try:
            user_input = input("Price change (e.g., -0.2): ").strip()
            if user_input.lower() in ['q', 'quit', 'exit']:
                print("Goodbye!")
                break

            price_delta = float(user_input)
            if price_delta < -1 or price_delta > 1:
                print("⚠ Warning: Value outside [-1, 1], clipping...")
                price_delta = max(-1, min(1, price_delta))

            action_idx, action_name = agent.get_action(price_delta, explore=False)

            if action_name == "buy":
                emoji = "💰"
            elif action_name == "sell":
                emoji = "💸"
            else:
                emoji = "🤝"

            print(f"  → AI Decision: {emoji} {action_name.upper()}\n")

        except ValueError:
            print("❌ Invalid input. Please enter a number.\n")
        except KeyboardInterrupt:
            print("\n\nGoodbye!")
            break


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "interactive":
        quick_decision_test()
    else:
        demo_ai_agent()
        print("\n💡 Tip: Run 'python demo.py interactive' for interactive mode\n")
