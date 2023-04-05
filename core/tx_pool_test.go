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
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/holiman/uint256"
	"go.uber.org/goleak"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
	"pgregory.net/rapid"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/debug"
	"github.com/ethereum/go-ethereum/common/leak"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/JekaMas/crand"
)

var (
	// testTxPoolConfig is a transaction pool configuration without stateful disk
	// sideeffects used during testing.
	testTxPoolConfig TxPoolConfig

	// eip1559Config is a chain config with EIP-1559 enabled at block 0.
	eip1559Config *params.ChainConfig
)

const (
	txPoolGasLimit = 10_000_000
)

func init() {
	testTxPoolConfig = DefaultTxPoolConfig
	testTxPoolConfig.Journal = ""

	cpy := *params.TestChainConfig
	eip1559Config = &cpy
	eip1559Config.BerlinBlock = common.Big0
	eip1559Config.LondonBlock = common.Big0
}

type testBlockChain struct {
	gasLimit      uint64 // must be first field for 64 bit alignment (atomic access)
	statedb       *state.StateDB
	chainHeadFeed *event.Feed
}

func (bc *testBlockChain) CurrentBlock() *types.Block {
	return types.NewBlock(&types.Header{
		GasLimit: atomic.LoadUint64(&bc.gasLimit),
	}, nil, nil, nil, trie.NewStackTrie(nil))
}

func (bc *testBlockChain) GetBlock(hash common.Hash, number uint64) *types.Block {
	return bc.CurrentBlock()
}

func (bc *testBlockChain) StateAt(common.Hash) (*state.StateDB, error) {
	return bc.statedb, nil
}

func (bc *testBlockChain) SubscribeChainHeadEvent(ch chan<- ChainHeadEvent) event.Subscription {
	return bc.chainHeadFeed.Subscribe(ch)
}

func transaction(nonce uint64, gaslimit uint64, key *ecdsa.PrivateKey) *types.Transaction {
	return pricedTransaction(nonce, gaslimit, big.NewInt(1), key)
}

func pricedTransaction(nonce uint64, gaslimit uint64, gasprice *big.Int, key *ecdsa.PrivateKey) *types.Transaction {
	tx, _ := types.SignTx(types.NewTransaction(nonce, common.Address{0x01}, big.NewInt(100), gaslimit, gasprice, nil), types.HomesteadSigner{}, key)
	return tx
}

func pricedDataTransaction(nonce uint64, gaslimit uint64, gasprice *big.Int, key *ecdsa.PrivateKey, bytes uint64) *types.Transaction {
	data := make([]byte, bytes)
	rand.Read(data)

	tx, _ := types.SignTx(types.NewTransaction(nonce, common.Address{}, big.NewInt(0), gaslimit, gasprice, data), types.HomesteadSigner{}, key)
	return tx
}

func dynamicFeeTx(nonce uint64, gaslimit uint64, gasFee *big.Int, tip *big.Int, key *ecdsa.PrivateKey) *types.Transaction {
	tx, _ := types.SignNewTx(key, types.LatestSignerForChainID(params.TestChainConfig.ChainID), &types.DynamicFeeTx{
		ChainID:    params.TestChainConfig.ChainID,
		Nonce:      nonce,
		GasTipCap:  tip,
		GasFeeCap:  gasFee,
		Gas:        gaslimit,
		To:         &common.Address{},
		Value:      big.NewInt(100),
		Data:       nil,
		AccessList: nil,
	})
	return tx
}

func setupTxPool() (*TxPool, *ecdsa.PrivateKey) {
	return setupTxPoolWithConfig(params.TestChainConfig, testTxPoolConfig, txPoolGasLimit)
}

func setupTxPoolWithConfig(config *params.ChainConfig, txPoolConfig TxPoolConfig, gasLimit uint64, options ...func(pool *TxPool)) (*TxPool, *ecdsa.PrivateKey) {
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)

	blockchain := &testBlockChain{gasLimit, statedb, new(event.Feed)}

	key, _ := crypto.GenerateKey()

	pool := NewTxPool(txPoolConfig, config, blockchain, options...)

	// wait for the pool to initialize
	<-pool.initDoneCh
	return pool, key
}

// validateTxPoolInternals checks various consistency invariants within the pool.
func validateTxPoolInternals(pool *TxPool) error {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	// Ensure the total transaction set is consistent with pending + queued
	pending, queued := pool.stats()
	if total := pool.all.Count(); total != pending+queued {
		return fmt.Errorf("total transaction count %d != %d pending + %d queued", total, pending, queued)
	}

	pool.priced.Reheap()
	priced, remote := pool.priced.urgent.Len()+pool.priced.floating.Len(), pool.all.RemoteCount()
	if priced != remote {
		return fmt.Errorf("total priced transaction count %d != %d", priced, remote)
	}

	// Ensure the next nonce to assign is the correct one
	pool.pendingMu.RLock()
	defer pool.pendingMu.RUnlock()

	for addr, txs := range pool.pending {
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
	}

	return nil
}

// validateEvents checks that the correct number of transaction addition events
// were fired on the pool's event feed.
func validateEvents(events chan NewTxsEvent, count int) error {
	var received []*types.Transaction

	for len(received) < count {
		select {
		case ev := <-events:
			received = append(received, ev.Txs...)
		case <-time.After(time.Second):
			return fmt.Errorf("event #%d not fired", len(received))
		}
	}
	if len(received) > count {
		return fmt.Errorf("more than %d events fired: %v", count, received[count:])
	}
	select {
	case ev := <-events:
		return fmt.Errorf("more than %d events fired: %v", count, ev.Txs)

	case <-time.After(50 * time.Millisecond):
		// This branch should be "default", but it's a data race between goroutines,
		// reading the event channel and pushing into it, so better wait a bit ensuring
		// really nothing gets injected.
	}
	return nil
}

func deriveSender(tx *types.Transaction) (common.Address, error) {
	return types.Sender(types.HomesteadSigner{}, tx)
}

type testChain struct {
	*testBlockChain
	address common.Address
	trigger *bool
}

// testChain.State() is used multiple times to reset the pending state.
// when simulate is true it will create a state that indicates
// that tx0 and tx1 are included in the chain.
func (c *testChain) State() (*state.StateDB, error) {
	// delay "state change" by one. The tx pool fetches the
	// state multiple times and by delaying it a bit we simulate
	// a state change between those fetches.
	stdb := c.statedb
	if *c.trigger {
		c.statedb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		// simulate that the new head block included tx0 and tx1
		c.statedb.SetNonce(c.address, 2)
		c.statedb.SetBalance(c.address, new(big.Int).SetUint64(params.Ether))
		*c.trigger = false
	}
	return stdb, nil
}

// This test simulates a scenario where a new block is imported during a
// state reset and tests whether the pending state is in sync with the
// block head event that initiated the resetState().
func TestStateChangeDuringTransactionPoolReset(t *testing.T) {
	t.Parallel()

	var (
		key, _     = crypto.GenerateKey()
		address    = crypto.PubkeyToAddress(key.PublicKey)
		statedb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		trigger    = false
	)

	// setup pool with 2 transaction in it
	statedb.SetBalance(address, new(big.Int).SetUint64(params.Ether))
	blockchain := &testChain{&testBlockChain{1000000000, statedb, new(event.Feed)}, address, &trigger}

	tx0 := transaction(0, 100000, key)
	tx1 := transaction(1, 100000, key)

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	nonce := pool.Nonce(address)
	if nonce != 0 {
		t.Fatalf("Invalid nonce, want 0, got %d", nonce)
	}

	pool.AddRemotesSync([]*types.Transaction{tx0, tx1})

	nonce = pool.Nonce(address)
	if nonce != 2 {
		t.Fatalf("Invalid nonce, want 2, got %d", nonce)
	}

	// trigger state change in the background
	trigger = true
	<-pool.requestReset(nil, nil)

	nonce = pool.Nonce(address)
	if nonce != 2 {
		t.Fatalf("Invalid nonce, want 2, got %d", nonce)
	}
}

func testAddBalance(pool *TxPool, addr common.Address, amount *big.Int) {
	pool.mu.Lock()
	pool.currentState.AddBalance(addr, amount)
	pool.mu.Unlock()
}

func testSetNonce(pool *TxPool, addr common.Address, nonce uint64) {
	pool.mu.Lock()
	pool.currentState.SetNonce(addr, nonce)
	pool.mu.Unlock()
}

func getBalance(pool *TxPool, addr common.Address) *big.Int {
	bal := big.NewInt(0)

	pool.mu.Lock()
	bal.Set(pool.currentState.GetBalance(addr))
	pool.mu.Unlock()

	return bal
}

func TestInvalidTransactions(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	tx := transaction(0, 100, key)
	from, _ := deriveSender(tx)

	testAddBalance(pool, from, big.NewInt(1))
	if err := pool.AddRemote(tx); !errors.Is(err, ErrInsufficientFunds) {
		t.Error("expected", ErrInsufficientFunds)
	}

	balance := new(big.Int).Add(tx.Value(), new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), tx.GasPrice()))
	testAddBalance(pool, from, balance)
	if err := pool.AddRemote(tx); !errors.Is(err, ErrIntrinsicGas) {
		t.Error("expected", ErrIntrinsicGas, "got", err)
	}

	testSetNonce(pool, from, 1)
	testAddBalance(pool, from, big.NewInt(0xffffffffffffff))
	tx = transaction(0, 100000, key)
	if err := pool.AddRemote(tx); !errors.Is(err, ErrNonceTooLow) {
		t.Error("expected", ErrNonceTooLow)
	}

	tx = transaction(1, 100000, key)

	pool.gasPriceMu.Lock()

	pool.gasPrice = big.NewInt(1000)
	pool.gasPriceUint = uint256.NewInt(1000)

	pool.gasPriceMu.Unlock()

	if err := pool.AddRemote(tx); !errors.Is(err, ErrUnderpriced) {
		t.Error("expected", ErrUnderpriced, "got", err)
	}

	if err := pool.AddLocal(tx); err != nil {
		t.Error("expected", nil, "got", err)
	}
}

func TestTransactionQueue(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	tx := transaction(0, 100, key)
	from, _ := deriveSender(tx)
	testAddBalance(pool, from, big.NewInt(1000))
	<-pool.requestReset(nil, nil)

	pool.enqueueTx(tx.Hash(), tx, false, true)
	<-pool.requestPromoteExecutables(newAccountSet(pool.signer, from))

	pool.pendingMu.RLock()
	if len(pool.pending) != 1 {
		t.Error("expected valid txs to be 1 is", len(pool.pending))
	}
	pool.pendingMu.RUnlock()

	tx = transaction(1, 100, key)
	from, _ = deriveSender(tx)
	testSetNonce(pool, from, 2)
	pool.enqueueTx(tx.Hash(), tx, false, true)

	<-pool.requestPromoteExecutables(newAccountSet(pool.signer, from))

	pool.pendingMu.RLock()
	if _, ok := pool.pending[from].txs.items[tx.Nonce()]; ok {
		t.Error("expected transaction to be in tx pool")
	}
	pool.pendingMu.RUnlock()

	if len(pool.queue) > 0 {
		t.Error("expected transaction queue to be empty. is", len(pool.queue))
	}
}

func TestTransactionQueue2(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	tx1 := transaction(0, 100, key)
	tx2 := transaction(10, 100, key)
	tx3 := transaction(11, 100, key)
	from, _ := deriveSender(tx1)
	testAddBalance(pool, from, big.NewInt(1000))
	pool.reset(nil, nil)

	pool.enqueueTx(tx1.Hash(), tx1, false, true)
	pool.enqueueTx(tx2.Hash(), tx2, false, true)
	pool.enqueueTx(tx3.Hash(), tx3, false, true)

	pool.promoteExecutables([]common.Address{from})

	pool.pendingMu.RLock()
	if len(pool.pending) != 1 {
		t.Error("expected pending length to be 1, got", len(pool.pending))
	}
	pool.pendingMu.RUnlock()

	if pool.queue[from].Len() != 2 {
		t.Error("expected len(queue) == 2, got", pool.queue[from].Len())
	}
}

func TestTransactionNegativeValue(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	tx, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(-1), 100, big.NewInt(1), nil), types.HomesteadSigner{}, key)
	from, _ := deriveSender(tx)

	testAddBalance(pool, from, big.NewInt(1))

	if err := pool.AddRemote(tx); !errors.Is(err, ErrNegativeValue) {
		t.Error("expected", ErrNegativeValue, "got", err)
	}
}

func TestTransactionTipAboveFeeCap(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPoolWithConfig(eip1559Config, testTxPoolConfig, txPoolGasLimit)
	defer pool.Stop()

	tx := dynamicFeeTx(0, 100, big.NewInt(1), big.NewInt(2), key)

	if err := pool.AddRemote(tx); !errors.Is(err, ErrTipAboveFeeCap) {
		t.Error("expected", ErrTipAboveFeeCap, "got", err)
	}
}

func TestTransactionVeryHighValues(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPoolWithConfig(eip1559Config, testTxPoolConfig, txPoolGasLimit)
	defer pool.Stop()

	veryBigNumber := big.NewInt(1)
	veryBigNumber.Lsh(veryBigNumber, 300)

	tx := dynamicFeeTx(0, 100, big.NewInt(1), veryBigNumber, key)
	if err := pool.AddRemote(tx); !errors.Is(err, ErrTipVeryHigh) {
		t.Error("expected", ErrTipVeryHigh, "got", err)
	}

	tx2 := dynamicFeeTx(0, 100, veryBigNumber, big.NewInt(1), key)
	if err := pool.AddRemote(tx2); !errors.Is(err, ErrFeeCapVeryHigh) {
		t.Error("expected", ErrFeeCapVeryHigh, "got", err)
	}
}

func TestTransactionChainFork(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	resetState := func() {
		statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		statedb.AddBalance(addr, big.NewInt(100000000000000))

		pool.chain = &testBlockChain{1000000, statedb, new(event.Feed)}
		<-pool.requestReset(nil, nil)
	}
	resetState()

	tx := transaction(0, 100000, key)
	if _, err := pool.add(tx, false); err != nil {
		t.Error("didn't expect error", err)
	}
	pool.removeTx(tx.Hash(), true)

	// reset the pool's internal state
	resetState()
	if _, err := pool.add(tx, false); err != nil {
		t.Error("didn't expect error", err)
	}
}

func TestTransactionDoubleNonce(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	resetState := func() {
		statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		statedb.AddBalance(addr, big.NewInt(100000000000000))

		pool.chain = &testBlockChain{1000000, statedb, new(event.Feed)}
		<-pool.requestReset(nil, nil)
	}
	resetState()

	signer := types.HomesteadSigner{}
	tx1, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(100), 100000, big.NewInt(1), nil), signer, key)
	tx2, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(100), 1000000, big.NewInt(2), nil), signer, key)
	tx3, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(100), 1000000, big.NewInt(1), nil), signer, key)

	// Add the first two transaction, ensure higher priced stays only
	if replace, err := pool.add(tx1, false); err != nil || replace {
		t.Errorf("first transaction insert failed (%v) or reported replacement (%v)", err, replace)
	}
	if replace, err := pool.add(tx2, false); err != nil || !replace {
		t.Errorf("second transaction insert failed (%v) or not reported replacement (%v)", err, replace)
	}

	<-pool.requestPromoteExecutables(newAccountSet(signer, addr))

	pool.pendingMu.RLock()
	if pool.pending[addr].Len() != 1 {
		t.Error("expected 1 pending transactions, got", pool.pending[addr].Len())
	}
	if tx := pool.pending[addr].txs.items[0]; tx.Hash() != tx2.Hash() {
		t.Errorf("transaction mismatch: have %x, want %x", tx.Hash(), tx2.Hash())
	}
	pool.pendingMu.RUnlock()

	// Add the third transaction and ensure it's not saved (smaller price)
	pool.add(tx3, false)

	<-pool.requestPromoteExecutables(newAccountSet(signer, addr))

	pool.pendingMu.RLock()
	if pool.pending[addr].Len() != 1 {
		t.Error("expected 1 pending transactions, got", pool.pending[addr].Len())
	}
	if tx := pool.pending[addr].txs.items[0]; tx.Hash() != tx2.Hash() {
		t.Errorf("transaction mismatch: have %x, want %x", tx.Hash(), tx2.Hash())
	}
	pool.pendingMu.RUnlock()

	// Ensure the total transaction count is correct
	if pool.all.Count() != 1 {
		t.Error("expected 1 total transactions, got", pool.all.Count())
	}
}

