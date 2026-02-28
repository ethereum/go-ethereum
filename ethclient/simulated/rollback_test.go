// Copyright 2025 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package simulated

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

// TestTransactionRollbackBehavior tests that calling Rollback on the simulated backend doesn't prevent subsequent
// addition of new transactions
func TestTransactionRollbackBehavior(t *testing.T) {
	sim := NewBackend(
		types.GenesisAlloc{
			testAddr:  {Balance: big.NewInt(10000000000000000)},
			testAddr2: {Balance: big.NewInt(10000000000000000)},
		},
	)
	defer sim.Close()
	client := sim.Client()

	btx0 := testSendSignedTx(t, testKey, sim, true, 0)
	tx0 := testSendSignedTx(t, testKey2, sim, false, 0)
	tx1 := testSendSignedTx(t, testKey2, sim, false, 1)

	sim.Rollback()

	if pendingStateHasTx(client, btx0) || pendingStateHasTx(client, tx0) || pendingStateHasTx(client, tx1) {
		t.Fatalf("all transactions were not rolled back")
	}

	btx2 := testSendSignedTx(t, testKey, sim, true, 0)
	tx2 := testSendSignedTx(t, testKey2, sim, false, 0)
	tx3 := testSendSignedTx(t, testKey2, sim, false, 1)

	sim.Commit()

	if !pendingStateHasTx(client, btx2) || !pendingStateHasTx(client, tx2) || !pendingStateHasTx(client, tx3) {
		t.Fatalf("all post-rollback transactions were not included")
	}
}

// testSendSignedTx sends a signed transaction to the simulated backend.
// It does not commit the block.
func testSendSignedTx(t *testing.T, key *ecdsa.PrivateKey, sim *Backend, isBlobTx bool, nonce uint64) *types.Transaction {
	t.Helper()
	client := sim.Client()
	ctx := context.Background()

	var (
		err      error
		signedTx *types.Transaction
	)
	if isBlobTx {
		signedTx, err = newBlobTx(sim, key, nonce)
	} else {
		signedTx, err = newTx(sim, key, nonce)
	}
	if err != nil {
		t.Fatalf("failed to create transaction: %v", err)
	}

	if err = client.SendTransaction(ctx, signedTx); err != nil {
		t.Fatalf("failed to send transaction: %v", err)
	}

	return signedTx
}

// pendingStateHasTx returns true if a given transaction was successfully included as of the latest pending state.
func pendingStateHasTx(client Client, tx *types.Transaction) bool {
	ctx := context.Background()

	var (
		receipt *types.Receipt
		err     error
	)

	// Poll for receipt with timeout
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		receipt, err = client.TransactionReceipt(ctx, tx.Hash())
		if err == nil && receipt != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if err != nil {
		return false
	}
	if receipt == nil {
		return false
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return false
	}
	return true
}
