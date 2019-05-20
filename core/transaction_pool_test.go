// Copyright 2015 The go-ethereum Authors
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

package core

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
)

func transaction(nonce uint64, gaslimit *big.Int, key *ecdsa.PrivateKey) *types.Transaction {
	tx, _ := types.NewTransaction(nonce, common.Address{}, big.NewInt(100), gaslimit, big.NewInt(1), nil).SignECDSA(key)
	return tx
}

func setupTxPool() (*TxPool, *ecdsa.PrivateKey) {
	db, _ := ethdb.NewMemDatabase()
	statedb := state.New(common.Hash{}, db)

	var m event.TypeMux
	key, _ := crypto.GenerateKey()
	return NewTxPool(&m, func() *state.StateDB { return statedb }, func() *big.Int { return big.NewInt(1000000) }), key
}

func TestInvalidTransactions(t *testing.T) {
	pool, key := setupTxPool()

	tx := transaction(0, big.NewInt(100), key)
	if err := pool.Add(tx); err != ErrNonExistentAccount {
		t.Error("expected", ErrNonExistentAccount)
	}

	from, _ := tx.From()
	pool.currentState().AddBalance(from, big.NewInt(1))
	if err := pool.Add(tx); err != ErrInsufficientFunds {
		t.Error("expected", ErrInsufficientFunds)
	}

	balance := new(big.Int).Add(tx.Value(), new(big.Int).Mul(tx.Gas(), tx.GasPrice()))
	pool.currentState().AddBalance(from, balance)
	if err := pool.Add(tx); err != ErrIntrinsicGas {
		t.Error("expected", ErrIntrinsicGas, "got", err)
	}

	pool.currentState().SetNonce(from, 1)
	pool.currentState().AddBalance(from, big.NewInt(0xffffffffffffff))
	tx = transaction(0, big.NewInt(100000), key)
	if err := pool.Add(tx); err != ErrNonce {
		t.Error("expected", ErrNonce)
	}
}

func TestTransactionQueue(t *testing.T) {
	pool, key := setupTxPool()
	tx := transaction(0, big.NewInt(100), key)
	from, _ := tx.From()
	pool.currentState().AddBalance(from, big.NewInt(1))
	pool.queueTx(tx.Hash(), tx)

	pool.checkQueue()
	if len(pool.pending) != 1 {
		t.Error("expected valid txs to be 1 is", len(pool.pending))
	}

	tx = transaction(1, big.NewInt(100), key)
	from, _ = tx.From()
	pool.currentState().SetNonce(from, 2)
	pool.queueTx(tx.Hash(), tx)
	pool.checkQueue()
	if _, ok := pool.pending[tx.Hash()]; ok {
		t.Error("expected transaction to be in tx pool")
	}

	if len(pool.queue[from]) > 0 {
		t.Error("expected transaction queue to be empty. is", len(pool.queue[from]))
	}

	pool, key = setupTxPool()
	tx1 := transaction(0, big.NewInt(100), key)
	tx2 := transaction(10, big.NewInt(100), key)
	tx3 := transaction(11, big.NewInt(100), key)
	pool.queueTx(tx1.Hash(), tx1)
	pool.queueTx(tx2.Hash(), tx2)
	pool.queueTx(tx3.Hash(), tx3)
	from, _ = tx1.From()

	pool.checkQueue()

	if len(pool.pending) != 1 {
		t.Error("expected tx pool to be 1 =")
	}
	if len(pool.queue[from]) != 2 {
		t.Error("expected len(queue) == 2, got", len(pool.queue[from]))
	}
}

func TestRemoveTx(t *testing.T) {
	pool, key := setupTxPool()
	tx := transaction(0, big.NewInt(100), key)
	from, _ := tx.From()
	pool.currentState().AddBalance(from, big.NewInt(1))
	pool.queueTx(tx.Hash(), tx)
	pool.addTx(tx.Hash(), from, tx)
	if len(pool.queue) != 1 {
		t.Error("expected queue to be 1, got", len(pool.queue))
	}

	if len(pool.pending) != 1 {
		t.Error("expected txs to be 1, got", len(pool.pending))
	}

	pool.removeTx(tx.Hash())

	if len(pool.queue) > 0 {
		t.Error("expected queue to be 0, got", len(pool.queue))
	}

	if len(pool.pending) > 0 {
		t.Error("expected txs to be 0, got", len(pool.pending))
	}
}

func TestNegativeValue(t *testing.T) {
	pool, key := setupTxPool()

	tx, _ := types.NewTransaction(0, common.Address{}, big.NewInt(-1), big.NewInt(100), big.NewInt(1), nil).SignECDSA(key)
	from, _ := tx.From()
	pool.currentState().AddBalance(from, big.NewInt(1))
	if err := pool.Add(tx); err != ErrNegativeValue {
		t.Error("expected", ErrNegativeValue, "got", err)
	}
}

func TestTransactionChainFork(t *testing.T) {
	pool, key := setupTxPool()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	resetState := func() {
		db, _ := ethdb.NewMemDatabase()
		statedb := state.New(common.Hash{}, db)
		pool.currentState = func() *state.StateDB { return statedb }
		pool.currentState().AddBalance(addr, big.NewInt(100000000000000))
		pool.resetState()
	}
	resetState()

	tx := transaction(0, big.NewInt(100000), key)
	if err := pool.add(tx); err != nil {
		t.Error("didn't expect error", err)
	}
	pool.RemoveTransactions([]*types.Transaction{tx})

	// reset the pool's internal state
	resetState()
	if err := pool.add(tx); err != nil {
		t.Error("didn't expect error", err)
	}
}

func TestTransactionDoubleNonce(t *testing.T) {
	pool, key := setupTxPool()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	resetState := func() {
		db, _ := ethdb.NewMemDatabase()
		statedb := state.New(common.Hash{}, db)
		pool.currentState = func() *state.StateDB { return statedb }
		pool.currentState().AddBalance(addr, big.NewInt(100000000000000))
		pool.resetState()
	}
	resetState()

	tx := transaction(0, big.NewInt(100000), key)
	tx2 := transaction(0, big.NewInt(1000000), key)
	if err := pool.add(tx); err != nil {
		t.Error("didn't expect error", err)
	}
	if err := pool.add(tx2); err != nil {
		t.Error("didn't expect error", err)
	}

	pool.checkQueue()
	if len(pool.pending) != 2 {
		t.Error("expected 2 pending txs. Got", len(pool.pending))
	}
}

func TestMissingNonce(t *testing.T) {
	pool, key := setupTxPool()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState().AddBalance(addr, big.NewInt(100000000000000))
	tx := transaction(1, big.NewInt(100000), key)
	if err := pool.add(tx); err != nil {
		t.Error("didn't expect error", err)
	}
	if len(pool.pending) != 0 {
		t.Error("expected 0 pending transactions, got", len(pool.pending))
	}
	if len(pool.queue[addr]) != 1 {
		t.Error("expected 1 queued transaction, got", len(pool.queue[addr]))
	}
}