func TestTransactionMissingNonce(t *testing.T) {
	t.Parallel()

	pool, key := setupTxPool()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, addr, big.NewInt(100000000000000))
	tx := transaction(1, 100000, key)
	if _, err := pool.add(tx, false); err != nil {
		t.Error("didn't expect error", err)
	}

	pool.pendingMu.RLock()
	if len(pool.pending) != 0 {
		t.Error("expected 0 pending transactions, got", len(pool.pending))
	}
	pool.pendingMu.RUnlock()

	if pool.queue[addr].Len() != 1 {
		t.Error("expected 1 queued transaction, got", pool.queue[addr].Len())
	}
	if pool.all.Count() != 1 {
		t.Error("expected 1 total transactions, got", pool.all.Count())
	}
}

func TestTransactionNonceRecovery(t *testing.T) {
	t.Parallel()

	const n = 10
	pool, key := setupTxPool()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	testSetNonce(pool, addr, n)
	testAddBalance(pool, addr, big.NewInt(100000000000000))
	<-pool.requestReset(nil, nil)

	tx := transaction(n, 100000, key)
	if err := pool.AddRemote(tx); err != nil {
		t.Error(err)
	}
	// simulate some weird re-order of transactions and missing nonce(s)
	testSetNonce(pool, addr, n-1)
	<-pool.requestReset(nil, nil)
	if fn := pool.Nonce(addr); fn != n-1 {
		t.Errorf("expected nonce to be %d, got %d", n-1, fn)
	}
}

// Tests that if an account runs out of funds, any pending and queued transactions
// are dropped.
func TestTransactionDropping(t *testing.T) {
	t.Parallel()

	// Create a test account and fund it
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000))

	// Add some pending and some queued transactions
	var (
		tx0  = transaction(0, 100, key)
		tx1  = transaction(1, 200, key)
		tx2  = transaction(2, 300, key)
		tx10 = transaction(10, 100, key)
		tx11 = transaction(11, 200, key)
		tx12 = transaction(12, 300, key)
	)
	pool.all.Add(tx0, false)
	pool.priced.Put(tx0, false)
	pool.promoteTx(account, tx0.Hash(), tx0)

	pool.all.Add(tx1, false)
	pool.priced.Put(tx1, false)
	pool.promoteTx(account, tx1.Hash(), tx1)

	pool.all.Add(tx2, false)
	pool.priced.Put(tx2, false)
	pool.promoteTx(account, tx2.Hash(), tx2)

	pool.enqueueTx(tx10.Hash(), tx10, false, true)
	pool.enqueueTx(tx11.Hash(), tx11, false, true)
	pool.enqueueTx(tx12.Hash(), tx12, false, true)

	// Check that pre and post validations leave the pool as is
	pool.pendingMu.RLock()
	if pool.pending[account].Len() != 3 {
		t.Errorf("pending transaction mismatch: have %d, want %d", pool.pending[account].Len(), 3)
	}
	pool.pendingMu.RUnlock()

	if pool.queue[account].Len() != 3 {
		t.Errorf("queued transaction mismatch: have %d, want %d", pool.queue[account].Len(), 3)
	}
	if pool.all.Count() != 6 {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), 6)
	}

	<-pool.requestReset(nil, nil)

	pool.pendingMu.RLock()
	if pool.pending[account].Len() != 3 {
		t.Errorf("pending transaction mismatch: have %d, want %d", pool.pending[account].Len(), 3)
	}
	pool.pendingMu.RUnlock()

	if pool.queue[account].Len() != 3 {
		t.Errorf("queued transaction mismatch: have %d, want %d", pool.queue[account].Len(), 3)
	}
	if pool.all.Count() != 6 {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), 6)
	}
	// Reduce the balance of the account, and check that invalidated transactions are dropped
	testAddBalance(pool, account, big.NewInt(-650))
	<-pool.requestReset(nil, nil)

	pool.pendingMu.RLock()
	if _, ok := pool.pending[account].txs.items[tx0.Nonce()]; !ok {
		t.Errorf("funded pending transaction missing: %v", tx0)
	}
	if _, ok := pool.pending[account].txs.items[tx1.Nonce()]; !ok {
		t.Errorf("funded pending transaction missing: %v", tx0)
	}
	if _, ok := pool.pending[account].txs.items[tx2.Nonce()]; ok {
		t.Errorf("out-of-fund pending transaction present: %v", tx1)
	}
	pool.pendingMu.RUnlock()

	if _, ok := pool.queue[account].txs.items[tx10.Nonce()]; !ok {
		t.Errorf("funded queued transaction missing: %v", tx10)
	}
	if _, ok := pool.queue[account].txs.items[tx11.Nonce()]; !ok {
		t.Errorf("funded queued transaction missing: %v", tx10)
	}
	if _, ok := pool.queue[account].txs.items[tx12.Nonce()]; ok {
		t.Errorf("out-of-fund queued transaction present: %v", tx11)
	}
	if pool.all.Count() != 4 {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), 4)
	}
	// Reduce the block gas limit, check that invalidated transactions are dropped
	atomic.StoreUint64(&pool.chain.(*testBlockChain).gasLimit, 100)
	<-pool.requestReset(nil, nil)

	pool.pendingMu.RLock()
	if _, ok := pool.pending[account].txs.items[tx0.Nonce()]; !ok {
		t.Errorf("funded pending transaction missing: %v", tx0)
	}
	if _, ok := pool.pending[account].txs.items[tx1.Nonce()]; ok {
		t.Errorf("over-gased pending transaction present: %v", tx1)
	}
	pool.pendingMu.RUnlock()

	if _, ok := pool.queue[account].txs.items[tx10.Nonce()]; !ok {
		t.Errorf("funded queued transaction missing: %v", tx10)
	}
	if _, ok := pool.queue[account].txs.items[tx11.Nonce()]; ok {
		t.Errorf("over-gased queued transaction present: %v", tx11)
	}
	if pool.all.Count() != 2 {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), 2)
	}
}

// Tests that if a transaction is dropped from the current pending pool (e.g. out
// of fund), all consecutive (still valid, but not executable) transactions are
// postponed back into the future queue to prevent broadcasting them.
func TestTransactionPostponing(t *testing.T) {
	t.Parallel()

	// Create the pool to test the postponing with
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{1000000, statedb, new(event.Feed)}

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create two test accounts to produce different gap profiles with
	keys := make([]*ecdsa.PrivateKey, 2)
	accs := make([]common.Address, len(keys))

	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		accs[i] = crypto.PubkeyToAddress(keys[i].PublicKey)

		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(50100))
	}
	// Add a batch consecutive pending transactions for validation
	txs := []*types.Transaction{}
	for i, key := range keys {

		for j := 0; j < 100; j++ {
			var tx *types.Transaction
			if (i+j)%2 == 0 {
				tx = transaction(uint64(j), 25000, key)
			} else {
				tx = transaction(uint64(j), 50000, key)
			}
			txs = append(txs, tx)
		}
	}
	for i, err := range pool.AddRemotesSync(txs) {
		if err != nil {
			t.Fatalf("tx %d: failed to add transactions: %v", i, err)
		}
	}
	// Check that pre and post validations leave the pool as is
	pool.pendingMu.RLock()
	if pending := pool.pending[accs[0]].Len() + pool.pending[accs[1]].Len(); pending != len(txs) {
		t.Errorf("pending transaction mismatch: have %d, want %d", pending, len(txs))
	}
	pool.pendingMu.RUnlock()

	if len(pool.queue) != 0 {
		t.Errorf("queued accounts mismatch: have %d, want %d", len(pool.queue), 0)
	}
	if pool.all.Count() != len(txs) {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), len(txs))
	}

	<-pool.requestReset(nil, nil)

	pool.pendingMu.RLock()
	if pending := pool.pending[accs[0]].Len() + pool.pending[accs[1]].Len(); pending != len(txs) {
		t.Errorf("pending transaction mismatch: have %d, want %d", pending, len(txs))
	}
	pool.pendingMu.RUnlock()

	if len(pool.queue) != 0 {
		t.Errorf("queued accounts mismatch: have %d, want %d", len(pool.queue), 0)
	}
	if pool.all.Count() != len(txs) {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), len(txs))
	}
	// Reduce the balance of the account, and check that transactions are reorganised
	for _, addr := range accs {
		testAddBalance(pool, addr, big.NewInt(-1))
	}
	<-pool.requestReset(nil, nil)

	// The first account's first transaction remains valid, check that subsequent
	// ones are either filtered out, or queued up for later.
	pool.pendingMu.RLock()
	if _, ok := pool.pending[accs[0]].txs.items[txs[0].Nonce()]; !ok {
		t.Errorf("tx %d: valid and funded transaction missing from pending pool: %v", 0, txs[0])
	}
	pool.pendingMu.RUnlock()

	if _, ok := pool.queue[accs[0]].txs.items[txs[0].Nonce()]; ok {
		t.Errorf("tx %d: valid and funded transaction present in future queue: %v", 0, txs[0])
	}

	pool.pendingMu.RLock()
	for i, tx := range txs[1:100] {
		if i%2 == 1 {
			if _, ok := pool.pending[accs[0]].txs.items[tx.Nonce()]; ok {
				t.Errorf("tx %d: valid but future transaction present in pending pool: %v", i+1, tx)
			}
			if _, ok := pool.queue[accs[0]].txs.items[tx.Nonce()]; !ok {
				t.Errorf("tx %d: valid but future transaction missing from future queue: %v", i+1, tx)
			}
		} else {
			if _, ok := pool.pending[accs[0]].txs.items[tx.Nonce()]; ok {
				t.Errorf("tx %d: out-of-fund transaction present in pending pool: %v", i+1, tx)
			}
			if _, ok := pool.queue[accs[0]].txs.items[tx.Nonce()]; ok {
				t.Errorf("tx %d: out-of-fund transaction present in future queue: %v", i+1, tx)
			}
		}
	}
	pool.pendingMu.RUnlock()

	// The second account's first transaction got invalid, check that all transactions
	// are either filtered out, or queued up for later.
	pool.pendingMu.RLock()
	if pool.pending[accs[1]] != nil {
		t.Errorf("invalidated account still has pending transactions")
	}
	pool.pendingMu.RUnlock()

	for i, tx := range txs[100:] {
		if i%2 == 1 {
			if _, ok := pool.queue[accs[1]].txs.items[tx.Nonce()]; !ok {
				t.Errorf("tx %d: valid but future transaction missing from future queue: %v", 100+i, tx)
			}
		} else {
			if _, ok := pool.queue[accs[1]].txs.items[tx.Nonce()]; ok {
				t.Errorf("tx %d: out-of-fund transaction present in future queue: %v", 100+i, tx)
			}
		}
	}
	if pool.all.Count() != len(txs)/2 {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), len(txs)/2)
	}
}

// Tests that if the transaction pool has both executable and non-executable
// transactions from an origin account, filling the nonce gap moves all queued
// ones into the pending pool.
func TestTransactionGapFilling(t *testing.T) {
	t.Parallel()

	// Create a test account and fund it
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000))

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsEvent, testTxPoolConfig.AccountQueue+5)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a pending and a queued transaction with a nonce-gap in between
	pool.AddRemotesSync([]*types.Transaction{
		transaction(0, 100000, key),
		transaction(2, 100000, key),
	})
	pending, queued := pool.Stats()
	if pending != 1 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 1)
	}
	if queued != 1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
	}

	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("original event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Fill the nonce gap and ensure all transactions become pending
	if err := pool.addRemoteSync(transaction(1, 100000, key)); err != nil {
		t.Fatalf("failed to add gapped transaction: %v", err)
	}
	pending, queued = pool.Stats()
	if pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("gap-filling event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that if the transaction count belonging to a single account goes above
// some threshold, the higher transactions are dropped to prevent DOS attacks.
func TestTransactionQueueAccountLimiting(t *testing.T) {
	t.Parallel()

	// Create a test account and fund it
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000))

	// Keep queuing up transactions and make sure all above a limit are dropped
	for i := uint64(1); i <= testTxPoolConfig.AccountQueue+5; i++ {
		if err := pool.addRemoteSync(transaction(i, 100000, key)); err != nil {
			t.Fatalf("tx %d: failed to add transaction: %v", i, err)
		}

		pool.pendingMu.RLock()
		if len(pool.pending) != 0 {
			t.Errorf("tx %d: pending pool size mismatch: have %d, want %d", i, len(pool.pending), 0)
		}
		pool.pendingMu.RUnlock()

		if i <= testTxPoolConfig.AccountQueue {
			if pool.queue[account].Len() != int(i) {
				t.Errorf("tx %d: queue size mismatch: have %d, want %d", i, pool.queue[account].Len(), i)
			}
		} else {
			if pool.queue[account].Len() != int(testTxPoolConfig.AccountQueue) {
				t.Errorf("tx %d: queue limit mismatch: have %d, want %d", i, pool.queue[account].Len(), testTxPoolConfig.AccountQueue)
			}
		}
	}
	if pool.all.Count() != int(testTxPoolConfig.AccountQueue) {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), testTxPoolConfig.AccountQueue)
	}
}

// Test that txpool rejects unprotected txs by default
// FIXME: The below test causes some tests to fail randomly (probably due to parallel execution)
//
//nolint:paralleltest
func TestRejectUnprotectedTransaction(t *testing.T) {
	//nolint:paralleltest
	t.Skip()

	pool, key := setupTxPool()
	defer pool.Stop()

	tx := dynamicFeeTx(0, 22000, big.NewInt(5), big.NewInt(2), key)
	from := crypto.PubkeyToAddress(key.PublicKey)

	pool.chainconfig.ChainID = big.NewInt(5)
	pool.signer = types.LatestSignerForChainID(pool.chainconfig.ChainID)
	testAddBalance(pool, from, big.NewInt(0xffffffffffffff))

	if err := pool.AddRemote(tx); !errors.Is(err, types.ErrInvalidChainId) {
		t.Error("expected", types.ErrInvalidChainId, "got", err)
	}
}

// Test that txpool allows unprotected txs when AllowUnprotectedTxs flag is set
// FIXME: The below test causes some tests to fail randomly (probably due to parallel execution)
//
//nolint:paralleltest
func TestAllowUnprotectedTransactionWhenSet(t *testing.T) {
	t.Skip()

	pool, key := setupTxPool()
	defer pool.Stop()

	tx := dynamicFeeTx(0, 22000, big.NewInt(5), big.NewInt(2), key)
	from := crypto.PubkeyToAddress(key.PublicKey)

	// Allow unprotected txs
	pool.config.AllowUnprotectedTxs = true
	pool.chainconfig.ChainID = big.NewInt(5)
	pool.signer = types.LatestSignerForChainID(pool.chainconfig.ChainID)
	testAddBalance(pool, from, big.NewInt(0xffffffffffffff))

	if err := pool.AddRemote(tx); err != nil {
		t.Error("expected", nil, "got", err)
	}
}

// Tests that if the transaction count belonging to multiple accounts go above
// some threshold, the higher transactions are dropped to prevent DOS attacks.
//
// This logic should not hold for local transactions, unless the local tracking
// mechanism is disabled.
func TestTransactionQueueGlobalLimiting(t *testing.T) {
	testTransactionQueueGlobalLimiting(t, false)
}
func TestTransactionQueueGlobalLimitingNoLocals(t *testing.T) {
	testTransactionQueueGlobalLimiting(t, true)
}

