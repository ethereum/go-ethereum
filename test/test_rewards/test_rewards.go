package main

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// Connect to node1
	client1, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		fmt.Printf("Failed to connect to node1: %v\n", err)
		return
	}
	defer client1.Close()

	// Validator addresses
	miner1 := common.HexToAddress("0xca6b49ee60cdd276ab503fbd6fb80a3cfbc06ffc") // Node1 (whitelisted)
	miner2 := common.HexToAddress("0xab52b2c71f61cd9447a932c0cb55d1752571dab8") // Node2 (not whitelisted)

	fmt.Println("=== Testing Whitelist Rewards ===\n")

	// Get initial balances
	ctx := context.Background()
	balance1Before, err := client1.BalanceAt(ctx, miner1, nil)
	if err != nil {
		fmt.Printf("Error getting balance1: %v\n", err)
		return
	}

	balance2Before, err := client1.BalanceAt(ctx, miner2, nil)
	if err != nil {
		fmt.Printf("Error getting balance2: %v\n", err)
		return
	}

	blockNumBefore, err := client1.BlockNumber(ctx)
	if err != nil {
		fmt.Printf("Error getting block number: %v\n", err)
		return
	}

	fmt.Printf("Block number: %d\n", blockNumBefore)
	fmt.Printf("Miner1 (Node1 - should be whitelisted) balance: %s ETH\n", weiToEth(balance1Before))
	fmt.Printf("Miner2 (Node2 - not whitelisted) balance: %s ETH\n", weiToEth(balance2Before))
	fmt.Println("\nWaiting for 10 blocks to be mined...")

	// Wait for 10 blocks
	var blockNumAfter uint64
	for i := 0; i < 30; i++ { // Check every 2 seconds, max 60 seconds
		time.Sleep(2 * time.Second)
		blockNumAfter, err = client1.BlockNumber(ctx)
		if err == nil && blockNumAfter >= blockNumBefore+10 {
			break
		}
	}

	if blockNumAfter < blockNumBefore+10 {
		fmt.Printf("Warning: Only %d blocks mined (expected 10). Continuing anyway...\n", blockNumAfter-blockNumBefore)
	}

	// Get balances after
	balance1After, err := client1.BalanceAt(ctx, miner1, nil)
	if err != nil {
		fmt.Printf("Error getting balance1 after: %v\n", err)
		return
	}

	balance2After, err := client1.BalanceAt(ctx, miner2, nil)
	if err != nil {
		fmt.Printf("Error getting balance2 after: %v\n", err)
		return
	}

	blocksMined := blockNumAfter - blockNumBefore
	balance1Diff := new(big.Int).Sub(balance1After, balance1Before)
	balance2Diff := new(big.Int).Sub(balance2After, balance2Before)

	fmt.Printf("\n=== Results after %d blocks ===\n", blocksMined)
	fmt.Printf("Miner1 balance: %s ETH (change: %s ETH)\n", weiToEth(balance1After), weiToEth(balance1Diff))
	fmt.Printf("Miner2 balance: %s ETH (change: %s ETH)\n", weiToEth(balance2After), weiToEth(balance2Diff))

	// Calculate per-block reward
	if blocksMined > 0 {
		reward1PerBlock := new(big.Int).Div(balance1Diff, big.NewInt(int64(blocksMined)))
		reward2PerBlock := new(big.Int).Div(balance2Diff, big.NewInt(int64(blocksMined)))

		fmt.Printf("\n=== Per-Block Rewards ===\n")
		fmt.Printf("Miner1 (whitelisted) per-block reward: %s ETH\n", weiToEth(reward1PerBlock))
		fmt.Printf("Miner2 (not whitelisted) per-block reward: %s ETH\n", weiToEth(reward2PerBlock))

		// Compare rewards
		if reward2PerBlock.Sign() > 0 {
			ratio := new(big.Float).Quo(
				new(big.Float).SetInt(reward1PerBlock),
				new(big.Float).SetInt(reward2PerBlock),
			)
			fmt.Printf("\n=== Analysis ===\n")
			fmt.Printf("Miner1 reward / Miner2 reward ratio: %.2fx\n", ratio)
			if ratio.Cmp(big.NewFloat(1.5)) > 0 {
				fmt.Println("✅ SUCCESS: Miner1 is receiving EXTRA rewards (whitelisted)!")
			} else {
				fmt.Println("⚠️  WARNING: Miner1 reward ratio is not significantly higher.")
				fmt.Println("   Make sure you've run: go run ./docker/whitelist_tx.go")
			}
		}
	}

	// Check latest blocks to see who mined them
	fmt.Println("\n=== Recent Block Miners ===")
	for i := uint64(0); i < 5 && blockNumAfter >= i; i++ {
		blockNum := blockNumAfter - i
		block, err := client1.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
		if err == nil && block != nil {
			coinbase := block.Coinbase()
			isMiner1 := coinbase == miner1
			isMiner2 := coinbase == miner2
			minerName := "Unknown"
			if isMiner1 {
				minerName = "Miner1 (Node1 - whitelisted)"
			} else if isMiner2 {
				minerName = "Miner2 (Node2 - not whitelisted)"
			}
			fmt.Printf("Block %d: mined by %s (%s)\n", blockNum, coinbase.Hex(), minerName)
		}
	}
}

func weiToEth(wei *big.Int) string {
	eth := new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(1e18))
	return eth.Text('f', 6)
}
