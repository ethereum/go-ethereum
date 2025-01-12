package simulated

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

// TestTransactionRollbackBehavior verifies the behavior of transactions
// in the simulated backend after rollback operations.
//
// The test demonstrates that after a rollback:
//  1. The first test shows normal transaction processing without rollback
//  2. The second test shows that transactions immediately after rollback fail
//  3. The third test shows a workaround: committing an empty block after rollback
//     makes subsequent transactions succeed
func TestTransactionRollbackBehavior(t *testing.T) {
	sim := simTestBackend(testAddr)
	defer sim.Close()
	client := sim.Client()

	t.Run("Case 1: Basic Transaction (Control Case)", func(t *testing.T) {
		// Demonstrates normal transaction processing works as expected
		tx := testSendSignedTx(t, sim)
		sim.Commit()
		assertSuccessfulReceipt(t, client, tx)
	})

	t.Run("Case 2: Transaction After Rollback (Shows Issue)", func(t *testing.T) {
		// First transaction gets rolled back
		_ = testSendSignedTx(t, sim)
		sim.Rollback()

		// Attempting to process a new transaction immediately after rollback
		// Currently, this case fails to get a valid receipt
		tx := testSendSignedTx(t, sim)
		sim.Commit()
		assertSuccessfulReceipt(t, client, tx)
	})

	t.Run("Case 3: Transaction After Rollback with Empty Block (Workaround)", func(t *testing.T) {
		// First transaction gets rolled back
		_ = testSendSignedTx(t, sim)
		sim.Rollback()

		// Workaround: Commit an empty block after rollback
		sim.Commit()

		// Now the new transaction succeeds
		tx := testSendSignedTx(t, sim)
		sim.Commit()
		assertSuccessfulReceipt(t, client, tx)
	})
}

// testSendSignedTx sends a signed transaction to the simulated backend.
// It does not commit the block.
func testSendSignedTx(t *testing.T, sim *Backend) *types.Transaction {
	t.Helper()
	client := sim.Client()
	ctx := context.Background()

	signedTx, err := newTx(sim, testKey)
	if err != nil {
		t.Fatalf("failed to create transaction: %v", err)
	}

	if err = client.SendTransaction(ctx, signedTx); err != nil {
		t.Fatalf("failed to send transaction: %v", err)
	}

	return signedTx
}

// assertSuccessfulReceipt verifies that a transaction was successfully processed
// by checking its receipt status.
func assertSuccessfulReceipt(t *testing.T, client Client, tx *types.Transaction) {
	t.Helper()
	ctx := context.Background()

	var (
		receipt *types.Receipt
		err     error
	)

	// Poll for receipt with timeout
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		receipt, err = client.TransactionReceipt(ctx, tx.Hash())
		if err == nil && receipt != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("failed to get transaction receipt: %v", err)
	}
	if receipt == nil {
		t.Fatal("transaction receipt is nil")
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("transaction failed with status: %v", receipt.Status)
	}
}
