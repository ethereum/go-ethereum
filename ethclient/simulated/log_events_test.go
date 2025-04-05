package simulated

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// TestLogEventsImmediateAvailability verifies that the fix for issue #31518
// ensures log events are immediately available after transaction commitment.
func TestLogEventsImmediateAvailability(t *testing.T) {
	// Generate a random private key instead of using a hardcoded one
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate random key: %v", err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)

	// Create a simulated backend with initial allocation
	balance := new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))
	sim := NewBackend(types.GenesisAlloc{addr: {Balance: balance}})
	defer sim.Close()

	// Create a client to interact with the simulated backend
	client := sim.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Deploy a simple contract that emits logs
	contractCode := []byte{
		// PUSH1 1, PUSH1 0, PUSH1 0, LOG0 - Simple bytecode that emits a log
		0x60, 0x01, 0x60, 0x00, 0x60, 0x00, 0xa0,
		// PUSH1 0, PUSH1 0, RETURN - Return empty
		0x60, 0x00, 0x60, 0x00, 0xf3,
	}

	// Get the nonce for the sender
	nonce, err := client.PendingNonceAt(ctx, addr)
	if err != nil {
		t.Fatalf("Failed to get nonce: %v", err)
	}

	// Get the current gas price
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		t.Fatalf("Failed to get gas price: %v", err)
	}

	// Create the contract creation transaction
	tx := types.NewContractCreation(
		nonce,
		big.NewInt(0),
		500000, // Gas limit
		gasPrice,
		contractCode,
	)

	// Sign the transaction
	chainID, err := client.ChainID(ctx)
	if err != nil {
		t.Fatalf("Failed to get chain ID: %v", err)
	}
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), key)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Set up a filter to catch logs
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(0),
		Topics:    [][]common.Hash{},
	}

	// Send the transaction
	t.Log("Sending contract creation transaction")
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		t.Fatalf("Failed to send transaction: %v", err)
	}

	// Commit the transaction - this should emit logs
	t.Log("Committing block")
	sim.Commit()

	// Check for the transaction receipt
	receipt, err := client.TransactionReceipt(ctx, signedTx.Hash())
	if err != nil {
		t.Fatalf("Failed to get transaction receipt: %v", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatal("Transaction failed")
	}

	// Update the filter query to include the contract address
	query.Addresses = []common.Address{receipt.ContractAddress}

	// Check for logs immediately without any sleep or wait
	t.Log("Checking for logs immediately after commit")
	logs, err := client.FilterLogs(ctx, query)
	if err != nil {
		t.Fatalf("Error filtering logs: %v", err)
	}

	// Verify logs are found immediately
	if len(logs) == 0 {
		t.Fatal("No logs found immediately after commit")
	}

	t.Logf("Success: Found %d logs immediately after commit", len(logs))

	// Additional verification - check log details
	for i, log := range logs {
		t.Logf("Log #%d - BlockNumber: %d, TxHash: %s, Index: %d",
			i, log.BlockNumber, log.TxHash.Hex(), log.Index)
		if log.BlockNumber != receipt.BlockNumber.Uint64() {
			t.Errorf("Log block number mismatch: expected %d, got %d",
				receipt.BlockNumber.Uint64(), log.BlockNumber)
		}
		if log.TxHash != signedTx.Hash() {
			t.Errorf("Log transaction hash mismatch: expected %s, got %s",
				signedTx.Hash().Hex(), log.TxHash.Hex())
		}
	}
}
