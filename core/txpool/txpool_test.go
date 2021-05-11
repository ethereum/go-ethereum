// Copyright 2021 The go-ethereum Authors
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

package txpool

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
)

func TestInvalidTransactions(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	tx := transaction(0, 100, key)
	from, _ := deriveSender(tx)

	pool.currentState.AddBalance(from, big.NewInt(1))
	if err := pool.AddRemotes([]*types.Transaction{tx}); !errors.Is(err[0], core.ErrInsufficientFunds) {
		t.Error("expected", core.ErrInsufficientFunds)
	}

	balance := new(big.Int).Add(tx.Value(), new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), tx.GasPrice()))
	pool.currentState.AddBalance(from, balance)
	if err := pool.AddRemotes([]*types.Transaction{tx}); !errors.Is(err[0], core.ErrIntrinsicGas) {
		t.Error("expected", core.ErrIntrinsicGas)
	}

	pool.currentState.SetNonce(from, 1)
	pool.currentState.AddBalance(from, big.NewInt(0xffffffffffffff))
	tx = transaction(0, 100000, key)
	if err := pool.AddRemotes([]*types.Transaction{tx}); !errors.Is(err[0], core.ErrNonceTooLow) {
		t.Error("expected", core.ErrNonceTooLow)
	}
	// Test negative value
	tx2, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(-1), 100, big.NewInt(1), nil), types.HomesteadSigner{}, key)
	from2, _ := deriveSender(tx2)
	pool.currentState.AddBalance(from2, big.NewInt(1))
	if err := pool.AddRemotes([]*types.Transaction{tx2}); err[0] != core.ErrNegativeValue {
		t.Error("expected", core.ErrNegativeValue, "got", err[0])
	}

	tx = transaction(1, 100000, key)
	pool.config.minGasPrice = big.NewInt(1000)
	if err := pool.AddRemotes([]*types.Transaction{tx}); err[0] != core.ErrUnderpriced {
		t.Error("expected", core.ErrUnderpriced)
	}
	if err := pool.AddLocal(tx); err != nil {
		t.Error("expected", nil, "got", err)
	}

}

func TestTransactionMissingNonce(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState.AddBalance(addr, big.NewInt(100000000000000))
	tx := transaction(1, 100000, key)
	if err := pool.AddRemotes([]*types.Transaction{tx}); err[0] != nil {
		t.Error("didn't expect error", err)
	}
	if pool.remoteTxs.Len() != 0 {
		t.Error("expected 0 pending transactions, got", pool.remoteTxs.Len())
	}
	if pool.gappedTxs[addr].Len() != 1 {
		t.Error("expected 1 queued transaction, got", pool.gappedTxs[addr].Len())
	}
	if pool.all.Count() != 1 {
		t.Error("expected 1 total transactions, got", pool.all.Count())
	}
}

// Tests that the pool rejects duplicate transactions.
func TestTransactionDeduplication(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{statedb, 1000000, new(event.Feed)}

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create a test account to add transactions with
	key, _ := crypto.GenerateKey()
	pool.currentState.AddBalance(crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000000))

	// Create a batch of transactions and add a few of them
	txs := make([]*types.Transaction, 16)
	for i := 0; i < len(txs); i++ {
		txs[i] = pricedTransaction(uint64(i), 100000, big.NewInt(1), key)
	}
	var firsts []*types.Transaction
	for i := 0; i < len(txs); i += 2 {
		firsts = append(firsts, txs[i])
	}
	errs := pool.AddRemotesSync(firsts)
	if len(errs) != len(firsts) {
		t.Fatalf("first add mismatching result count: have %d, want %d", len(errs), len(firsts))
	}
	for i, err := range errs {
		if err != nil {
			t.Errorf("add %d failed: %v", i, err)
		}
	}
	pending, queued := pool.Stats()
	if pending != 1 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 1)
	}
	if queued != len(txs)/2-1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, len(txs)/2-1)
	}
	// Try to add all of them now and ensure previous ones error out as knowns
	errs = pool.AddRemotesSync(txs)
	if len(errs) != len(txs) {
		t.Fatalf("all add mismatching result count: have %d, want %d", len(errs), len(txs))
	}
	for i, err := range errs {
		if i%2 == 0 && err == nil {
			t.Errorf("add %d succeeded, should have failed as known", i)
		}
		if i%2 == 1 && err != nil {
			t.Errorf("add %d failed: %v", i, err)
		}
	}
	pending, queued = pool.Stats()
	if pending != len(txs) {
		_, remote := pool.PendingBlock()
		for _, t := range remote {
			fmt.Printf("%v\n", t.Nonce())
		}
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, len(txs))
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// validateTxPoolInternals checks various consistency invariants within the pool.
func validateTxPoolInternals(pool *TxPool) error {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	// Ensure the total transaction set is consistent with pending + queued
	pending, queued := pool.Stats()
	if total := pool.all.Count(); total != pending+queued {
		return fmt.Errorf("total transaction count %d != %d pending + %d queued", total, pending, queued)
	}

	/*
		// Ensure the next nonce to assign is the correct one
		for addr, txs := range pool.Pending() {
			// Find the last transaction
			var last uint64
			for nonce := range txs.txs.items {
				if last < nonce {
					last = nonce
				}
			}
			if nonce := pool.pendingNonces.get(addr); nonce != last+1 {
				return fmt.Errorf("pending nonce mismatch: have %v, want %v", nonce, last+1)
			}
		}*/
	return nil
}

func deriveSender(tx *types.Transaction) (common.Address, error) {
	return types.Sender(types.HomesteadSigner{}, tx)
}