func testTransactionQueueGlobalLimiting(t *testing.T, nolocals bool) {
	t.Parallel()

	// Create the pool to test the limit enforcement with
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{1000000, statedb, new(event.Feed)}

	config := testTxPoolConfig
	config.NoLocals = nolocals
	config.GlobalQueue = config.AccountQueue*3 - 1 // reduce the queue limits to shorten test time (-1 to make it non divisible)

	pool := NewTxPool(config, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create a number of test accounts and fund them (last one will be the local)
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	local := keys[len(keys)-1]

	// Generate and queue a batch of transactions
	nonces := make(map[common.Address]uint64)

	txs := make(types.Transactions, 0, 3*config.GlobalQueue)
	for len(txs) < cap(txs) {
		key := keys[rand.Intn(len(keys)-1)] // skip adding transactions with the local account
		addr := crypto.PubkeyToAddress(key.PublicKey)

		txs = append(txs, transaction(nonces[addr]+1, 100000, key))
		nonces[addr]++
	}
	// Import the batch and verify that limits have been enforced
	pool.AddRemotesSync(txs)

	queued := 0
	for addr, list := range pool.queue {
		if list.Len() > int(config.AccountQueue) {
			t.Errorf("addr %x: queued accounts overflown allowance: %d > %d", addr, list.Len(), config.AccountQueue)
		}
		queued += list.Len()
	}
	if queued > int(config.GlobalQueue) {
		t.Fatalf("total transactions overflow allowance: %d > %d", queued, config.GlobalQueue)
	}
	// Generate a batch of transactions from the local account and import them
	txs = txs[:0]
	for i := uint64(0); i < 3*config.GlobalQueue; i++ {
		txs = append(txs, transaction(i+1, 100000, local))
	}

	pool.AddLocals(txs)

	// If locals are disabled, the previous eviction algorithm should apply here too
	if nolocals {
		queued := 0
		for addr, list := range pool.queue {
			if list.Len() > int(config.AccountQueue) {
				t.Errorf("addr %x: queued accounts overflown allowance: %d > %d", addr, list.Len(), config.AccountQueue)
			}
			queued += list.Len()
		}
		if queued > int(config.GlobalQueue) {
			t.Fatalf("total transactions overflow allowance: %d > %d", queued, config.GlobalQueue)
		}
	} else {
		// Local exemptions are enabled, make sure the local account owned the queue
		if len(pool.queue) != 1 {
			t.Errorf("multiple accounts in queue: have %v, want %v", len(pool.queue), 1)
		}
		// Also ensure no local transactions are ever dropped, even if above global limits
		if queued := pool.queue[crypto.PubkeyToAddress(local.PublicKey)].Len(); uint64(queued) != 3*config.GlobalQueue {
			t.Fatalf("local account queued transaction count mismatch: have %v, want %v", queued, 3*config.GlobalQueue)
		}
	}
}

// Tests that if an account remains idle for a prolonged amount of time, any
// non-executable transactions queued up are dropped to prevent wasting resources
// on shuffling them around.
//
// This logic should not hold for local transactions, unless the local tracking
// mechanism is disabled.
func TestTransactionQueueTimeLimiting(t *testing.T) {
	testTransactionQueueTimeLimiting(t, false)
}
func TestTransactionQueueTimeLimitingNoLocals(t *testing.T) {
	testTransactionQueueTimeLimiting(t, true)
}

func testTransactionQueueTimeLimiting(t *testing.T, nolocals bool) {
	// Reduce the eviction interval to a testable amount
	defer func(old time.Duration) { evictionInterval = old }(evictionInterval)
	evictionInterval = time.Millisecond * 100

	// Create the pool to test the non-expiration enforcement
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{1000000, statedb, new(event.Feed)}

	config := testTxPoolConfig
	config.Lifetime = time.Second
	config.NoLocals = nolocals

	pool := NewTxPool(config, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create two test accounts to ensure remotes expire but locals do not
	local, _ := crypto.GenerateKey()
	remote, _ := crypto.GenerateKey()

	testAddBalance(pool, crypto.PubkeyToAddress(local.PublicKey), big.NewInt(1000000000))
	testAddBalance(pool, crypto.PubkeyToAddress(remote.PublicKey), big.NewInt(1000000000))

	// Add the two transactions and ensure they both are queued up
	if err := pool.AddLocal(pricedTransaction(1, 100000, big.NewInt(1), local)); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}
	if err := pool.AddRemote(pricedTransaction(1, 100000, big.NewInt(1), remote)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	pending, queued := pool.Stats()
	if pending != 0 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
	}
	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Allow the eviction interval to run
	time.Sleep(2 * evictionInterval)

	// Transactions should not be evicted from the queue yet since lifetime duration has not passed
	pending, queued = pool.Stats()
	if pending != 0 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
	}
	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Wait a bit for eviction to run and clean up any leftovers, and ensure only the local remains
	time.Sleep(2 * config.Lifetime)

	pending, queued = pool.Stats()
	if pending != 0 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
	}
	if nolocals {
		if queued != 0 {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
		}
	} else {
		if queued != 1 {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
		}
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// remove current transactions and increase nonce to prepare for a reset and cleanup
	statedb.SetNonce(crypto.PubkeyToAddress(remote.PublicKey), 2)
	statedb.SetNonce(crypto.PubkeyToAddress(local.PublicKey), 2)
	<-pool.requestReset(nil, nil)

	// make sure queue, pending are cleared
	pending, queued = pool.Stats()
	if pending != 0 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Queue gapped transactions
	if err := pool.AddLocal(pricedTransaction(4, 100000, big.NewInt(1), local)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(4, 100000, big.NewInt(1), remote)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	time.Sleep(5 * evictionInterval) // A half lifetime pass

	// Queue executable transactions, the life cycle should be restarted.
	if err := pool.AddLocal(pricedTransaction(2, 100000, big.NewInt(1), local)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(2, 100000, big.NewInt(1), remote)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	time.Sleep(6 * evictionInterval)

	// All gapped transactions shouldn't be kicked out
	pending, queued = pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 3)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// The whole life time pass after last promotion, kick out stale transactions
	time.Sleep(2 * config.Lifetime)
	pending, queued = pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if nolocals {
		if queued != 0 {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
		}
	} else {
		if queued != 1 {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
		}
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that even if the transaction count belonging to a single account goes
// above some threshold, as long as the transactions are executable, they are
// accepted.
func TestTransactionPendingLimiting(t *testing.T) {
	t.Parallel()

	// Create a test account and fund it
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000))

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsEvent, testTxPoolConfig.AccountQueue+5)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Keep queuing up transactions and make sure all above a limit are dropped
	for i := uint64(0); i < testTxPoolConfig.AccountQueue+5; i++ {
		if err := pool.addRemoteSync(transaction(i, 100000, key)); err != nil {
			t.Fatalf("tx %d: failed to add transaction: %v", i, err)
		}

		pool.pendingMu.RLock()
		if pool.pending[account].Len() != int(i)+1 {
			t.Errorf("tx %d: pending pool size mismatch: have %d, want %d", i, pool.pending[account].Len(), i+1)
		}
		pool.pendingMu.RUnlock()

		if len(pool.queue) != 0 {
			t.Errorf("tx %d: queue size mismatch: have %d, want %d", i, pool.queue[account].Len(), 0)
		}
	}
	if pool.all.Count() != int(testTxPoolConfig.AccountQueue+5) {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), testTxPoolConfig.AccountQueue+5)
	}
	if err := validateEvents(events, int(testTxPoolConfig.AccountQueue+5)); err != nil {
		t.Fatalf("event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that if the transaction count belonging to multiple accounts go above
// some hard threshold, the higher transactions are dropped to prevent DOS
// attacks.
func TestTransactionPendingGlobalLimiting(t *testing.T) {
	t.Parallel()

	// Create the pool to test the limit enforcement with
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{1000000, statedb, new(event.Feed)}

	config := testTxPoolConfig
	config.GlobalSlots = config.AccountSlots * 10

	pool := NewTxPool(config, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Generate and queue a batch of transactions
	nonces := make(map[common.Address]uint64)

	txs := types.Transactions{}
	for _, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)
		for j := 0; j < int(config.GlobalSlots)/len(keys)*2; j++ {
			txs = append(txs, transaction(nonces[addr], 100000, key))
			nonces[addr]++
		}
	}
	// Import the batch and verify that limits have been enforced
	pool.AddRemotesSync(txs)

	pending := 0

	pool.pendingMu.RLock()
	for _, list := range pool.pending {
		pending += list.Len()
	}
	pool.pendingMu.RUnlock()

	if pending > int(config.GlobalSlots) {
		t.Fatalf("total pending transactions overflow allowance: %d > %d", pending, config.GlobalSlots)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Test the limit on transaction size is enforced correctly.
// This test verifies every transaction having allowed size
// is added to the pool, and longer transactions are rejected.
func TestTransactionAllowedTxSize(t *testing.T) {
	t.Parallel()

	// Create a test account and fund it
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000000))

	// Compute maximal data size for transactions (lower bound).
	//
	// It is assumed the fields in the transaction (except of the data) are:
	//   - nonce     <= 32 bytes
	//   - gasPrice  <= 32 bytes
	//   - gasLimit  <= 32 bytes
	//   - recipient == 20 bytes
	//   - value     <= 32 bytes
	//   - signature == 65 bytes
	// All those fields are summed up to at most 213 bytes.
	baseSize := uint64(213)
	dataSize := txMaxSize - baseSize

	// Try adding a transaction with maximal allowed size
	tx := pricedDataTransaction(0, pool.currentMaxGas, big.NewInt(1), key, dataSize)
	if err := pool.addRemoteSync(tx); err != nil {
		t.Fatalf("failed to add transaction of size %d, close to maximal: %v", int(tx.Size()), err)
	}
	// Try adding a transaction with random allowed size
	if err := pool.addRemoteSync(pricedDataTransaction(1, pool.currentMaxGas, big.NewInt(1), key, uint64(rand.Intn(int(dataSize))))); err != nil {
		t.Fatalf("failed to add transaction of random allowed size: %v", err)
	}
	// Try adding a transaction of minimal not allowed size
	if err := pool.addRemoteSync(pricedDataTransaction(2, pool.currentMaxGas, big.NewInt(1), key, txMaxSize)); err == nil {
		t.Fatalf("expected rejection on slightly oversize transaction")
	}
	// Try adding a transaction of random not allowed size
	if err := pool.addRemoteSync(pricedDataTransaction(2, pool.currentMaxGas, big.NewInt(1), key, dataSize+1+uint64(rand.Intn(10*txMaxSize)))); err == nil {
		t.Fatalf("expected rejection on oversize transaction")
	}
	// Run some sanity checks on the pool internals
	pending, queued := pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that if transactions start being capped, transactions are also removed from 'all'
func TestTransactionCapClearsFromAll(t *testing.T) {
	t.Parallel()

	// Create the pool to test the limit enforcement with
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{1000000, statedb, new(event.Feed)}

	config := testTxPoolConfig
	config.AccountSlots = 2
	config.AccountQueue = 2
	config.GlobalSlots = 8

	pool := NewTxPool(config, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create a number of test accounts and fund them
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, addr, big.NewInt(1000000))

	txs := types.Transactions{}
	for j := 0; j < int(config.GlobalSlots)*2; j++ {
		txs = append(txs, transaction(uint64(j), 100000, key))
	}
	// Import the batch and verify that limits have been enforced
	pool.AddRemotes(txs)
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that if the transaction count belonging to multiple accounts go above
// some hard threshold, if they are under the minimum guaranteed slot count then
// the transactions are still kept.
func TestTransactionPendingMinimumAllowance(t *testing.T) {
	t.Parallel()

	// Create the pool to test the limit enforcement with
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{1000000, statedb, new(event.Feed)}

	config := testTxPoolConfig
	config.GlobalSlots = 1

	pool := NewTxPool(config, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Generate and queue a batch of transactions
	nonces := make(map[common.Address]uint64)

	txs := types.Transactions{}
	for _, key := range keys {
		addr := crypto.PubkeyToAddress(key.PublicKey)
		for j := 0; j < int(config.AccountSlots)*2; j++ {
			txs = append(txs, transaction(nonces[addr], 100000, key))
			nonces[addr]++
		}
	}
	// Import the batch and verify that limits have been enforced
	pool.AddRemotesSync(txs)

	pool.pendingMu.RLock()
	for addr, list := range pool.pending {
		if list.Len() != int(config.AccountSlots) {
			t.Errorf("addr %x: total pending transactions mismatch: have %d, want %d", addr, list.Len(), config.AccountSlots)
		}
	}
	pool.pendingMu.RUnlock()

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that setting the transaction pool gas price to a higher value correctly
// discards everything cheaper than that and moves any gapped transactions back
// from the pending pool to the queue.
//
// Note, local transactions are never allowed to be dropped.
func TestTransactionPoolRepricing(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{1000000, statedb, new(event.Feed)}

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsEvent, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 4)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Generate and queue a batch of transactions, both pending and queued
	txs := types.Transactions{}

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(2), keys[0]))
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(1), keys[0]))
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(2), keys[0]))

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(1), keys[1]))
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(2), keys[1]))
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(2), keys[1]))

	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(2), keys[2]))
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(1), keys[2]))
	txs = append(txs, pricedTransaction(3, 100000, big.NewInt(2), keys[2]))

	ltx := pricedTransaction(0, 100000, big.NewInt(1), keys[3])

	// Import the batch and that both pending and queued transactions match up
	pool.AddRemotesSync(txs)
	pool.AddLocal(ltx)

	pending, queued := pool.Stats()
	if pending != 7 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 7)
	}

	if queued != 3 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 3)
	}

	if err := validateEvents(events, 7); err != nil {
		t.Fatalf("original event firing failed: %v", err)
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Reprice the pool and check that underpriced transactions get dropped
	pool.SetGasPrice(big.NewInt(2))

	pending, queued = pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}

	if queued != 5 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 5)
	}

	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("reprice event firing failed: %v", err)
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Check that we can't add the old transactions back
	if err := pool.AddRemote(pricedTransaction(1, 100000, big.NewInt(1), keys[0])); !errors.Is(err, ErrUnderpriced) {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}

	if err := pool.AddRemote(pricedTransaction(0, 100000, big.NewInt(1), keys[1])); !errors.Is(err, ErrUnderpriced) {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}

	if err := pool.AddRemote(pricedTransaction(2, 100000, big.NewInt(1), keys[2])); !errors.Is(err, ErrUnderpriced) {
		t.Fatalf("adding underpriced queued transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}

	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("post-reprice event firing failed: %v", err)
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// However we can add local underpriced transactions
	tx := pricedTransaction(1, 100000, big.NewInt(1), keys[3])

	if err := pool.AddLocal(tx); err != nil {
		t.Fatalf("failed to add underpriced local transaction: %v", err)
	}

	if pending, _ = pool.Stats(); pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}

	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("post-reprice local event firing failed: %v", err)
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// And we can fill gaps with properly priced transactions
	if err := pool.AddRemote(pricedTransaction(1, 100000, big.NewInt(2), keys[0])); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}

	if err := pool.AddRemote(pricedTransaction(0, 100000, big.NewInt(2), keys[1])); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}

	if err := pool.AddRemote(pricedTransaction(2, 100000, big.NewInt(2), keys[2])); err != nil {
		t.Fatalf("failed to add queued transaction: %v", err)
	}

	if err := validateEvents(events, 5); err != nil {
		t.Fatalf("post-reprice event firing failed: %v", err)
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that setting the transaction pool gas price to a higher value correctly
// discards everything cheaper (legacy & dynamic fee) than that and moves any
// gapped transactions back from the pending pool to the queue.
//
// Note, local transactions are never allowed to be dropped.
func TestTransactionPoolRepricingDynamicFee(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	pool, _ := setupTxPoolWithConfig(eip1559Config, testTxPoolConfig, txPoolGasLimit)
	defer pool.Stop()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsEvent, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 4)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}

	// Generate and queue a batch of transactions, both pending and queued
	txs := types.Transactions{}

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(2), keys[0]))
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(1), keys[0]))
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(2), keys[0]))

	txs = append(txs, dynamicFeeTx(0, 100000, big.NewInt(2), big.NewInt(1), keys[1]))
	txs = append(txs, dynamicFeeTx(1, 100000, big.NewInt(3), big.NewInt(2), keys[1]))
	txs = append(txs, dynamicFeeTx(2, 100000, big.NewInt(3), big.NewInt(2), keys[1]))

	txs = append(txs, dynamicFeeTx(1, 100000, big.NewInt(2), big.NewInt(2), keys[2]))
	txs = append(txs, dynamicFeeTx(2, 100000, big.NewInt(1), big.NewInt(1), keys[2]))
	txs = append(txs, dynamicFeeTx(3, 100000, big.NewInt(2), big.NewInt(2), keys[2]))

	ltx := dynamicFeeTx(0, 100000, big.NewInt(2), big.NewInt(1), keys[3])

	// Import the batch and that both pending and queued transactions match up
	pool.AddRemotesSync(txs)
	pool.AddLocal(ltx)

	pending, queued := pool.Stats()
	if pending != 7 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 7)
	}

	if queued != 3 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 3)
	}

	if err := validateEvents(events, 7); err != nil {
		t.Fatalf("original event firing failed: %v", err)
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Reprice the pool and check that underpriced transactions get dropped
	pool.SetGasPrice(big.NewInt(2))

	pending, queued = pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}

	if queued != 5 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 5)
	}

	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("reprice event firing failed: %v", err)
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Check that we can't add the old transactions back
	tx := pricedTransaction(1, 100000, big.NewInt(1), keys[0])

	if err := pool.AddRemote(tx); !errors.Is(err, ErrUnderpriced) {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}

	tx = dynamicFeeTx(0, 100000, big.NewInt(2), big.NewInt(1), keys[1])

	if err := pool.AddRemote(tx); !errors.Is(err, ErrUnderpriced) {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}

	tx = dynamicFeeTx(2, 100000, big.NewInt(1), big.NewInt(1), keys[2])
	if err := pool.AddRemote(tx); !errors.Is(err, ErrUnderpriced) {
		t.Fatalf("adding underpriced queued transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}

	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("post-reprice event firing failed: %v", err)
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// However we can add local underpriced transactions
	tx = dynamicFeeTx(1, 100000, big.NewInt(1), big.NewInt(1), keys[3])

	if err := pool.AddLocal(tx); err != nil {
		t.Fatalf("failed to add underpriced local transaction: %v", err)
	}

	if pending, _ = pool.Stats(); pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}

	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("post-reprice local event firing failed: %v", err)
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// And we can fill gaps with properly priced transactions
	tx = pricedTransaction(1, 100000, big.NewInt(2), keys[0])

	if err := pool.AddRemote(tx); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}

	tx = dynamicFeeTx(0, 100000, big.NewInt(3), big.NewInt(2), keys[1])

	if err := pool.AddRemote(tx); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}

	tx = dynamicFeeTx(2, 100000, big.NewInt(2), big.NewInt(2), keys[2])

	if err := pool.AddRemote(tx); err != nil {
		t.Fatalf("failed to add queued transaction: %v", err)
	}

	if err := validateEvents(events, 5); err != nil {
		t.Fatalf("post-reprice event firing failed: %v", err)
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that setting the transaction pool gas price to a higher value does not
// remove local transactions (legacy & dynamic fee).
func TestTransactionPoolRepricingKeepsLocals(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{1000000, statedb, new(event.Feed)}

	pool := NewTxPool(testTxPoolConfig, eip1559Config, blockchain)
	defer pool.Stop()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 3)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000*1000000))
	}
	// Create transaction (both pending and queued) with a linearly growing gasprice
	for i := uint64(0); i < 500; i++ {
		// Add pending transaction.
		pendingTx := pricedTransaction(i, 100000, big.NewInt(int64(i)), keys[2])
		if err := pool.AddLocal(pendingTx); err != nil {
			t.Fatal(err)
		}
		// Add queued transaction.
		queuedTx := pricedTransaction(i+501, 100000, big.NewInt(int64(i)), keys[2])
		if err := pool.AddLocal(queuedTx); err != nil {
			t.Fatal(err)
		}

		// Add pending dynamic fee transaction.
		pendingTx = dynamicFeeTx(i, 100000, big.NewInt(int64(i)+1), big.NewInt(int64(i)), keys[1])
		if err := pool.AddLocal(pendingTx); err != nil {
			t.Fatal(err)
		}
		// Add queued dynamic fee transaction.
		queuedTx = dynamicFeeTx(i+501, 100000, big.NewInt(int64(i)+1), big.NewInt(int64(i)), keys[1])
		if err := pool.AddLocal(queuedTx); err != nil {
			t.Fatal(err)
		}
	}
	pending, queued := pool.Stats()
	expPending, expQueued := 1000, 1000
	validate := func() {
		pending, queued = pool.Stats()
		if pending != expPending {
			t.Fatalf("pending transactions mismatched: have %d, want %d", pending, expPending)
		}
		if queued != expQueued {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, expQueued)
		}

		if err := validateTxPoolInternals(pool); err != nil {
			t.Fatalf("pool internal state corrupted: %v", err)
		}
	}
	validate()

	// Reprice the pool and check that nothing is dropped
	pool.SetGasPrice(big.NewInt(2))
	validate()

	pool.SetGasPrice(big.NewInt(2))
	pool.SetGasPrice(big.NewInt(4))
	pool.SetGasPrice(big.NewInt(8))
	pool.SetGasPrice(big.NewInt(100))
	validate()
}

// Tests that when the pool reaches its global transaction limit, underpriced
// transactions are gradually shifted out for more expensive ones and any gapped
// pending transactions are moved into the queue.
//
// Note, local transactions are never allowed to be dropped.
func TestTransactionPoolUnderpricing(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{1000000, statedb, new(event.Feed)}

	config := testTxPoolConfig
	config.GlobalSlots = 2
	config.GlobalQueue = 2

	pool := NewTxPool(config, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsEvent, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 4)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Generate and queue a batch of transactions, both pending and queued
	txs := types.Transactions{}

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(1), keys[0]))
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(2), keys[0]))

	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(1), keys[1]))

	ltx := pricedTransaction(0, 100000, big.NewInt(1), keys[2])

	// Import the batch and that both pending and queued transactions match up
	pool.AddRemotes(txs)
	pool.AddLocal(ltx)

	pending, queued := pool.Stats()
	if pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}
	if queued != 1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
	}
	if err := validateEvents(events, 3); err != nil {
		t.Fatalf("original event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Ensure that adding an underpriced transaction on block limit fails
	if err := pool.AddRemote(pricedTransaction(0, 100000, big.NewInt(1), keys[1])); !errors.Is(err, ErrUnderpriced) {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}
	// Ensure that adding high priced transactions drops cheap ones, but not own
	if err := pool.AddRemote(pricedTransaction(0, 100000, big.NewInt(3), keys[1])); err != nil { // +K1:0 => -K1:1 => Pend K0:0, K0:1, K1:0, K2:0; Que -
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	if err := pool.AddRemote(pricedTransaction(2, 100000, big.NewInt(4), keys[1])); err != nil { // +K1:2 => -K0:0 => Pend K1:0, K2:0; Que K0:1 K1:2
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	if err := pool.AddRemote(pricedTransaction(3, 100000, big.NewInt(5), keys[1])); err != nil { // +K1:3 => -K0:1 => Pend K1:0, K2:0; Que K1:2 K1:3
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	pending, queued = pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}

	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("additional event firing failed: %v", err)
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Ensure that adding local transactions can push out even higher priced ones
	ltx = pricedTransaction(1, 100000, big.NewInt(0), keys[2])
	if err := pool.AddLocal(ltx); err != nil {
		t.Fatalf("failed to append underpriced local transaction: %v", err)
	}
	ltx = pricedTransaction(0, 100000, big.NewInt(0), keys[3])
	if err := pool.AddLocal(ltx); err != nil {
		t.Fatalf("failed to add new underpriced local transaction: %v", err)
	}
	pending, queued = pool.Stats()
	if pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}
	if queued != 1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
	}
	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("local event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that more expensive transactions push out cheap ones from the pool, but
// without producing instability by creating gaps that start jumping transactions
// back and forth between queued/pending.
func TestTransactionPoolStableUnderpricing(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{1000000, statedb, new(event.Feed)}

	config := testTxPoolConfig
	config.GlobalSlots = 128
	config.GlobalQueue = 0

	pool := NewTxPool(config, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsEvent, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 2)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Fill up the entire queue with the same transaction price points
	txs := types.Transactions{}
	for i := uint64(0); i < config.GlobalSlots; i++ {
		txs = append(txs, pricedTransaction(i, 100000, big.NewInt(1), keys[0]))
	}
	pool.AddRemotesSync(txs)

	pending, queued := pool.Stats()
	if pending != int(config.GlobalSlots) {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, config.GlobalSlots)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateEvents(events, int(config.GlobalSlots)); err != nil {
		t.Fatalf("original event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Ensure that adding high priced transactions drops a cheap, but doesn't produce a gap
	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(3), keys[1])); err != nil {
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	pending, queued = pool.Stats()
	if pending != int(config.GlobalSlots) {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, config.GlobalSlots)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("additional event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that when the pool reaches its global transaction limit, underpriced
// transactions (legacy & dynamic fee) are gradually shifted out for more
// expensive ones and any gapped pending transactions are moved into the queue.
//
// Note, local transactions are never allowed to be dropped.
func TestTransactionPoolUnderpricingDynamicFee(t *testing.T) {
	t.Parallel()

	pool, _ := setupTxPoolWithConfig(eip1559Config, testTxPoolConfig, txPoolGasLimit)
	defer pool.Stop()

	pool.config.GlobalSlots = 2
	pool.config.GlobalQueue = 2

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsEvent, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 4)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}

	// Generate and queue a batch of transactions, both pending and queued
	txs := types.Transactions{}

	txs = append(txs, dynamicFeeTx(0, 100000, big.NewInt(3), big.NewInt(2), keys[0]))
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(2), keys[0]))
	txs = append(txs, dynamicFeeTx(1, 100000, big.NewInt(2), big.NewInt(1), keys[1]))

	ltx := dynamicFeeTx(0, 100000, big.NewInt(2), big.NewInt(1), keys[2])

	// Import the batch and that both pending and queued transactions match up
	pool.AddRemotes(txs) // Pend K0:0, K0:1; Que K1:1
	pool.AddLocal(ltx)   // +K2:0 => Pend K0:0, K0:1, K2:0; Que K1:1

	pending, queued := pool.Stats()
	if pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}
	if queued != 1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
	}
	if err := validateEvents(events, 3); err != nil {
		t.Fatalf("original event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Ensure that adding an underpriced transaction fails
	tx := dynamicFeeTx(0, 100000, big.NewInt(2), big.NewInt(1), keys[1])
	if err := pool.AddRemote(tx); !errors.Is(err, ErrUnderpriced) { // Pend K0:0, K0:1, K2:0; Que K1:1
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, ErrUnderpriced)
	}

	// Ensure that adding high priced transactions drops cheap ones, but not own
	tx = pricedTransaction(0, 100000, big.NewInt(2), keys[1])
	if err := pool.AddRemote(tx); err != nil { // +K1:0, -K1:1 => Pend K0:0, K0:1, K1:0, K2:0; Que -
		t.Fatalf("failed to add well priced transaction: %v", err)
	}

	tx = pricedTransaction(2, 100000, big.NewInt(3), keys[1])
	if err := pool.AddRemote(tx); err != nil { // +K1:2, -K0:1 => Pend K0:0 K1:0, K2:0; Que K1:2
		t.Fatalf("failed to add well priced transaction: %v", err)
	}

	tx = dynamicFeeTx(3, 100000, big.NewInt(4), big.NewInt(1), keys[1])
	if err := pool.AddRemote(tx); err != nil { // +K1:3, -K1:0 => Pend K0:0 K2:0; Que K1:2 K1:3
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	pending, queued = pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}

	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("additional event firing failed: %v", err)
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Ensure that adding local transactions can push out even higher priced ones
	ltx = dynamicFeeTx(1, 100000, big.NewInt(0), big.NewInt(0), keys[2])
	if err := pool.AddLocal(ltx); err != nil {
		t.Fatalf("failed to append underpriced local transaction: %v", err)
	}
	ltx = dynamicFeeTx(0, 100000, big.NewInt(0), big.NewInt(0), keys[3])
	if err := pool.AddLocal(ltx); err != nil {
		t.Fatalf("failed to add new underpriced local transaction: %v", err)
	}
	pending, queued = pool.Stats()
	if pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}
	if queued != 1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
	}
	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("local event firing failed: %v", err)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests whether highest fee cap transaction is retained after a batch of high effective
// tip transactions are added and vice versa
func TestDualHeapEviction(t *testing.T) {
	t.Parallel()

	pool, _ := setupTxPoolWithConfig(eip1559Config, testTxPoolConfig, txPoolGasLimit)
	defer pool.Stop()

	pool.config.GlobalSlots = 10
	pool.config.GlobalQueue = 10

	var (
		highTip, highCap *types.Transaction
		baseFee          int
	)

	check := func(tx *types.Transaction, name string) {
		if pool.all.GetRemote(tx.Hash()) == nil {
			t.Fatalf("highest %s transaction evicted from the pool", name)
		}
	}

	add := func(urgent bool) {
		for i := 0; i < 20; i++ {
			var tx *types.Transaction
			// Create a test accounts and fund it
			key, _ := crypto.GenerateKey()
			testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000000000))
			if urgent {
				tx = dynamicFeeTx(0, 100000, big.NewInt(int64(baseFee+1+i)), big.NewInt(int64(1+i)), key)
				highTip = tx
			} else {
				tx = dynamicFeeTx(0, 100000, big.NewInt(int64(baseFee+200+i)), big.NewInt(1), key)
				highCap = tx
			}
			pool.AddRemotesSync([]*types.Transaction{tx})
		}
		pending, queued := pool.Stats()
		if pending+queued != 20 {
			t.Fatalf("transaction count mismatch: have %d, want %d", pending+queued, 10)
		}
	}

	add(false)
	for baseFee = 0; baseFee <= 1000; baseFee += 100 {
		pool.priced.SetBaseFee(uint256.NewInt(uint64(baseFee)))
		add(true)
		check(highCap, "fee cap")
		add(false)
		check(highTip, "effective tip")
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that the pool rejects duplicate transactions.
func TestTransactionDeduplication(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{1000000, statedb, new(event.Feed)}

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create a test account to add transactions with
	key, _ := crypto.GenerateKey()
	testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000000))

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

	if len(errs) != 0 {
		t.Fatalf("first add mismatching result count: have %d, want %d", len(errs), 0)
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
	if len(errs) != 0 {
		t.Fatalf("all add mismatching result count: have %d, want %d", len(errs), 0)
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
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, len(txs))
	}

	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that the pool rejects replacement transactions that don't meet the minimum
// price bump required.
func TestTransactionReplacement(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{1000000, statedb, new(event.Feed)}

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsEvent, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a test account to add transactions with
	key, _ := crypto.GenerateKey()
	testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000000))

	// Add pending transactions, ensuring the minimum price bump is enforced for replacement (for ultra low prices too)
	price := int64(100)
	threshold := (price * (100 + int64(testTxPoolConfig.PriceBump))) / 100

	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(1), key)); err != nil {
		t.Fatalf("failed to add original cheap pending transaction: %v", err)
	}

	if err := pool.AddRemote(pricedTransaction(0, 100001, big.NewInt(1), key)); !errors.Is(err, ErrReplaceUnderpriced) {
		t.Fatalf("original cheap pending transaction replacement error mismatch: have %v, want %v", err, ErrReplaceUnderpriced)
	}

	if err := pool.AddRemote(pricedTransaction(0, 100000, big.NewInt(2), key)); err != nil {
		t.Fatalf("failed to replace original cheap pending transaction: %v", err)
	}

	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("cheap replacement event firing failed: %v", err)
	}

	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(price), key)); err != nil {
		t.Fatalf("failed to add original proper pending transaction: %v", err)
	}

	if err := pool.AddRemote(pricedTransaction(0, 100001, big.NewInt(threshold-1), key)); !errors.Is(err, ErrReplaceUnderpriced) {
		t.Fatalf("original proper pending transaction replacement error mismatch: have %v, want %v", err, ErrReplaceUnderpriced)
	}

	if err := pool.AddRemote(pricedTransaction(0, 100000, big.NewInt(threshold), key)); err != nil {
		t.Fatalf("failed to replace original proper pending transaction: %v", err)
	}

	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("proper replacement event firing failed: %v", err)
	}

	// Add queued transactions, ensuring the minimum price bump is enforced for replacement (for ultra low prices too)
	if err := pool.AddRemote(pricedTransaction(2, 100000, big.NewInt(1), key)); err != nil {
		t.Fatalf("failed to add original cheap queued transaction: %v", err)
	}

	if err := pool.AddRemote(pricedTransaction(2, 100001, big.NewInt(1), key)); !errors.Is(err, ErrReplaceUnderpriced) {
		t.Fatalf("original cheap queued transaction replacement error mismatch: have %v, want %v", err, ErrReplaceUnderpriced)
	}

	if err := pool.AddRemote(pricedTransaction(2, 100000, big.NewInt(2), key)); err != nil {
		t.Fatalf("failed to replace original cheap queued transaction: %v", err)
	}

	if err := pool.AddRemote(pricedTransaction(2, 100000, big.NewInt(price), key)); err != nil {
		t.Fatalf("failed to add original proper queued transaction: %v", err)
	}

	if err := pool.AddRemote(pricedTransaction(2, 100001, big.NewInt(threshold-1), key)); !errors.Is(err, ErrReplaceUnderpriced) {
		t.Fatalf("original proper queued transaction replacement error mismatch: have %v, want %v", err, ErrReplaceUnderpriced)
	}

	if err := pool.AddRemote(pricedTransaction(2, 100000, big.NewInt(threshold), key)); err != nil {
		t.Fatalf("failed to replace original proper queued transaction: %v", err)
	}

	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("queued replacement event firing failed: %v", err)
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that the pool rejects replacement dynamic fee transactions that don't
// meet the minimum price bump required.
func TestTransactionReplacementDynamicFee(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	pool, key := setupTxPoolWithConfig(eip1559Config, testTxPoolConfig, txPoolGasLimit)
	defer pool.Stop()
	testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000000))

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan NewTxsEvent, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Add pending transactions, ensuring the minimum price bump is enforced for replacement (for ultra low prices too)
	gasFeeCap := int64(100)
	feeCapThreshold := (gasFeeCap * (100 + int64(testTxPoolConfig.PriceBump))) / 100
	gasTipCap := int64(60)
	tipThreshold := (gasTipCap * (100 + int64(testTxPoolConfig.PriceBump))) / 100

	// Run the following identical checks for both the pending and queue pools:
	//	1.  Send initial tx => accept
	//	2.  Don't bump tip or fee cap => discard
	//	3.  Bump both more than min => accept
	//	4.  Check events match expected (2 new executable txs during pending, 0 during queue)
	//	5.  Send new tx with larger tip and gasFeeCap => accept
	//	6.  Bump tip max allowed so it's still underpriced => discard
	//	7.  Bump fee cap max allowed so it's still underpriced => discard
	//	8.  Bump tip min for acceptance => discard
	//	9.  Bump feecap min for acceptance => discard
	//	10. Bump feecap and tip min for acceptance => accept
	//	11. Check events match expected (2 new executable txs during pending, 0 during queue)
	stages := []string{"pending", "queued"}
	for _, stage := range stages {
		// Since state is empty, 0 nonce txs are "executable" and can go
		// into pending immediately. 2 nonce txs are "happed
		nonce := uint64(0)
		if stage == "queued" {
			nonce = 2
		}

		// 1.  Send initial tx => accept
		tx := dynamicFeeTx(nonce, 100000, big.NewInt(2), big.NewInt(1), key)
		if err := pool.addRemoteSync(tx); err != nil {
			t.Fatalf("failed to add original cheap %s transaction: %v", stage, err)
		}
		// 2.  Don't bump tip or feecap => discard
		tx = dynamicFeeTx(nonce, 100001, big.NewInt(2), big.NewInt(1), key)
		if err := pool.AddRemote(tx); !errors.Is(err, ErrReplaceUnderpriced) {
			t.Fatalf("original cheap %s transaction replacement error mismatch: have %v, want %v", stage, err, ErrReplaceUnderpriced)
		}
		// 3.  Bump both more than min => accept
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(3), big.NewInt(2), key)
		if err := pool.AddRemote(tx); err != nil {
			t.Fatalf("failed to replace original cheap %s transaction: %v", stage, err)
		}
		// 4.  Check events match expected (2 new executable txs during pending, 0 during queue)
		count := 2
		if stage == "queued" {
			count = 0
		}
		if err := validateEvents(events, count); err != nil {
			t.Fatalf("cheap %s replacement event firing failed: %v", stage, err)
		}
		// 5.  Send new tx with larger tip and feeCap => accept
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(gasFeeCap), big.NewInt(gasTipCap), key)
		if err := pool.addRemoteSync(tx); err != nil {
			t.Fatalf("failed to add original proper %s transaction: %v", stage, err)
		}
		// 6.  Bump tip max allowed so it's still underpriced => discard
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(gasFeeCap), big.NewInt(tipThreshold-1), key)
		if err := pool.AddRemote(tx); !errors.Is(err, ErrReplaceUnderpriced) {
			t.Fatalf("original proper %s transaction replacement error mismatch: have %v, want %v", stage, err, ErrReplaceUnderpriced)
		}
		// 7.  Bump fee cap max allowed so it's still underpriced => discard
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(feeCapThreshold-1), big.NewInt(gasTipCap), key)
		if err := pool.AddRemote(tx); !errors.Is(err, ErrReplaceUnderpriced) {
			t.Fatalf("original proper %s transaction replacement error mismatch: have %v, want %v", stage, err, ErrReplaceUnderpriced)
		}
		// 8.  Bump tip min for acceptance => accept
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(gasFeeCap), big.NewInt(tipThreshold), key)
		if err := pool.AddRemote(tx); !errors.Is(err, ErrReplaceUnderpriced) {
			t.Fatalf("original proper %s transaction replacement error mismatch: have %v, want %v", stage, err, ErrReplaceUnderpriced)
		}
		// 9.  Bump fee cap min for acceptance => accept
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(feeCapThreshold), big.NewInt(gasTipCap), key)
		if err := pool.AddRemote(tx); !errors.Is(err, ErrReplaceUnderpriced) {
			t.Fatalf("original proper %s transaction replacement error mismatch: have %v, want %v", stage, err, ErrReplaceUnderpriced)
		}
		// 10. Check events match expected (3 new executable txs during pending, 0 during queue)
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(feeCapThreshold), big.NewInt(tipThreshold), key)
		if err := pool.AddRemote(tx); err != nil {
			t.Fatalf("failed to replace original cheap %s transaction: %v", stage, err)
		}
		// 11. Check events match expected (3 new executable txs during pending, 0 during queue)
		count = 2
		if stage == "queued" {
			count = 0
		}
		if err := validateEvents(events, count); err != nil {
			t.Fatalf("replacement %s event firing failed: %v", stage, err)
		}
	}

	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that local transactions are journaled to disk, but remote transactions
// get discarded between restarts.
func TestTransactionJournaling(t *testing.T)         { testTransactionJournaling(t, false) }
func TestTransactionJournalingNoLocals(t *testing.T) { testTransactionJournaling(t, true) }

func testTransactionJournaling(t *testing.T, nolocals bool) {
	t.Parallel()

	// Create a temporary file for the journal
	file, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("failed to create temporary journal: %v", err)
	}
	journal := file.Name()
	defer os.Remove(journal)

	// Clean up the temporary file, we only need the path for now
	file.Close()
	os.Remove(journal)

	// Create the original pool to inject transaction into the journal
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{1000000, statedb, new(event.Feed)}

	config := testTxPoolConfig
	config.NoLocals = nolocals
	config.Journal = journal
	config.Rejournal = time.Second

	pool := NewTxPool(config, params.TestChainConfig, blockchain)

	// Create two test accounts to ensure remotes expire but locals do not
	local, _ := crypto.GenerateKey()
	remote, _ := crypto.GenerateKey()

	testAddBalance(pool, crypto.PubkeyToAddress(local.PublicKey), big.NewInt(1000000000))
	testAddBalance(pool, crypto.PubkeyToAddress(remote.PublicKey), big.NewInt(1000000000))

	// Add three local and a remote transactions and ensure they are queued up
	if err := pool.AddLocal(pricedTransaction(0, 100000, big.NewInt(1), local)); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}
	if err := pool.AddLocal(pricedTransaction(1, 100000, big.NewInt(1), local)); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}
	if err := pool.AddLocal(pricedTransaction(2, 100000, big.NewInt(1), local)); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(1), remote)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	pending, queued := pool.Stats()
	if pending != 4 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 4)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Terminate the old pool, bump the local nonce, create a new pool and ensure relevant transaction survive
	pool.Stop()
	statedb.SetNonce(crypto.PubkeyToAddress(local.PublicKey), 1)
	blockchain = &testBlockChain{1000000, statedb, new(event.Feed)}

	pool = NewTxPool(config, params.TestChainConfig, blockchain)

	pending, queued = pool.Stats()
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if nolocals {
		if pending != 0 {
			t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
		}
	} else {
		if pending != 2 {
			t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
		}
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Bump the nonce temporarily and ensure the newly invalidated transaction is removed
	statedb.SetNonce(crypto.PubkeyToAddress(local.PublicKey), 2)
	<-pool.requestReset(nil, nil)
	time.Sleep(2 * config.Rejournal)
	pool.Stop()

	statedb.SetNonce(crypto.PubkeyToAddress(local.PublicKey), 1)
	blockchain = &testBlockChain{1000000, statedb, new(event.Feed)}
	pool = NewTxPool(config, params.TestChainConfig, blockchain)

	pending, queued = pool.Stats()
	if pending != 0 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
	}
	if nolocals {
		if queued != 0 {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
		}
	} else {
		if queued != 1 {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
		}
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	pool.Stop()
}

// TestTransactionStatusCheck tests that the pool can correctly retrieve the
// pending status of individual transactions.
func TestTransactionStatusCheck(t *testing.T) {
	t.Parallel()

	// Create the pool to test the status retrievals with
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{1000000, statedb, new(event.Feed)}

	pool := NewTxPool(testTxPoolConfig, params.TestChainConfig, blockchain)
	defer pool.Stop()

	// Create the test accounts to check various transaction statuses with
	keys := make([]*ecdsa.PrivateKey, 3)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}
	// Generate and queue a batch of transactions, both pending and queued
	txs := types.Transactions{}

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(1), keys[0])) // Pending only
	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(1), keys[1])) // Pending and queued
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(1), keys[1]))
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(1), keys[2])) // Queued only

	// Import the transaction and ensure they are correctly added
	pool.AddRemotesSync(txs)

	pending, queued := pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}
	if err := validateTxPoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Retrieve the status of each transaction and validate them
	hashes := make([]common.Hash, len(txs))
	for i, tx := range txs {
		hashes[i] = tx.Hash()
	}
	hashes = append(hashes, common.Hash{})

	statuses := pool.Status(hashes)
	expect := []TxStatus{TxStatusPending, TxStatusPending, TxStatusQueued, TxStatusQueued, TxStatusUnknown}

	for i := 0; i < len(statuses); i++ {
		if statuses[i] != expect[i] {
			t.Errorf("transaction %d: status mismatch: have %v, want %v", i, statuses[i], expect[i])
		}
	}
}

// Test the transaction slots consumption is computed correctly
func TestTransactionSlotCount(t *testing.T) {
	t.Parallel()

	key, _ := crypto.GenerateKey()

	// Check that an empty transaction consumes a single slot
	smallTx := pricedDataTransaction(0, 0, big.NewInt(0), key, 0)
	if slots := numSlots(smallTx); slots != 1 {
		t.Fatalf("small transactions slot count mismatch: have %d want %d", slots, 1)
	}
	// Check that a large transaction consumes the correct number of slots
	bigTx := pricedDataTransaction(0, 0, big.NewInt(0), key, uint64(10*txSlotSize))
	if slots := numSlots(bigTx); slots != 11 {
		t.Fatalf("big transactions slot count mismatch: have %d want %d", slots, 11)
	}
}

// Benchmarks the speed of validating the contents of the pending queue of the
// transaction pool.
func BenchmarkPendingDemotion100(b *testing.B)   { benchmarkPendingDemotion(b, 100) }
func BenchmarkPendingDemotion1000(b *testing.B)  { benchmarkPendingDemotion(b, 1000) }
func BenchmarkPendingDemotion10000(b *testing.B) { benchmarkPendingDemotion(b, 10000) }

func benchmarkPendingDemotion(b *testing.B, size int) {
	// Add a batch of transactions to a pool one by one
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000))

	for i := 0; i < size; i++ {
		tx := transaction(uint64(i), 100000, key)
		pool.promoteTx(account, tx.Hash(), tx)
	}
	// Benchmark the speed of pool validation
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pool.demoteUnexecutables()
	}
}

// Benchmarks the speed of scheduling the contents of the future queue of the
// transaction pool.
func BenchmarkFuturePromotion100(b *testing.B)   { benchmarkFuturePromotion(b, 100) }
func BenchmarkFuturePromotion1000(b *testing.B)  { benchmarkFuturePromotion(b, 1000) }
func BenchmarkFuturePromotion10000(b *testing.B) { benchmarkFuturePromotion(b, 10000) }

func benchmarkFuturePromotion(b *testing.B, size int) {
	// Add a batch of transactions to a pool one by one
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000))

	for i := 0; i < size; i++ {
		tx := transaction(uint64(1+i), 100000, key)
		pool.enqueueTx(tx.Hash(), tx, false, true)
	}
	// Benchmark the speed of pool validation
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.promoteExecutables(nil)
	}
}

// Benchmarks the speed of batched transaction insertion.
func BenchmarkPoolBatchInsert(b *testing.B) {
	// Generate a batch of transactions to enqueue into the pool
	pool, key := setupTxPool()
	defer pool.Stop()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000))

	const format = "size %d, is local %t"

	cases := []struct {
		name    string
		size    int
		isLocal bool
	}{
		{size: 100, isLocal: false},
		{size: 1000, isLocal: false},
		{size: 10000, isLocal: false},

		{size: 100, isLocal: true},
		{size: 1000, isLocal: true},
		{size: 10000, isLocal: true},
	}

	for i := range cases {
		cases[i].name = fmt.Sprintf(format, cases[i].size, cases[i].isLocal)
	}

	// Benchmark importing the transactions into the queue

	for _, testCase := range cases {
		singleCase := testCase

		b.Run(singleCase.name, func(b *testing.B) {
			batches := make([]types.Transactions, b.N)

			for i := 0; i < b.N; i++ {
				batches[i] = make(types.Transactions, singleCase.size)

				for j := 0; j < singleCase.size; j++ {
					batches[i][j] = transaction(uint64(singleCase.size*i+j), 100000, key)
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for _, batch := range batches {
				if testCase.isLocal {
					pool.AddLocals(batch)
				} else {
					pool.AddRemotes(batch)
				}
			}
		})
	}
}

func BenchmarkPoolMining(b *testing.B) {
	const format = "size %d"

	cases := []struct {
		name string
		size int
	}{
		{size: 1},
		{size: 5},
		{size: 10},
		{size: 20},
	}

	for i := range cases {
		cases[i].name = fmt.Sprintf(format, cases[i].size)
	}

	const blockGasLimit = 30_000_000

	// Benchmark importing the transactions into the queue

	for _, testCase := range cases {
		singleCase := testCase

		b.Run(singleCase.name, func(b *testing.B) {
			// Generate a batch of transactions to enqueue into the pool
			pendingAddedCh := make(chan struct{}, 1024)

			pool, localKey := setupTxPoolWithConfig(params.TestChainConfig, testTxPoolConfig, txPoolGasLimit, MakeWithPromoteTxCh(pendingAddedCh))
			defer pool.Stop()

			localKeyPub := localKey.PublicKey
			account := crypto.PubkeyToAddress(localKeyPub)

			const balanceStr = "1_000_000_000"
			balance, ok := big.NewInt(0).SetString(balanceStr, 0)
			if !ok {
				b.Fatal("incorrect initial balance", balanceStr)
			}

			testAddBalance(pool, account, balance)

			signer := types.NewEIP155Signer(big.NewInt(1))
			baseFee := uint256.NewInt(1)

			const batchesSize = 100

			batches := make([]types.Transactions, batchesSize)

			for i := 0; i < batchesSize; i++ {
				batches[i] = make(types.Transactions, singleCase.size)

				for j := 0; j < singleCase.size; j++ {
					batches[i][j] = transaction(uint64(singleCase.size*i+j), 100_000, localKey)
				}

				for _, batch := range batches {
					pool.AddRemotes(batch)
				}
			}

			var promoted int

			for range pendingAddedCh {
				promoted++

				if promoted >= batchesSize*singleCase.size/2 {
					break
				}
			}

			var total int

			b.ResetTimer()
			b.ReportAllocs()

			pendingDurations := make([]time.Duration, b.N)

			var added int

			for i := 0; i < b.N; i++ {
				added, pendingDurations[i], _ = mining(b, pool, signer, baseFee, blockGasLimit, i)
				total += added
			}

			b.StopTimer()

			pendingDurationsFloat := make([]float64, len(pendingDurations))

			for i, v := range pendingDurations {
				pendingDurationsFloat[i] = float64(v.Nanoseconds())
			}

			mean, stddev := stat.MeanStdDev(pendingDurationsFloat, nil)
			b.Logf("[%s] pending mean %v, stdev %v, %v-%v",
				common.NowMilliseconds(), time.Duration(mean), time.Duration(stddev), time.Duration(floats.Min(pendingDurationsFloat)), time.Duration(floats.Max(pendingDurationsFloat)))
		})
	}
}

func BenchmarkInsertRemoteWithAllLocals(b *testing.B) {
	// Allocate keys for testing
	key, _ := crypto.GenerateKey()
	account := crypto.PubkeyToAddress(key.PublicKey)

	remoteKey, _ := crypto.GenerateKey()
	remoteAddr := crypto.PubkeyToAddress(remoteKey.PublicKey)

	locals := make([]*types.Transaction, 4096+1024) // Occupy all slots
	for i := 0; i < len(locals); i++ {
		locals[i] = transaction(uint64(i), 100000, key)
	}
	remotes := make([]*types.Transaction, 1000)
	for i := 0; i < len(remotes); i++ {
		remotes[i] = pricedTransaction(uint64(i), 100000, big.NewInt(2), remoteKey) // Higher gasprice
	}
	// Benchmark importing the transactions into the queue
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pool, _ := setupTxPool()
		testAddBalance(pool, account, big.NewInt(100000000))
		for _, local := range locals {
			pool.AddLocal(local)
		}
		b.StartTimer()
		// Assign a high enough balance for testing
		testAddBalance(pool, remoteAddr, big.NewInt(100000000))
		for i := 0; i < len(remotes); i++ {
			pool.AddRemotes([]*types.Transaction{remotes[i]})
		}
		pool.Stop()
	}
}

// Benchmarks the speed of batch transaction insertion in case of multiple accounts.
func BenchmarkPoolAccountMultiBatchInsert(b *testing.B) {
	// Generate a batch of transactions to enqueue into the pool
	pool, _ := setupTxPool()
	defer pool.Stop()

	batches := make(types.Transactions, b.N)

	for i := 0; i < b.N; i++ {
		key, _ := crypto.GenerateKey()
		account := crypto.PubkeyToAddress(key.PublicKey)

		pool.currentState.AddBalance(account, big.NewInt(1000000))

		tx := transaction(uint64(0), 100000, key)

		batches[i] = tx
	}

	// Benchmark importing the transactions into the queue
	b.ReportAllocs()
	b.ResetTimer()

	for _, tx := range batches {
		pool.AddRemotesSync([]*types.Transaction{tx})
	}
}

func BenchmarkPoolAccountMultiBatchInsertRace(b *testing.B) {
	// Generate a batch of transactions to enqueue into the pool
	pool, _ := setupTxPool()
	defer pool.Stop()

	batches := make(types.Transactions, b.N)

	for i := 0; i < b.N; i++ {
		key, _ := crypto.GenerateKey()
		account := crypto.PubkeyToAddress(key.PublicKey)
		tx := transaction(uint64(0), 100000, key)

		pool.currentState.AddBalance(account, big.NewInt(1000000))

		batches[i] = tx
	}

	done := make(chan struct{})

	go func() {
		t := time.NewTicker(time.Microsecond)
		defer t.Stop()

		var pending map[common.Address]types.Transactions

	loop:
		for {
			select {
			case <-t.C:
				pending = pool.Pending(context.Background(), true)
			case <-done:
				break loop
			}
		}

		fmt.Fprint(io.Discard, pending)
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for _, tx := range batches {
		pool.AddRemotesSync([]*types.Transaction{tx})
	}

	close(done)
}

func BenchmarkPoolAccountMultiBatchInsertNoLockRace(b *testing.B) {
	// Generate a batch of transactions to enqueue into the pool
	pendingAddedCh := make(chan struct{}, 1024)

	pool, localKey := setupTxPoolWithConfig(params.TestChainConfig, testTxPoolConfig, txPoolGasLimit, MakeWithPromoteTxCh(pendingAddedCh))
	defer pool.Stop()

	_ = localKey

	batches := make(types.Transactions, b.N)

	for i := 0; i < b.N; i++ {
		key, _ := crypto.GenerateKey()
		account := crypto.PubkeyToAddress(key.PublicKey)
		tx := transaction(uint64(0), 100000, key)

		pool.currentState.AddBalance(account, big.NewInt(1000000))

		batches[i] = tx
	}

	done := make(chan struct{})

	go func() {
		t := time.NewTicker(time.Microsecond)
		defer t.Stop()

		var pending map[common.Address]types.Transactions

		for range t.C {
			pending = pool.Pending(context.Background(), true)

			if len(pending) >= b.N/2 {
				close(done)

				return
			}
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for _, tx := range batches {
		pool.AddRemotes([]*types.Transaction{tx})
	}

	<-done
}

func BenchmarkPoolAccountsBatchInsert(b *testing.B) {
	// Generate a batch of transactions to enqueue into the pool
	pool, _ := setupTxPool()
	defer pool.Stop()

	batches := make(types.Transactions, b.N)

	for i := 0; i < b.N; i++ {
		key, _ := crypto.GenerateKey()
		account := crypto.PubkeyToAddress(key.PublicKey)

		pool.currentState.AddBalance(account, big.NewInt(1000000))

		tx := transaction(uint64(0), 100000, key)

		batches[i] = tx
	}

	// Benchmark importing the transactions into the queue
	b.ReportAllocs()
	b.ResetTimer()

	for _, tx := range batches {
		_ = pool.AddRemoteSync(tx)
	}
}

func BenchmarkPoolAccountsBatchInsertRace(b *testing.B) {
	// Generate a batch of transactions to enqueue into the pool
	pool, _ := setupTxPool()
	defer pool.Stop()

	batches := make(types.Transactions, b.N)

	for i := 0; i < b.N; i++ {
		key, _ := crypto.GenerateKey()
		account := crypto.PubkeyToAddress(key.PublicKey)
		tx := transaction(uint64(0), 100000, key)

		pool.currentState.AddBalance(account, big.NewInt(1000000))

		batches[i] = tx
	}

	done := make(chan struct{})

	go func() {
		t := time.NewTicker(time.Microsecond)
		defer t.Stop()

		var pending map[common.Address]types.Transactions

	loop:
		for {
			select {
			case <-t.C:
				pending = pool.Pending(context.Background(), true)
			case <-done:
				break loop
			}
		}

		fmt.Fprint(io.Discard, pending)
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for _, tx := range batches {
		_ = pool.AddRemoteSync(tx)
	}

	close(done)
}

func BenchmarkPoolAccountsBatchInsertNoLockRace(b *testing.B) {
	// Generate a batch of transactions to enqueue into the pool
	pendingAddedCh := make(chan struct{}, 1024)

	pool, localKey := setupTxPoolWithConfig(params.TestChainConfig, testTxPoolConfig, txPoolGasLimit, MakeWithPromoteTxCh(pendingAddedCh))
	defer pool.Stop()

	_ = localKey

	batches := make(types.Transactions, b.N)

	for i := 0; i < b.N; i++ {
		key, _ := crypto.GenerateKey()
		account := crypto.PubkeyToAddress(key.PublicKey)
		tx := transaction(uint64(0), 100000, key)

		pool.currentState.AddBalance(account, big.NewInt(1000000))

		batches[i] = tx
	}

	done := make(chan struct{})

	go func() {
		t := time.NewTicker(time.Microsecond)
		defer t.Stop()

		var pending map[common.Address]types.Transactions

		for range t.C {
			pending = pool.Pending(context.Background(), true)

			if len(pending) >= b.N/2 {
				close(done)

				return
			}
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for _, tx := range batches {
		_ = pool.AddRemote(tx)
	}

	<-done
}

func TestPoolMultiAccountBatchInsertRace(t *testing.T) {
	t.Parallel()

	// Generate a batch of transactions to enqueue into the pool
	pool, _ := setupTxPool()
	defer pool.Stop()

	const n = 5000

	batches := make(types.Transactions, n)
	batchesSecond := make(types.Transactions, n)

	for i := 0; i < n; i++ {
		batches[i] = newTxs(pool)
		batchesSecond[i] = newTxs(pool)
	}

	done := make(chan struct{})

	go func() {
		t := time.NewTicker(time.Microsecond)
		defer t.Stop()

		var (
			pending map[common.Address]types.Transactions
			total   int
		)

		for range t.C {
			pending = pool.Pending(context.Background(), true)
			total = len(pending)

			_ = pool.Locals()

			if total >= n {
				close(done)

				return
			}
		}
	}()

	for _, tx := range batches {
		pool.AddRemotesSync([]*types.Transaction{tx})
	}

	for _, tx := range batchesSecond {
		pool.AddRemotes([]*types.Transaction{tx})
	}

	<-done
}

func newTxs(pool *TxPool) *types.Transaction {
	key, _ := crypto.GenerateKey()
	account := crypto.PubkeyToAddress(key.PublicKey)
	tx := transaction(uint64(0), 100000, key)

	pool.currentState.AddBalance(account, big.NewInt(1_000_000_000))

	return tx
}

type acc struct {
	nonce   uint64
	key     *ecdsa.PrivateKey
	account common.Address
}

type testTx struct {
	tx      *types.Transaction
	idx     int
	isLocal bool
}

const localIdx = 0

func getTransactionGen(t *rapid.T, keys []*acc, nonces []uint64, localKey *acc, gasPriceMin, gasPriceMax, gasLimitMin, gasLimitMax uint64) *testTx {
	idx := rapid.IntRange(0, len(keys)-1).Draw(t, "accIdx").(int)

	var (
		isLocal bool
		key     *ecdsa.PrivateKey
	)

	if idx == localIdx {
		isLocal = true
		key = localKey.key
	} else {
		key = keys[idx].key
	}

	nonces[idx]++

	gasPriceUint := rapid.Uint64Range(gasPriceMin, gasPriceMax).Draw(t, "gasPrice").(uint64)
	gasPrice := big.NewInt(0).SetUint64(gasPriceUint)
	gasLimit := rapid.Uint64Range(gasLimitMin, gasLimitMax).Draw(t, "gasLimit").(uint64)

	return &testTx{
		tx:      pricedTransaction(nonces[idx]-1, gasLimit, gasPrice, key),
		idx:     idx,
		isLocal: isLocal,
	}
}

type transactionBatches struct {
	txs      []*testTx
	totalTxs int
}

func transactionsGen(keys []*acc, nonces []uint64, localKey *acc, minTxs int, maxTxs int, gasPriceMin, gasPriceMax, gasLimitMin, gasLimitMax uint64, caseParams *strings.Builder) func(t *rapid.T) *transactionBatches {
	return func(t *rapid.T) *transactionBatches {
		totalTxs := rapid.IntRange(minTxs, maxTxs).Draw(t, "totalTxs").(int)
		txs := make([]*testTx, totalTxs)

		gasValues := make([]float64, totalTxs)

		fmt.Fprintf(caseParams, " totalTxs = %d;", totalTxs)

		keys = keys[:len(nonces)]

		for i := 0; i < totalTxs; i++ {
			txs[i] = getTransactionGen(t, keys, nonces, localKey, gasPriceMin, gasPriceMax, gasLimitMin, gasLimitMax)

			gasValues[i] = float64(txs[i].tx.Gas())
		}

		mean, stddev := stat.MeanStdDev(gasValues, nil)
		fmt.Fprintf(caseParams, " gasValues mean %d, stdev %d, %d-%d);", int64(mean), int64(stddev), int64(floats.Min(gasValues)), int64(floats.Max(gasValues)))

		return &transactionBatches{txs, totalTxs}
	}
}

type txPoolRapidConfig struct {
	gasLimit    uint64
	avgBlockTxs uint64

	minTxs int
	maxTxs int

	minAccs int
	maxAccs int

	// less tweakable, more like constants
	gasPriceMin uint64
	gasPriceMax uint64

	gasLimitMin uint64
	gasLimitMax uint64

	balance int64

	blockTime      time.Duration
	maxEmptyBlocks int
	maxStuckBlocks int
}

func defaultTxPoolRapidConfig() txPoolRapidConfig {
	gasLimit := uint64(30_000_000)
	avgBlockTxs := gasLimit/params.TxGas + 1
	maxTxs := int(25 * avgBlockTxs)

	return txPoolRapidConfig{
		gasLimit: gasLimit,

		avgBlockTxs: avgBlockTxs,

		minTxs: 1,
		maxTxs: maxTxs,

		minAccs: 1,
		maxAccs: maxTxs,

		// less tweakable, more like constants
		gasPriceMin: 1,
		gasPriceMax: 1_000,

		gasLimitMin: params.TxGas,
		gasLimitMax: gasLimit / 2,

		balance: 0xffffffffffffff,

		blockTime:      2 * time.Second,
		maxEmptyBlocks: 10,
		maxStuckBlocks: 10,
	}
}

// TestSmallTxPool is not something to run in parallel as far it uses all CPUs
// nolint:paralleltest
func TestSmallTxPool(t *testing.T) {
	t.Parallel()

	t.Skip("a red test to be fixed")

	cfg := defaultTxPoolRapidConfig()

	cfg.maxEmptyBlocks = 10
	cfg.maxStuckBlocks = 10

	cfg.minTxs = 1
	cfg.maxTxs = 2

	cfg.minAccs = 1
	cfg.maxAccs = 2

	testPoolBatchInsert(t, cfg)
}

// This test is not something to run in parallel as far it uses all CPUs
// nolint:paralleltest
func TestBigTxPool(t *testing.T) {
	t.Parallel()

	t.Skip("a red test to be fixed")

	cfg := defaultTxPoolRapidConfig()

	testPoolBatchInsert(t, cfg)
}

//nolint:gocognit,thelper
func testPoolBatchInsert(t *testing.T, cfg txPoolRapidConfig) {
	t.Helper()

	t.Parallel()

	const debug = false

	initialBalance := big.NewInt(cfg.balance)

	keys := make([]*acc, cfg.maxAccs)

	var key *ecdsa.PrivateKey

	// prealloc keys
	for idx := 0; idx < cfg.maxAccs; idx++ {
		key, _ = crypto.GenerateKey()

		keys[idx] = &acc{
			key:     key,
			nonce:   0,
			account: crypto.PubkeyToAddress(key.PublicKey),
		}
	}

	var threads = runtime.NumCPU()

	if debug {
		// 1 is set only for debug
		threads = 1
	}

	testsDone := new(uint64)

	for i := 0; i < threads; i++ {
		t.Run(fmt.Sprintf("thread %d", i), func(t *testing.T) {
			t.Parallel()

			rapid.Check(t, func(rt *rapid.T) {
				caseParams := new(strings.Builder)

				defer func() {
					res := atomic.AddUint64(testsDone, 1)

					if res%100 == 0 {
						fmt.Println("case-done", res)
					}
				}()

				// Generate a batch of transactions to enqueue into the pool
				testTxPoolConfig := testTxPoolConfig

				// from sentry config
				testTxPoolConfig.AccountQueue = 16
				testTxPoolConfig.AccountSlots = 16
				testTxPoolConfig.GlobalQueue = 32768
				testTxPoolConfig.GlobalSlots = 32768
				testTxPoolConfig.Lifetime = time.Hour + 30*time.Minute //"1h30m0s"
				testTxPoolConfig.PriceLimit = 1

				now := time.Now()
				pendingAddedCh := make(chan struct{}, 1024)
				pool, key := setupTxPoolWithConfig(params.TestChainConfig, testTxPoolConfig, cfg.gasLimit, MakeWithPromoteTxCh(pendingAddedCh))
				defer pool.Stop()

				totalAccs := rapid.IntRange(cfg.minAccs, cfg.maxAccs).Draw(rt, "totalAccs").(int)

				fmt.Fprintf(caseParams, "Case params: totalAccs = %d;", totalAccs)

				defer func() {
					pending, queued := pool.Content()

					if len(pending) != 0 {
						pendingGas := make([]float64, 0, len(pending))

						for _, txs := range pending {
							for _, tx := range txs {
								pendingGas = append(pendingGas, float64(tx.Gas()))
							}
						}

						mean, stddev := stat.MeanStdDev(pendingGas, nil)
						fmt.Fprintf(caseParams, "\tpending mean %d, stdev %d, %d-%d;\n", int64(mean), int64(stddev), int64(floats.Min(pendingGas)), int64(floats.Max(pendingGas)))
					}

					if len(queued) != 0 {
						queuedGas := make([]float64, 0, len(queued))

						for _, txs := range queued {
							for _, tx := range txs {
								queuedGas = append(queuedGas, float64(tx.Gas()))
							}
						}

						mean, stddev := stat.MeanStdDev(queuedGas, nil)
						fmt.Fprintf(caseParams, "\tqueued mean %d, stdev %d, %d-%d);\n\n", int64(mean), int64(stddev), int64(floats.Min(queuedGas)), int64(floats.Max(queuedGas)))
					}

					rt.Log(caseParams)
				}()

				// regenerate only local key
				localKey := &acc{
					key:     key,
					account: crypto.PubkeyToAddress(key.PublicKey),
				}

				if err := validateTxPoolInternals(pool); err != nil {
					rt.Fatalf("pool internal state corrupted: %v", err)
				}

				var wg sync.WaitGroup
				wg.Add(1)

				go func() {
					defer wg.Done()
					now = time.Now()

					testAddBalance(pool, localKey.account, initialBalance)

					for idx := 0; idx < totalAccs; idx++ {
						testAddBalance(pool, keys[idx].account, initialBalance)
					}
				}()

				nonces := make([]uint64, totalAccs)
				gen := rapid.Custom(transactionsGen(keys, nonces, localKey, cfg.minTxs, cfg.maxTxs, cfg.gasPriceMin, cfg.gasPriceMax, cfg.gasLimitMin, cfg.gasLimitMax, caseParams))

				txs := gen.Draw(rt, "batches").(*transactionBatches)

				wg.Wait()

				var (
					addIntoTxPool func(tx *types.Transaction) error
					totalInBatch  int
				)

				for _, tx := range txs.txs {
					addIntoTxPool = pool.AddRemoteSync

					if tx.isLocal {
						addIntoTxPool = pool.AddLocal
					}

					err := addIntoTxPool(tx.tx)
					if err != nil {
						rt.Log("on adding a transaction to the tx pool", err, tx.tx.Gas(), tx.tx.GasPrice(), pool.GasPrice(), getBalance(pool, keys[tx.idx].account))
					}
				}

				var (
					block              int
					emptyBlocks        int
					stuckBlocks        int
					lastTxPoolStats    int
					currentTxPoolStats int
				)

				for {
					// we'd expect fulfilling block take comparable, but less than blockTime
					ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.maxStuckBlocks)*cfg.blockTime)

					select {
					case <-pendingAddedCh:
					case <-ctx.Done():
						pendingStat, queuedStat := pool.Stats()
						if pendingStat+queuedStat == 0 {
							cancel()

							break
						}

						rt.Fatalf("got %ds block timeout (expected less then %s): total accounts %d. Pending %d, queued %d)",
							block, 5*cfg.blockTime, txs.totalTxs, pendingStat, queuedStat)
					}

					pendingStat, queuedStat := pool.Stats()
					currentTxPoolStats = pendingStat + queuedStat
					if currentTxPoolStats == 0 {
						cancel()
						break
					}

					// check if txPool got stuck
					if currentTxPoolStats == lastTxPoolStats {
						stuckBlocks++ //todo: need something better then that
					} else {
						stuckBlocks = 0
						lastTxPoolStats = currentTxPoolStats
					}

					// copy-paste
					start := time.Now()
					pending := pool.Pending(context.Background(), true)
					locals := pool.Locals()

					// from fillTransactions
					removedFromPool, blockGasLeft, err := fillTransactions(ctx, pool, locals, pending, cfg.gasLimit)

					done := time.Since(start)

					if removedFromPool > 0 {
						emptyBlocks = 0
					} else {
						emptyBlocks++
					}

					if emptyBlocks >= cfg.maxEmptyBlocks || stuckBlocks >= cfg.maxStuckBlocks {
						// check for nonce gaps
						var lastNonce, currentNonce int

						pending = pool.Pending(context.Background(), true)

						for txAcc, pendingTxs := range pending {
							lastNonce = int(pool.Nonce(txAcc)) - len(pendingTxs) - 1

							isFirst := true

							for _, tx := range pendingTxs {
								currentNonce = int(tx.Nonce())
								if currentNonce-lastNonce != 1 {
									rt.Fatalf("got a nonce gap for account %q. Current pending nonce %d, previous %d %v; emptyBlocks - %v; stuckBlocks - %v",
										txAcc, currentNonce, lastNonce, isFirst, emptyBlocks >= cfg.maxEmptyBlocks, stuckBlocks >= cfg.maxStuckBlocks)
								}

								lastNonce = currentNonce
							}
						}
					}

					if emptyBlocks >= cfg.maxEmptyBlocks {
						rt.Fatalf("got %d empty blocks in a row(expected less then %d): total time %s, total accounts %d. Pending %d, locals %d)",
							emptyBlocks, cfg.maxEmptyBlocks, done, txs.totalTxs, len(pending), len(locals))
					}

					if stuckBlocks >= cfg.maxStuckBlocks {
						rt.Fatalf("got %d empty blocks in a row(expected less then %d): total time %s, total accounts %d. Pending %d, locals %d)",
							emptyBlocks, cfg.maxEmptyBlocks, done, txs.totalTxs, len(pending), len(locals))
					}

					if err != nil {
						rt.Fatalf("took too long: total time %s(expected %s), total accounts %d. Pending %d, locals %d)",
							done, cfg.blockTime, txs.totalTxs, len(pending), len(locals))
					}

					rt.Log("current_total", txs.totalTxs, "in_batch", totalInBatch, "removed", removedFromPool, "emptyBlocks", emptyBlocks, "blockGasLeft", blockGasLeft, "pending", len(pending), "locals", len(locals),
						"locals+pending", done)

					rt.Log("block", block, "pending", pendingStat, "queued", queuedStat, "elapsed", done)

					block++

					cancel()

					//time.Sleep(time.Second)
				}

				rt.Logf("case completed totalTxs %d %v\n\n", txs.totalTxs, time.Since(now))
			})
		})
	}

	t.Log("done test cases", atomic.LoadUint64(testsDone))
}

func fillTransactions(ctx context.Context, pool *TxPool, locals []common.Address, pending map[common.Address]types.Transactions, gasLimit uint64) (int, uint64, error) {
	localTxs := make(map[common.Address]types.Transactions)
	remoteTxs := pending

	for _, txAcc := range locals {
		if txs := remoteTxs[txAcc]; len(txs) > 0 {
			delete(remoteTxs, txAcc)

			localTxs[txAcc] = txs
		}
	}

	// fake signer
	signer := types.NewLondonSigner(big.NewInt(1))

	// fake baseFee
	baseFee := uint256.NewInt(1)

	blockGasLimit := gasLimit

	var (
		txLocalCount  int
		txRemoteCount int
	)

	if len(localTxs) > 0 {
		txs := types.NewTransactionsByPriceAndNonce(signer, localTxs, baseFee)

		select {
		case <-ctx.Done():
			return txLocalCount + txRemoteCount, blockGasLimit, ctx.Err()
		default:
		}

		blockGasLimit, txLocalCount = commitTransactions(pool, txs, blockGasLimit)
	}

	select {
	case <-ctx.Done():
		return txLocalCount + txRemoteCount, blockGasLimit, ctx.Err()
	default:
	}

	if len(remoteTxs) > 0 {
		txs := types.NewTransactionsByPriceAndNonce(signer, remoteTxs, baseFee)

		select {
		case <-ctx.Done():
			return txLocalCount + txRemoteCount, blockGasLimit, ctx.Err()
		default:
		}

		blockGasLimit, txRemoteCount = commitTransactions(pool, txs, blockGasLimit)
	}

	return txLocalCount + txRemoteCount, blockGasLimit, nil
}

func commitTransactions(pool *TxPool, txs *types.TransactionsByPriceAndNonce, blockGasLimit uint64) (uint64, int) {
	var (
		tx      *types.Transaction
		txCount int
	)

	for {
		tx = txs.Peek()

		if tx == nil {
			return blockGasLimit, txCount
		}

		if tx.Gas() <= blockGasLimit {
			blockGasLimit -= tx.Gas()

			pool.mu.Lock()
			pool.removeTx(tx.Hash(), false)
			pool.mu.Unlock()

			txCount++
		} else {
			// we don't maximize fulfilment of the block. just fill somehow
			return blockGasLimit, txCount
		}
	}
}

func MakeWithPromoteTxCh(ch chan struct{}) func(*TxPool) {
	return func(pool *TxPool) {
		pool.promoteTxCh = ch
	}
}

func BenchmarkBigs(b *testing.B) {
	// max 256-bit
	max := new(big.Int)
	max.Exp(big.NewInt(2), big.NewInt(256), nil).Sub(max, big.NewInt(1))

	ints := make([]*big.Int, 1000000)
	intUs := make([]*uint256.Int, 1000000)

	var over bool

	for i := 0; i < len(ints); i++ {
		ints[i] = crand.BigInt(max)
		intUs[i], over = uint256.FromBig(ints[i])

		if over {
			b.Fatal(ints[i], over)
		}
	}

	b.Run("*big.Int", func(b *testing.B) {
		var r int

		for i := 0; i < b.N; i++ {
			r = ints[i%len(ints)%b.N].Cmp(ints[(i+1)%len(ints)%b.N])
		}

		fmt.Fprintln(io.Discard, r)
	})
	b.Run("*uint256.Int", func(b *testing.B) {
		var r int

		for i := 0; i < b.N; i++ {
			r = intUs[i%len(intUs)%b.N].Cmp(intUs[(i+1)%len(intUs)%b.N])
		}

		fmt.Fprintln(io.Discard, r)
	})
}

//nolint:thelper
func mining(tb testing.TB, pool *TxPool, signer types.Signer, baseFee *uint256.Int, blockGasLimit uint64, totalBlocks int) (int, time.Duration, time.Duration) {
	var (
		localTxsCount  int
		remoteTxsCount int
		localTxs       = make(map[common.Address]types.Transactions)
		remoteTxs      map[common.Address]types.Transactions
		total          int
	)

	start := time.Now()

	pending := pool.Pending(context.Background(), true)

	pendingDuration := time.Since(start)

	remoteTxs = pending

	locals := pool.Locals()

	pendingLen, queuedLen := pool.Stats()

	for _, account := range locals {
		if txs := remoteTxs[account]; len(txs) > 0 {
			delete(remoteTxs, account)

			localTxs[account] = txs
		}
	}

	localTxsCount = len(localTxs)
	remoteTxsCount = len(remoteTxs)

	var txLocalCount int

	if localTxsCount > 0 {
		txs := types.NewTransactionsByPriceAndNonce(signer, localTxs, baseFee)

		blockGasLimit, txLocalCount = commitTransactions(pool, txs, blockGasLimit)

		total += txLocalCount
	}

	var txRemoteCount int

	if remoteTxsCount > 0 {
		txs := types.NewTransactionsByPriceAndNonce(signer, remoteTxs, baseFee)

		_, txRemoteCount = commitTransactions(pool, txs, blockGasLimit)

		total += txRemoteCount
	}

	miningDuration := time.Since(start)

	tb.Logf("[%s] mining block. block %d. total %d: pending %d(added %d), local %d(added %d), queued %d, localTxsCount %d, remoteTxsCount %d, pending %v, mining %v",
		common.NowMilliseconds(), totalBlocks, total, pendingLen, txRemoteCount, localTxsCount, txLocalCount, queuedLen, localTxsCount, remoteTxsCount, pendingDuration, miningDuration)

	return total, pendingDuration, miningDuration
}

//nolint:paralleltest
func TestPoolMiningDataRaces(t *testing.T) {
	if testing.Short() {
		t.Skip("only for data race testing")
	}

	const format = "size %d, txs ticker %v, api ticker %v"

	cases := []struct {
		name              string
		size              int
		txsTickerDuration time.Duration
		apiTickerDuration time.Duration
	}{
		{
			size:              1,
			txsTickerDuration: 200 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
		{
			size:              1,
			txsTickerDuration: 400 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
		{
			size:              1,
			txsTickerDuration: 600 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
		{
			size:              1,
			txsTickerDuration: 800 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},

		{
			size:              5,
			txsTickerDuration: 200 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
		{
			size:              5,
			txsTickerDuration: 400 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
		{
			size:              5,
			txsTickerDuration: 600 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
		{
			size:              5,
			txsTickerDuration: 800 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},

		{
			size:              10,
			txsTickerDuration: 200 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
		{
			size:              10,
			txsTickerDuration: 400 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
		{
			size:              10,
			txsTickerDuration: 600 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
		{
			size:              10,
			txsTickerDuration: 800 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},

		{
			size:              20,
			txsTickerDuration: 200 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
		{
			size:              20,
			txsTickerDuration: 400 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
		{
			size:              20,
			txsTickerDuration: 600 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
		{
			size:              20,
			txsTickerDuration: 800 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},

		{
			size:              30,
			txsTickerDuration: 200 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
		{
			size:              30,
			txsTickerDuration: 400 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
		{
			size:              30,
			txsTickerDuration: 600 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
		{
			size:              30,
			txsTickerDuration: 800 * time.Millisecond,
			apiTickerDuration: 10 * time.Millisecond,
		},
	}

	for i := range cases {
		cases[i].name = fmt.Sprintf(format, cases[i].size, cases[i].txsTickerDuration, cases[i].apiTickerDuration)
	}

	//nolint:paralleltest
	for _, testCase := range cases {
		singleCase := testCase

		t.Run(singleCase.name, func(t *testing.T) {
			defer goleak.VerifyNone(t, leak.IgnoreList()...)

			const (
				blocks          = 300
				blockGasLimit   = 40_000_000
				blockPeriod     = time.Second
				threads         = 10
				batchesSize     = 10_000
				timeoutDuration = 10 * blockPeriod

				balanceStr = "1_000_000_000_000"
			)

			apiWithMining(t, balanceStr, batchesSize, singleCase, timeoutDuration, threads, blockPeriod, blocks, blockGasLimit)
		})
	}
}

//nolint:gocognit,thelper
func apiWithMining(tb testing.TB, balanceStr string, batchesSize int, singleCase struct {
	name              string
	size              int
	txsTickerDuration time.Duration
	apiTickerDuration time.Duration
}, timeoutDuration time.Duration, threads int, blockPeriod time.Duration, blocks int, blockGasLimit uint64) {
	done := make(chan struct{})

	var wg sync.WaitGroup

	defer func() {
		close(done)

		tb.Logf("[%s] finishing apiWithMining", common.NowMilliseconds())

		wg.Wait()

		tb.Logf("[%s] apiWithMining finished", common.NowMilliseconds())
	}()

	// Generate a batch of transactions to enqueue into the pool
	pendingAddedCh := make(chan struct{}, 1024)

	pool, localKey := setupTxPoolWithConfig(params.TestChainConfig, testTxPoolConfig, txPoolGasLimit, MakeWithPromoteTxCh(pendingAddedCh))
	defer pool.Stop()

	localKeyPub := localKey.PublicKey
	account := crypto.PubkeyToAddress(localKeyPub)

	balance, ok := big.NewInt(0).SetString(balanceStr, 0)
	if !ok {
		tb.Fatal("incorrect initial balance", balanceStr)
	}

	testAddBalance(pool, account, balance)

	signer := types.NewEIP155Signer(big.NewInt(1))
	baseFee := uint256.NewInt(1)

	batchesLocal := make([]types.Transactions, batchesSize)
	batchesRemote := make([]types.Transactions, batchesSize)
	batchesRemotes := make([]types.Transactions, batchesSize)
	batchesRemoteSync := make([]types.Transactions, batchesSize)
	batchesRemotesSync := make([]types.Transactions, batchesSize)

	for i := 0; i < batchesSize; i++ {
		batchesLocal[i] = make(types.Transactions, singleCase.size)

		for j := 0; j < singleCase.size; j++ {
			batchesLocal[i][j] = pricedTransaction(uint64(singleCase.size*i+j), 100_000, big.NewInt(int64(i+1)), localKey)
		}

		batchesRemote[i] = make(types.Transactions, singleCase.size)

		remoteKey, _ := crypto.GenerateKey()
		remoteAddr := crypto.PubkeyToAddress(remoteKey.PublicKey)
		testAddBalance(pool, remoteAddr, balance)

		for j := 0; j < singleCase.size; j++ {
			batchesRemote[i][j] = pricedTransaction(uint64(j), 100_000, big.NewInt(int64(i+1)), remoteKey)
		}

		batchesRemotes[i] = make(types.Transactions, singleCase.size)

		remotesKey, _ := crypto.GenerateKey()
		remotesAddr := crypto.PubkeyToAddress(remotesKey.PublicKey)
		testAddBalance(pool, remotesAddr, balance)

		for j := 0; j < singleCase.size; j++ {
			batchesRemotes[i][j] = pricedTransaction(uint64(j), 100_000, big.NewInt(int64(i+1)), remotesKey)
		}

		batchesRemoteSync[i] = make(types.Transactions, singleCase.size)

		remoteSyncKey, _ := crypto.GenerateKey()
		remoteSyncAddr := crypto.PubkeyToAddress(remoteSyncKey.PublicKey)
		testAddBalance(pool, remoteSyncAddr, balance)

		for j := 0; j < singleCase.size; j++ {
			batchesRemoteSync[i][j] = pricedTransaction(uint64(j), 100_000, big.NewInt(int64(i+1)), remoteSyncKey)
		}

		batchesRemotesSync[i] = make(types.Transactions, singleCase.size)

		remotesSyncKey, _ := crypto.GenerateKey()
		remotesSyncAddr := crypto.PubkeyToAddress(remotesSyncKey.PublicKey)
		testAddBalance(pool, remotesSyncAddr, balance)

		for j := 0; j < singleCase.size; j++ {
			batchesRemotesSync[i][j] = pricedTransaction(uint64(j), 100_000, big.NewInt(int64(i+1)), remotesSyncKey)
		}
	}

	tb.Logf("[%s] starting goroutines", common.NowMilliseconds())

	txsTickerDuration := singleCase.txsTickerDuration
	apiTickerDuration := singleCase.apiTickerDuration

	// locals
	wg.Add(1)

	go func() {
		defer func() {
			tb.Logf("[%s] stopping AddLocal(s)", common.NowMilliseconds())

			wg.Done()

			tb.Logf("[%s] stopped AddLocal(s)", common.NowMilliseconds())
		}()

		tb.Logf("[%s] starting AddLocal(s)", common.NowMilliseconds())

		for _, batch := range batchesLocal {
			batch := batch

			select {
			case <-done:
				return
			default:
			}

			if rand.Int()%2 == 0 {
				runWithTimeout(tb, func(_ chan struct{}) {
					errs := pool.AddLocals(batch)
					if len(errs) != 0 {
						tb.Logf("[%s] AddLocals error, %v", common.NowMilliseconds(), errs)
					}
				}, done, "AddLocals", timeoutDuration, 0, 0)
			} else {
				for _, tx := range batch {
					tx := tx

					runWithTimeout(tb, func(_ chan struct{}) {
						err := pool.AddLocal(tx)
						if err != nil {
							tb.Logf("[%s] AddLocal error %s", common.NowMilliseconds(), err)
						}
					}, done, "AddLocal", timeoutDuration, 0, 0)

					time.Sleep(txsTickerDuration)
				}
			}

			time.Sleep(txsTickerDuration)
		}
	}()

	// remotes
	wg.Add(1)

	go func() {
		defer func() {
			tb.Logf("[%s] stopping AddRemotes", common.NowMilliseconds())

			wg.Done()

			tb.Logf("[%s] stopped AddRemotes", common.NowMilliseconds())
		}()

		addTransactionsBatches(tb, batchesRemotes, getFnForBatches(pool.AddRemotes), done, timeoutDuration, txsTickerDuration, "AddRemotes", 0)
	}()

	// remote
	wg.Add(1)

	go func() {
		defer func() {
			tb.Logf("[%s] stopping AddRemote", common.NowMilliseconds())

			wg.Done()

			tb.Logf("[%s] stopped AddRemote", common.NowMilliseconds())
		}()

		addTransactions(tb, batchesRemote, pool.AddRemote, done, timeoutDuration, txsTickerDuration, "AddRemote", 0)
	}()

	// sync
	// remotes
	wg.Add(1)

	go func() {
		defer func() {
			tb.Logf("[%s] stopping AddRemotesSync", common.NowMilliseconds())

			wg.Done()

			tb.Logf("[%s] stopped AddRemotesSync", common.NowMilliseconds())
		}()

		addTransactionsBatches(tb, batchesRemotesSync, getFnForBatches(pool.AddRemotesSync), done, timeoutDuration, txsTickerDuration, "AddRemotesSync", 0)
	}()

	// remote
	wg.Add(1)

	go func() {
		defer func() {
			tb.Logf("[%s] stopping AddRemoteSync", common.NowMilliseconds())

			wg.Done()

			tb.Logf("[%s] stopped AddRemoteSync", common.NowMilliseconds())
		}()

		addTransactions(tb, batchesRemoteSync, pool.AddRemoteSync, done, timeoutDuration, txsTickerDuration, "AddRemoteSync", 0)
	}()

	// tx pool API
	for i := 0; i < threads; i++ {
		i := i

		wg.Add(1)

		go func() {
			defer func() {
				tb.Logf("[%s] stopping Pending-no-tips, thread %d", common.NowMilliseconds(), i)

				wg.Done()

				tb.Logf("[%s] stopped Pending-no-tips, thread %d", common.NowMilliseconds(), i)
			}()

			runWithTicker(tb, func(_ chan struct{}) {
				p := pool.Pending(context.Background(), false)
				fmt.Fprint(io.Discard, p)
			}, done, "Pending-no-tips", apiTickerDuration, timeoutDuration, i)
		}()

		wg.Add(1)

		go func() {
			defer func() {
				tb.Logf("[%s] stopping Pending-with-tips, thread %d", common.NowMilliseconds(), i)

				wg.Done()

				tb.Logf("[%s] stopped Pending-with-tips, thread %d", common.NowMilliseconds(), i)
			}()

			runWithTicker(tb, func(_ chan struct{}) {
				p := pool.Pending(context.Background(), true)
				fmt.Fprint(io.Discard, p)
			}, done, "Pending-with-tips", apiTickerDuration, timeoutDuration, i)
		}()

		wg.Add(1)

		go func() {
			defer func() {
				tb.Logf("[%s] stopping Locals, thread %d", common.NowMilliseconds(), i)

				wg.Done()

				tb.Logf("[%s] stopped Locals, thread %d", common.NowMilliseconds(), i)
			}()

			runWithTicker(tb, func(_ chan struct{}) {
				l := pool.Locals()
				fmt.Fprint(io.Discard, l)
			}, done, "Locals", apiTickerDuration, timeoutDuration, i)
		}()

		wg.Add(1)

		go func() {
			defer func() {
				tb.Logf("[%s] stopping Content, thread %d", common.NowMilliseconds(), i)

				wg.Done()

				tb.Logf("[%s] stopped Content, thread %d", common.NowMilliseconds(), i)
			}()

			runWithTicker(tb, func(_ chan struct{}) {
				p, q := pool.Content()
				fmt.Fprint(io.Discard, p, q)
			}, done, "Content", apiTickerDuration, timeoutDuration, i)
		}()

		wg.Add(1)

		go func() {
			defer func() {
				tb.Logf("[%s] stopping GasPriceUint256, thread %d", common.NowMilliseconds(), i)

				wg.Done()

				tb.Logf("[%s] stopped GasPriceUint256, thread %d", common.NowMilliseconds(), i)
			}()

			runWithTicker(tb, func(_ chan struct{}) {
				res := pool.GasPriceUint256()
				fmt.Fprint(io.Discard, res)
			}, done, "GasPriceUint256", apiTickerDuration, timeoutDuration, i)
		}()

		wg.Add(1)

		go func() {
			defer func() {
				tb.Logf("[%s] stopping GasPrice, thread %d", common.NowMilliseconds(), i)

				wg.Done()

				tb.Logf("[%s] stopped GasPrice, thread %d", common.NowMilliseconds(), i)
			}()

			runWithTicker(tb, func(_ chan struct{}) {
				res := pool.GasPrice()
				fmt.Fprint(io.Discard, res)
			}, done, "GasPrice", apiTickerDuration, timeoutDuration, i)
		}()

		wg.Add(1)

		go func() {
			defer func() {
				tb.Logf("[%s] stopping SetGasPrice, thread %d", common.NowMilliseconds(), i)

				wg.Done()

				tb.Logf("[%s] stopped SetGasPrice, , thread %d", common.NowMilliseconds(), i)
			}()

			runWithTicker(tb, func(_ chan struct{}) {
				pool.SetGasPrice(pool.GasPrice())
			}, done, "SetGasPrice", apiTickerDuration, timeoutDuration, i)
		}()

		wg.Add(1)

		go func() {
			defer func() {
				tb.Logf("[%s] stopping ContentFrom, thread %d", common.NowMilliseconds(), i)

				wg.Done()

				tb.Logf("[%s] stopped ContentFrom, thread %d", common.NowMilliseconds(), i)
			}()

			runWithTicker(tb, func(_ chan struct{}) {
				p, q := pool.ContentFrom(account)
				fmt.Fprint(io.Discard, p, q)
			}, done, "ContentFrom", apiTickerDuration, timeoutDuration, i)
		}()

		wg.Add(1)

		go func() {
			defer func() {
				tb.Logf("[%s] stopping Has, thread %d", common.NowMilliseconds(), i)

				wg.Done()

				tb.Logf("[%s] stopped Has, thread %d", common.NowMilliseconds(), i)
			}()

			runWithTicker(tb, func(_ chan struct{}) {
				res := pool.Has(batchesRemotes[0][0].Hash())
				fmt.Fprint(io.Discard, res)
			}, done, "Has", apiTickerDuration, timeoutDuration, i)
		}()

		wg.Add(1)

		go func() {
			defer func() {
				tb.Logf("[%s] stopping Get, thread %d", common.NowMilliseconds(), i)

				wg.Done()

				tb.Logf("[%s] stopped Get, thread %d", common.NowMilliseconds(), i)
			}()

			runWithTicker(tb, func(_ chan struct{}) {
				tx := pool.Get(batchesRemotes[0][0].Hash())
				fmt.Fprint(io.Discard, tx == nil)
			}, done, "Get", apiTickerDuration, timeoutDuration, i)
		}()

		wg.Add(1)

		go func() {
			defer func() {
				tb.Logf("[%s] stopping Nonce, thread %d", common.NowMilliseconds(), i)

				wg.Done()

				tb.Logf("[%s] stopped Nonce, thread %d", common.NowMilliseconds(), i)
			}()

			runWithTicker(tb, func(_ chan struct{}) {
				res := pool.Nonce(account)
				fmt.Fprint(io.Discard, res)
			}, done, "Nonce", apiTickerDuration, timeoutDuration, i)
		}()

		wg.Add(1)

		go func() {
			defer func() {
				tb.Logf("[%s] stopping Stats, thread %d", common.NowMilliseconds(), i)

				wg.Done()

				tb.Logf("[%s] stopped Stats, thread %d", common.NowMilliseconds(), i)
			}()

			runWithTicker(tb, func(_ chan struct{}) {
				p, q := pool.Stats()
				fmt.Fprint(io.Discard, p, q)
			}, done, "Stats", apiTickerDuration, timeoutDuration, i)
		}()

		wg.Add(1)

		go func() {
			defer func() {
				tb.Logf("[%s] stopping Status, thread %d", common.NowMilliseconds(), i)

				wg.Done()

				tb.Logf("[%s] stopped Status, thread %d", common.NowMilliseconds(), i)
			}()

			runWithTicker(tb, func(_ chan struct{}) {
				st := pool.Status([]common.Hash{batchesRemotes[1][0].Hash()})
				fmt.Fprint(io.Discard, st)
			}, done, "Status", apiTickerDuration, timeoutDuration, i)
		}()

		wg.Add(1)

		go func() {
			defer func() {
				tb.Logf("[%s] stopping SubscribeNewTxsEvent, thread %d", common.NowMilliseconds(), i)

				wg.Done()

				tb.Logf("[%s] stopped SubscribeNewTxsEvent, thread %d", common.NowMilliseconds(), i)
			}()

			runWithTicker(tb, func(c chan struct{}) {
				ch := make(chan NewTxsEvent, 10)
				sub := pool.SubscribeNewTxsEvent(ch)

				if sub == nil {
					return
				}

				defer sub.Unsubscribe()

				select {
				case <-done:
					return
				case <-c:
				case res := <-ch:
					fmt.Fprint(io.Discard, res)
				}

			}, done, "SubscribeNewTxsEvent", apiTickerDuration, timeoutDuration, i)
		}()
	}

	// wait for the start
	tb.Logf("[%s] before the first propagated transaction", common.NowMilliseconds())
	<-pendingAddedCh
	tb.Logf("[%s] after the first propagated transaction", common.NowMilliseconds())

	var (
		totalTxs    int
		totalBlocks int
	)

	pendingDurations := make([]time.Duration, 0, blocks)

	var (
		added           int
		pendingDuration time.Duration
		miningDuration  time.Duration
		diff            time.Duration
	)

	for {
		added, pendingDuration, miningDuration = mining(tb, pool, signer, baseFee, blockGasLimit, totalBlocks)

		totalTxs += added

		pendingDurations = append(pendingDurations, pendingDuration)

		totalBlocks++

		if totalBlocks > blocks {
			fmt.Fprint(io.Discard, totalTxs)
			break
		}

		diff = blockPeriod - miningDuration
		if diff > 0 {
			time.Sleep(diff)
		}
	}

	pendingDurationsFloat := make([]float64, len(pendingDurations))

	for i, v := range pendingDurations {
		pendingDurationsFloat[i] = float64(v.Nanoseconds())
	}

	mean, stddev := stat.MeanStdDev(pendingDurationsFloat, nil)
	tb.Logf("[%s] pending mean %v, stddev %v, %v-%v",
		common.NowMilliseconds(), time.Duration(mean), time.Duration(stddev), time.Duration(floats.Min(pendingDurationsFloat)), time.Duration(floats.Max(pendingDurationsFloat)))
}

func addTransactionsBatches(tb testing.TB, batches []types.Transactions, fn func(types.Transactions) error, done chan struct{}, timeoutDuration time.Duration, tickerDuration time.Duration, name string, thread int) {
	tb.Helper()

	tb.Logf("[%s] starting %s", common.NowMilliseconds(), name)

	defer func() {
		tb.Logf("[%s] stop %s", common.NowMilliseconds(), name)
	}()

	for _, batch := range batches {
		batch := batch

		select {
		case <-done:
			return
		default:
		}

		runWithTimeout(tb, func(_ chan struct{}) {
			err := fn(batch)
			if err != nil {
				tb.Logf("[%s] %s error: %s", common.NowMilliseconds(), name, err)
			}
		}, done, name, timeoutDuration, 0, thread)

		time.Sleep(tickerDuration)
	}
}

func addTransactions(tb testing.TB, batches []types.Transactions, fn func(*types.Transaction) error, done chan struct{}, timeoutDuration time.Duration, tickerDuration time.Duration, name string, thread int) {
	tb.Helper()

	tb.Logf("[%s] starting %s", common.NowMilliseconds(), name)

	defer func() {
		tb.Logf("[%s] stop %s", common.NowMilliseconds(), name)
	}()

	for _, batch := range batches {
		for _, tx := range batch {
			tx := tx

			select {
			case <-done:
				return
			default:
			}

			runWithTimeout(tb, func(_ chan struct{}) {
				err := fn(tx)
				if err != nil {
					tb.Logf("%s error: %s", name, err)
				}
			}, done, name, timeoutDuration, 0, thread)

			time.Sleep(tickerDuration)
		}

		time.Sleep(tickerDuration)
	}
}

func getFnForBatches(fn func([]*types.Transaction) []error) func(types.Transactions) error {
	return func(batch types.Transactions) error {
		errs := fn(batch)
		if len(errs) != 0 {
			return errs[0]
		}

		return nil
	}
}

//nolint:unparam
func runWithTicker(tb testing.TB, fn func(c chan struct{}), done chan struct{}, name string, tickerDuration, timeoutDuration time.Duration, thread int) {
	tb.Helper()

	select {
	case <-done:
		tb.Logf("[%s] Short path. finishing outer runWithTicker for %q, thread %d", common.NowMilliseconds(), name, thread)

		return
	default:
	}

	defer func() {
		tb.Logf("[%s] finishing outer runWithTicker for %q, thread %d", common.NowMilliseconds(), name, thread)
	}()

	localTicker := time.NewTicker(tickerDuration)
	defer localTicker.Stop()

	n := 0

	for range localTicker.C {
		select {
		case <-done:
			return
		default:
		}

		runWithTimeout(tb, fn, done, name, timeoutDuration, n, thread)

		n++
	}
}

func runWithTimeout(tb testing.TB, fn func(chan struct{}), outerDone chan struct{}, name string, timeoutDuration time.Duration, n, thread int) {
	tb.Helper()

	select {
	case <-outerDone:
		tb.Logf("[%s] Short path. exiting inner runWithTimeout by outer exit event for %q, thread %d, iteration %d", common.NowMilliseconds(), name, thread, n)

		return
	default:
	}

	timeout := time.NewTimer(timeoutDuration)
	defer timeout.Stop()

	doneCh := make(chan struct{})

	isError := new(int32)
	*isError = 0

	go func() {
		defer close(doneCh)

		select {
		case <-outerDone:
			return
		default:
			fn(doneCh)
		}
	}()

	const isDebug = false

	var stack string

	select {
	case <-outerDone:
		tb.Logf("[%s] exiting inner runWithTimeout by outer exit event for %q, thread %d, iteration %d", common.NowMilliseconds(), name, thread, n)
	case <-doneCh:
		// only for debug
		//tb.Logf("[%s] exiting inner runWithTimeout by successful call for %q, thread %d, iteration %d", common.NowMilliseconds(), name, thread, n)
	case <-timeout.C:
		atomic.StoreInt32(isError, 1)

		if isDebug {
			stack = string(debug.Stack(true))
		}

		tb.Errorf("[%s] %s timeouted, thread %d, iteration %d. Stack %s", common.NowMilliseconds(), name, thread, n, stack)
	}
}
