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

package legacypool

import (
	"crypto/ecdsa"
	crand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	crand2 "github.com/maticnetwork/crand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

var (
	// testTxPoolConfig is a transaction pool configuration without stateful disk
	// sideeffects used during testing.
	testTxPoolConfig Config

	// eip1559Config is a chain config with EIP-1559 enabled at block 0.
	eip1559Config *params.ChainConfig
)

const (
// txPoolGasLimit = 10_000_000
)

func init() {
	testTxPoolConfig = DefaultConfig
	testTxPoolConfig.Journal = ""
	/*
		Given the introduction of `BorDefaultTxPoolPriceLimit=25gwei`,
		we set `testTxPoolConfig.PriceLimit = 1` to avoid rewriting all `legacypool_test.go` tests,
		causing code divergence from geth, as this has been widely tested on different networks.
		Also, `worker_test.go` has been adapted to reflect such changes.
		Furthermore, config test can be found in `TestTxPoolDefaultPriceLimit`
	*/
	testTxPoolConfig.PriceLimit = 1

	cpy := *params.TestChainConfig
	eip1559Config = &cpy
	eip1559Config.BerlinBlock = common.Big0
	eip1559Config.LondonBlock = common.Big0
}

type testBlockChain struct {
	config        *params.ChainConfig
	gasLimit      atomic.Uint64
	statedb       *state.StateDB
	chainHeadFeed *event.Feed
}

func newTestBlockChain(config *params.ChainConfig, gasLimit uint64, statedb *state.StateDB, chainHeadFeed *event.Feed) *testBlockChain {
	bc := testBlockChain{config: config, statedb: statedb, chainHeadFeed: new(event.Feed)}
	bc.gasLimit.Store(gasLimit)

	return &bc
}

func (bc *testBlockChain) Config() *params.ChainConfig {
	return bc.config
}

func (bc *testBlockChain) CurrentBlock() *types.Header {
	return &types.Header{
		Number:     new(big.Int),
		Difficulty: common.Big0,
		GasLimit:   bc.gasLimit.Load(),
	}
}

func (bc *testBlockChain) GetBlock(hash common.Hash, number uint64) *types.Block {
	return types.NewBlock(bc.CurrentBlock(), nil, nil, trie.NewStackTrie(nil))
}

func (bc *testBlockChain) StateAt(common.Hash) (*state.StateDB, error) {
	return bc.statedb, nil
}

func (bc *testBlockChain) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
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
	crand.Read(data)

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

type unsignedAuth struct {
	nonce uint64
	key   *ecdsa.PrivateKey
}

func setCodeTx(nonce uint64, key *ecdsa.PrivateKey, unsigned []unsignedAuth) *types.Transaction {
	return pricedSetCodeTx(nonce, 250000, uint256.NewInt(1000), uint256.NewInt(1), key, unsigned)
}

func pricedSetCodeTx(nonce uint64, gaslimit uint64, gasFee, tip *uint256.Int, key *ecdsa.PrivateKey, unsigned []unsignedAuth) *types.Transaction {
	var authList []types.SetCodeAuthorization
	for _, u := range unsigned {
		auth, _ := types.SignSetCode(u.key, types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(params.TestChainConfig.ChainID),
			Address: common.Address{0x42},
			Nonce:   u.nonce,
		})
		authList = append(authList, auth)
	}
	return pricedSetCodeTxWithAuth(nonce, gaslimit, gasFee, tip, key, authList)
}

func pricedSetCodeTxWithAuth(nonce uint64, gaslimit uint64, gasFee, tip *uint256.Int, key *ecdsa.PrivateKey, authList []types.SetCodeAuthorization) *types.Transaction {
	return types.MustSignNewTx(key, types.LatestSignerForChainID(params.TestChainConfig.ChainID), &types.SetCodeTx{
		ChainID:    uint256.MustFromBig(params.TestChainConfig.ChainID),
		Nonce:      nonce,
		GasTipCap:  tip,
		GasFeeCap:  gasFee,
		Gas:        gaslimit,
		To:         common.Address{},
		Value:      uint256.NewInt(100),
		Data:       nil,
		AccessList: nil,
		AuthList:   authList,
	})
}

func setupPool() (*LegacyPool, *ecdsa.PrivateKey) {
	return setupPoolWithConfig(params.TestChainConfig)
}

// reserver is a utility struct to sanity check that accounts are
// properly reserved by the blobpool (no duplicate reserves or unreserves).
type reserver struct {
	accounts map[common.Address]struct{}
	lock     sync.RWMutex
}

func newReserver() txpool.Reserver {
	return &reserver{accounts: make(map[common.Address]struct{})}
}

func (r *reserver) Hold(addr common.Address) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if _, exists := r.accounts[addr]; exists {
		panic("already reserved")
	}
	r.accounts[addr] = struct{}{}
	return nil
}

func (r *reserver) Release(addr common.Address) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if _, exists := r.accounts[addr]; !exists {
		panic("not reserved")
	}
	delete(r.accounts, addr)
	return nil
}

func (r *reserver) Has(address common.Address) bool {
	return false // reserver only supports a single pool
}

func setupPoolWithConfig(config *params.ChainConfig, options ...func(pool *LegacyPool)) (*LegacyPool, *ecdsa.PrivateKey) {
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(config, 10000000, statedb, new(event.Feed))

	key, _ := crypto.GenerateKey()
	pool := New(testTxPoolConfig, blockchain, options...)
	if err := pool.Init(testTxPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver()); err != nil {
		panic(err)
	}
	// wait for the pool to initialize
	<-pool.initDoneCh

	return pool, key
}

// validatePoolInternals checks various consistency invariants within the pool.
func validatePoolInternals(pool *LegacyPool) error {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	// Ensure the total transaction set is consistent with pending + queued
	pending, queued := pool.stats()
	if total := pool.all.Count(); total != pending+queued {
		return fmt.Errorf("total transaction count %d != %d pending + %d queued", total, pending, queued)
	}

	pool.priced.Reheap()
	priced, remote := pool.priced.urgent.Len()+pool.priced.floating.Len(), pool.all.Count()
	if priced != remote {
		return fmt.Errorf("total priced transaction count %d != %d", priced, remote)
	}

	// Ensure the next nonce to assign is the correct one
	// pool.pendingMu.RLock()
	// defer pool.pendingMu.RUnlock()

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
	// Ensure all auths in pool are tracked
	for _, tx := range pool.all.txs {
		for _, addr := range tx.SetCodeAuthorities() {
			list := pool.all.auths[addr]
			if i := slices.Index(list, tx.Hash()); i < 0 {
				return fmt.Errorf("authority not tracked: addr %s, tx %s", addr, tx.Hash())
			}
		}
	}
	// Ensure all auths in pool have an associated tx.
	for addr, hashes := range pool.all.auths {
		for _, hash := range hashes {
			if _, ok := pool.all.txs[hash]; !ok {
				return fmt.Errorf("dangling authority, missing originating tx: addr %s, hash %s", addr, hash.Hex())
			}
		}
	}
	return nil
}

// validateEvents checks that the correct number of transaction addition events
// were fired on the pool's event feed.
func validateEvents(events chan core.NewTxsEvent, count int) error {
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
		c.statedb, _ = state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		// simulate that the new head block included tx0 and tx1
		c.statedb.SetNonce(c.address, 2, tracing.NonceChangeUnspecified)
		c.statedb.SetBalance(c.address, new(uint256.Int).SetUint64(params.Ether), tracing.BalanceChangeUnspecified)
		*c.trigger = false
	}

	return stdb, nil
}

// TestTxPoolDefaultPriceLimit ensures the bor default tx pool price limit is set correctly.
func TestTxPoolDefaultPriceLimit(t *testing.T) {
	t.Parallel()

	pool, _ := setupPool()
	defer pool.Close()

	if have, want := pool.config.PriceLimit, uint64(params.BorDefaultTxPoolPriceLimit); have != want {
		t.Fatalf("txpool price limit incorrect: have %d, want %d", have, want)
	}
}

// This test simulates a scenario where a new block is imported during a
// state reset and tests whether the pending state is in sync with the
// block head event that initiated the resetState().
func TestStateChangeDuringReset(t *testing.T) {
	t.Parallel()

	var (
		key, _     = crypto.GenerateKey()
		address    = crypto.PubkeyToAddress(key.PublicKey)
		statedb, _ = state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		trigger    = false
	)

	// setup pool with 2 transaction in it
	statedb.SetBalance(address, new(uint256.Int).SetUint64(params.Ether), tracing.BalanceChangeUnspecified)
	blockchain := &testChain{newTestBlockChain(params.TestChainConfig, 1000000000, statedb, new(event.Feed)), address, &trigger}

	tx0 := transaction(0, 100000, key)
	tx1 := transaction(1, 100000, key)

	pool := New(testTxPoolConfig, blockchain)
	pool.Init(testTxPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	nonce := pool.Nonce(address)
	if nonce != 0 {
		t.Fatalf("Invalid nonce, want 0, got %d", nonce)
	}

	pool.addRemotesSync([]*types.Transaction{tx0, tx1})

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

func testAddBalance(pool *LegacyPool, addr common.Address, amount *big.Int) {
	pool.mu.Lock()
	pool.currentState.AddBalance(addr, uint256.MustFromBig(amount), tracing.BalanceChangeUnspecified)
	pool.mu.Unlock()
}

func testSetNonce(pool *LegacyPool, addr common.Address, nonce uint64) {
	pool.mu.Lock()
	pool.currentState.SetNonce(addr, nonce, tracing.NonceChangeUnspecified)
	pool.mu.Unlock()
}

// func getBalance(pool *LegacyPool, addr common.Address) *big.Int {
// 	bal := big.NewInt(0)

// 	pool.mu.Lock()
// 	bal.Set(pool.currentState.GetBalance(addr))
// 	pool.mu.Unlock()

// 	return bal
// }

func TestInvalidTransactions(t *testing.T) {
	t.Parallel()

	pool, key := setupPool()
	defer pool.Close()

	tx := transaction(0, 100, key)
	from, _ := deriveSender(tx)

	// Intrinsic gas too low
	testAddBalance(pool, from, big.NewInt(1))
	if err, want := pool.addRemote(tx), core.ErrIntrinsicGas; !errors.Is(err, want) {
		t.Errorf("want %v have %v", want, err)
	}

	// Insufficient funds
	tx = transaction(0, 100000, key)
	if err, want := pool.addRemote(tx), core.ErrInsufficientFunds; !errors.Is(err, want) {
		t.Errorf("want %v have %v", want, err)
	}

	testSetNonce(pool, from, 1)
	testAddBalance(pool, from, big.NewInt(0xffffffffffffff))

	tx = transaction(0, 100000, key)
	if err, want := pool.addRemote(tx), core.ErrNonceTooLow; !errors.Is(err, want) {
		t.Errorf("want %v have %v", want, err)
	}

	tx = transaction(1, 100000, key)
	pool.gasTip.Store(uint256.NewInt(1000))
	if err, want := pool.addRemote(tx), txpool.ErrTxGasPriceTooLow; !errors.Is(err, want) {
		t.Errorf("want %v have %v", want, err)
	}
}

func TestQueue(t *testing.T) {
	t.Parallel()

	pool, key := setupPool()
	defer pool.Close()

	tx := transaction(0, 100, key)
	from, _ := deriveSender(tx)
	testAddBalance(pool, from, big.NewInt(1000))
	<-pool.requestReset(nil, nil)

	pool.enqueueTx(tx.Hash(), tx, true)
	<-pool.requestPromoteExecutables(newAccountSet(pool.signer, from))

	// pool.pendingMu.RLock()
	if len(pool.pending) != 1 {
		t.Error("expected valid txs to be 1 is", len(pool.pending))
	}
	// pool.pendingMu.RUnlock()

	tx = transaction(1, 100, key)
	from, _ = deriveSender(tx)
	testSetNonce(pool, from, 2)
	pool.enqueueTx(tx.Hash(), tx, true)

	<-pool.requestPromoteExecutables(newAccountSet(pool.signer, from))

	// pool.pendingMu.RLock()
	if _, ok := pool.pending[from].txs.items[tx.Nonce()]; ok {
		t.Error("expected transaction to be in tx pool")
	}
	// pool.pendingMu.RUnlock()

	if len(pool.queue) > 0 {
		t.Error("expected transaction queue to be empty. is", len(pool.queue))
	}
}

func TestQueue2(t *testing.T) {
	t.Parallel()

	pool, key := setupPool()
	defer pool.Close()

	tx1 := transaction(0, 100, key)
	tx2 := transaction(10, 100, key)
	tx3 := transaction(11, 100, key)
	from, _ := deriveSender(tx1)
	testAddBalance(pool, from, big.NewInt(1000))
	pool.reset(nil, nil)

	pool.enqueueTx(tx1.Hash(), tx1, true)
	pool.enqueueTx(tx2.Hash(), tx2, true)
	pool.enqueueTx(tx3.Hash(), tx3, true)

	pool.promoteExecutables([]common.Address{from})

	// pool.pendingMu.RLock()
	if len(pool.pending) != 1 {
		t.Error("expected pending length to be 1, got", len(pool.pending))
	}
	// pool.pendingMu.RUnlock()

	if pool.queue[from].Len() != 2 {
		t.Error("expected len(queue) == 2, got", pool.queue[from].Len())
	}
}

func TestNegativeValue(t *testing.T) {
	t.Parallel()

	pool, key := setupPool()
	defer pool.Close()

	tx, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(-1), 100, big.NewInt(1), nil), types.HomesteadSigner{}, key)
	from, _ := deriveSender(tx)

	testAddBalance(pool, from, big.NewInt(1))
	if err := pool.addRemote(tx); !errors.Is(err, txpool.ErrNegativeValue) {
		t.Error("expected", txpool.ErrNegativeValue, "got", err)
	}
}

func TestTipAboveFeeCap(t *testing.T) {
	t.Parallel()

	pool, key := setupPoolWithConfig(eip1559Config)
	defer pool.Close()

	tx := dynamicFeeTx(0, 100, big.NewInt(1), big.NewInt(2), key)

	if err := pool.addRemote(tx); !errors.Is(err, core.ErrTipAboveFeeCap) {
		t.Error("expected", core.ErrTipAboveFeeCap, "got", err)
	}
}

func TestVeryHighValues(t *testing.T) {
	t.Parallel()

	pool, key := setupPoolWithConfig(eip1559Config)
	defer pool.Close()

	veryBigNumber := big.NewInt(1)
	veryBigNumber.Lsh(veryBigNumber, 300)

	tx := dynamicFeeTx(0, 100, big.NewInt(1), veryBigNumber, key)
	if err := pool.addRemote(tx); !errors.Is(err, core.ErrTipVeryHigh) {
		t.Error("expected", core.ErrTipVeryHigh, "got", err)
	}

	tx2 := dynamicFeeTx(0, 100, veryBigNumber, big.NewInt(1), key)
	if err := pool.addRemote(tx2); !errors.Is(err, core.ErrFeeCapVeryHigh) {
		t.Error("expected", core.ErrFeeCapVeryHigh, "got", err)
	}
}

func TestChainFork(t *testing.T) {
	t.Parallel()

	pool, key := setupPool()
	defer pool.Close()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	resetState := func() {
		statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		statedb.AddBalance(addr, uint256.NewInt(100000000000000), tracing.BalanceChangeUnspecified)

		pool.chain = newTestBlockChain(pool.chainconfig, 1000000, statedb, new(event.Feed))
		<-pool.requestReset(nil, nil)
	}
	resetState()

	tx := transaction(0, 100000, key)
	if _, err := pool.add(tx); err != nil {
		t.Error("didn't expect error", err)
	}
	pool.removeTx(tx.Hash(), true, true)

	// reset the pool's internal state
	resetState()
	if _, err := pool.add(tx); err != nil {
		t.Error("didn't expect error", err)
	}
}

func TestDoubleNonce(t *testing.T) {
	t.Parallel()

	pool, key := setupPool()
	defer pool.Close()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	resetState := func() {
		statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		statedb.AddBalance(addr, uint256.NewInt(100000000000000), tracing.BalanceChangeUnspecified)

		pool.chain = newTestBlockChain(pool.chainconfig, 1000000, statedb, new(event.Feed))
		<-pool.requestReset(nil, nil)
	}
	resetState()

	signer := types.HomesteadSigner{}
	tx1, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(100), 100000, big.NewInt(1), nil), signer, key)
	tx2, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(100), 1000000, big.NewInt(2), nil), signer, key)
	tx3, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(100), 1000000, big.NewInt(1), nil), signer, key)

	// Add the first two transaction, ensure higher priced stays only
	if replace, err := pool.add(tx1); err != nil || replace {
		t.Errorf("first transaction insert failed (%v) or reported replacement (%v)", err, replace)
	}
	if replace, err := pool.add(tx2); err != nil || !replace {
		t.Errorf("second transaction insert failed (%v) or not reported replacement (%v)", err, replace)
	}

	<-pool.requestPromoteExecutables(newAccountSet(signer, addr))

	// pool.pendingMu.RLock()
	if pool.pending[addr].Len() != 1 {
		t.Error("expected 1 pending transactions, got", pool.pending[addr].Len())
	}

	if tx := pool.pending[addr].txs.items[0]; tx.Hash() != tx2.Hash() {
		t.Errorf("transaction mismatch: have %x, want %x", tx.Hash(), tx2.Hash())
	}
	// pool.pendingMu.RUnlock()

	// Add the third transaction and ensure it's not saved (smaller price)
	pool.add(tx3)
	<-pool.requestPromoteExecutables(newAccountSet(signer, addr))

	// pool.pendingMu.RLock()
	if pool.pending[addr].Len() != 1 {
		t.Error("expected 1 pending transactions, got", pool.pending[addr].Len())
	}

	if tx := pool.pending[addr].txs.items[0]; tx.Hash() != tx2.Hash() {
		t.Errorf("transaction mismatch: have %x, want %x", tx.Hash(), tx2.Hash())
	}
	// pool.pendingMu.RUnlock()

	// Ensure the total transaction count is correct
	if pool.all.Count() != 1 {
		t.Error("expected 1 total transactions, got", pool.all.Count())
	}
}

func TestMissingNonce(t *testing.T) {
	t.Parallel()

	pool, key := setupPool()
	defer pool.Close()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, addr, big.NewInt(100000000000000))

	tx := transaction(1, 100000, key)
	if _, err := pool.add(tx); err != nil {
		t.Error("didn't expect error", err)
	}

	// pool.pendingMu.RLock()
	if len(pool.pending) != 0 {
		t.Error("expected 0 pending transactions, got", len(pool.pending))
	}
	// pool.pendingMu.RUnlock()

	if pool.queue[addr].Len() != 1 {
		t.Error("expected 1 queued transaction, got", pool.queue[addr].Len())
	}

	if pool.all.Count() != 1 {
		t.Error("expected 1 total transactions, got", pool.all.Count())
	}
}

func TestNonceRecovery(t *testing.T) {
	t.Parallel()

	const n = 10

	pool, key := setupPool()
	defer pool.Close()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	testSetNonce(pool, addr, n)
	testAddBalance(pool, addr, big.NewInt(100000000000000))
	<-pool.requestReset(nil, nil)

	tx := transaction(n, 100000, key)
	if err := pool.addRemote(tx); err != nil {
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
func TestDropping(t *testing.T) {
	t.Parallel()

	// Create a test account and fund it
	pool, key := setupPool()
	defer pool.Close()

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
	pool.all.Add(tx0)
	pool.priced.Put(tx0)
	pool.promoteTx(account, tx0.Hash(), tx0)

	pool.all.Add(tx1)
	pool.priced.Put(tx1)
	pool.promoteTx(account, tx1.Hash(), tx1)

	pool.all.Add(tx2)
	pool.priced.Put(tx2)
	pool.promoteTx(account, tx2.Hash(), tx2)

	pool.enqueueTx(tx10.Hash(), tx10, true)
	pool.enqueueTx(tx11.Hash(), tx11, true)
	pool.enqueueTx(tx12.Hash(), tx12, true)

	// Check that pre and post validations leave the pool as is
	// pool.pendingMu.RLock()
	if pool.pending[account].Len() != 3 {
		t.Errorf("pending transaction mismatch: have %d, want %d", pool.pending[account].Len(), 3)
	}
	// pool.pendingMu.RUnlock()

	if pool.queue[account].Len() != 3 {
		t.Errorf("queued transaction mismatch: have %d, want %d", pool.queue[account].Len(), 3)
	}

	if pool.all.Count() != 6 {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), 6)
	}

	<-pool.requestReset(nil, nil)

	// pool.pendingMu.RLock()
	if pool.pending[account].Len() != 3 {
		t.Errorf("pending transaction mismatch: have %d, want %d", pool.pending[account].Len(), 3)
	}
	// pool.pendingMu.RUnlock()

	if pool.queue[account].Len() != 3 {
		t.Errorf("queued transaction mismatch: have %d, want %d", pool.queue[account].Len(), 3)
	}

	if pool.all.Count() != 6 {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), 6)
	}
	// Reduce the balance of the account, and check that invalidated transactions are dropped
	testAddBalance(pool, account, big.NewInt(-650))
	<-pool.requestReset(nil, nil)

	// pool.pendingMu.RLock()
	if _, ok := pool.pending[account].txs.items[tx0.Nonce()]; !ok {
		t.Errorf("funded pending transaction missing: %v", tx0)
	}

	if _, ok := pool.pending[account].txs.items[tx1.Nonce()]; !ok {
		t.Errorf("funded pending transaction missing: %v", tx0)
	}

	if _, ok := pool.pending[account].txs.items[tx2.Nonce()]; ok {
		t.Errorf("out-of-fund pending transaction present: %v", tx1)
	}
	// pool.pendingMu.RUnlock()

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
	pool.chain.(*testBlockChain).gasLimit.Store(100)
	<-pool.requestReset(nil, nil)

	// pool.pendingMu.RLock()
	if _, ok := pool.pending[account].txs.items[tx0.Nonce()]; !ok {
		t.Errorf("funded pending transaction missing: %v", tx0)
	}

	if _, ok := pool.pending[account].txs.items[tx1.Nonce()]; ok {
		t.Errorf("over-gased pending transaction present: %v", tx1)
	}
	// pool.pendingMu.RUnlock()

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
// nolint:gocognit
func TestPostponing(t *testing.T) {
	t.Parallel()

	// Create the pool to test the postponing with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	pool := New(testTxPoolConfig, blockchain)
	pool.Init(testTxPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

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
	for i, err := range pool.addRemotesSync(txs) {
		if err != nil {
			t.Fatalf("tx %d: failed to add transactions: %v", i, err)
		}
	}
	// Check that pre and post validations leave the pool as is
	// pool.pendingMu.RLock()
	if pending := pool.pending[accs[0]].Len() + pool.pending[accs[1]].Len(); pending != len(txs) {
		t.Errorf("pending transaction mismatch: have %d, want %d", pending, len(txs))
	}
	// pool.pendingMu.RUnlock()

	if len(pool.queue) != 0 {
		t.Errorf("queued accounts mismatch: have %d, want %d", len(pool.queue), 0)
	}

	if pool.all.Count() != len(txs) {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), len(txs))
	}

	<-pool.requestReset(nil, nil)

	// pool.pendingMu.RLock()
	if pending := pool.pending[accs[0]].Len() + pool.pending[accs[1]].Len(); pending != len(txs) {
		t.Errorf("pending transaction mismatch: have %d, want %d", pending, len(txs))
	}
	// pool.pendingMu.RUnlock()

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
	// pool.pendingMu.RLock()
	if _, ok := pool.pending[accs[0]].txs.items[txs[0].Nonce()]; !ok {
		t.Errorf("tx %d: valid and funded transaction missing from pending pool: %v", 0, txs[0])
	}
	// pool.pendingMu.RUnlock()

	if _, ok := pool.queue[accs[0]].txs.items[txs[0].Nonce()]; ok {
		t.Errorf("tx %d: valid and funded transaction present in future queue: %v", 0, txs[0])
	}

	// pool.pendingMu.RLock()
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
	// pool.pendingMu.RUnlock()

	// The second account's first transaction got invalid, check that all transactions
	// are either filtered out, or queued up for later.
	// pool.pendingMu.RLock()
	if pool.pending[accs[1]] != nil {
		t.Errorf("invalidated account still has pending transactions")
	}
	// pool.pendingMu.RUnlock()

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
func TestGapFilling(t *testing.T) {
	t.Parallel()

	// Create a test account and fund it
	pool, key := setupPool()
	defer pool.Close()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000))

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan core.NewTxsEvent, testTxPoolConfig.AccountQueue+5)

	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a pending and a queued transaction with a nonce-gap in between
	pool.addRemotesSync([]*types.Transaction{
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

	if err := validatePoolInternals(pool); err != nil {
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

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that if the transaction count belonging to a single account goes above
// some threshold, the higher transactions are dropped to prevent DOS attacks.
func TestQueueAccountLimiting(t *testing.T) {
	t.Parallel()

	// Create a test account and fund it
	pool, key := setupPool()
	defer pool.Close()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000))

	// Keep queuing up transactions and make sure all above a limit are dropped
	for i := uint64(1); i <= testTxPoolConfig.AccountQueue+5; i++ {
		if err := pool.addRemoteSync(transaction(i, 100000, key)); err != nil {
			t.Fatalf("tx %d: failed to add transaction: %v", i, err)
		}

		// pool.pendingMu.RLock()
		if len(pool.pending) != 0 {
			t.Errorf("tx %d: pending pool size mismatch: have %d, want %d", i, len(pool.pending), 0)
		}
		// pool.pendingMu.RUnlock()

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

	pool, key := setupPool()
	defer pool.Close()

	tx := dynamicFeeTx(0, 22000, big.NewInt(5), big.NewInt(2), key)
	from := crypto.PubkeyToAddress(key.PublicKey)

	pool.chainconfig.ChainID = big.NewInt(5)
	pool.signer = types.LatestSignerForChainID(pool.chainconfig.ChainID)
	testAddBalance(pool, from, big.NewInt(0xffffffffffffff))

	if err := pool.addRemote(tx); !errors.Is(err, types.ErrInvalidChainId) {
		t.Error("expected", types.ErrInvalidChainId, "got", err)
	}
}

// Test that txpool allows unprotected txs when AllowUnprotectedTxs flag is set
// FIXME: The below test causes some tests to fail randomly (probably due to parallel execution)
//
//nolint:paralleltest
func TestAllowUnprotectedTransactionWhenSet(t *testing.T) {
	t.Skip()

	pool, key := setupPool()
	defer pool.Close()

	tx := dynamicFeeTx(0, 22000, big.NewInt(5), big.NewInt(2), key)
	from := crypto.PubkeyToAddress(key.PublicKey)

	// Allow unprotected txs
	pool.config.AllowUnprotectedTxs = true
	pool.chainconfig.ChainID = big.NewInt(5)
	pool.signer = types.LatestSignerForChainID(pool.chainconfig.ChainID)
	testAddBalance(pool, from, big.NewInt(0xffffffffffffff))

	if err := pool.addRemote(tx); err != nil {
		t.Error("expected", nil, "got", err)
	}
}

// Tests that if the transaction count belonging to multiple accounts go above
// some threshold, the higher transactions are dropped to prevent DOS attacks.
//
// This logic should not hold for local transactions, unless the local tracking
// mechanism is disabled.
func TestQueueGlobalLimiting(t *testing.T) {
	t.Parallel()

	// Create the pool to test the limit enforcement with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	config := testTxPoolConfig
	config.NoLocals = true
	config.GlobalQueue = config.AccountQueue*3 - 1 // reduce the queue limits to shorten test time (-1 to make it non divisible)

	pool := New(config, blockchain)
	pool.Init(testTxPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	// Create a number of test accounts and fund them (last one will be the local)
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1000000))
	}

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
	pool.addRemotesSync(txs)

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
}

// Tests that if an account remains idle for a prolonged amount of time, any
// non-executable transactions queued up are dropped to prevent wasting resources
// on shuffling them around.
func TestQueueTimeLimiting(t *testing.T) {
	// Reduce the eviction interval to a testable amount
	defer func(old time.Duration) { evictionInterval = old }(evictionInterval)
	evictionInterval = time.Millisecond * 100

	// Create the pool to test the non-expiration enforcement
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	config := testTxPoolConfig
	config.Lifetime = time.Second

	pool := New(config, blockchain)
	pool.Init(config.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	// Create a test account to ensure remotes expire
	remote, _ := crypto.GenerateKey()

	testAddBalance(pool, crypto.PubkeyToAddress(remote.PublicKey), big.NewInt(1000000000))

	// Add the transaction and ensure it is queued up
	if err := pool.addRemote(pricedTransaction(1, 100000, big.NewInt(1), remote)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	pending, queued := pool.Stats()
	if pending != 0 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
	}
	if queued != 1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Allow the eviction interval to run
	time.Sleep(2 * evictionInterval)

	// Transactions should not be evicted from the queue yet since lifetime duration has not passed
	pending, queued = pool.Stats()
	if pending != 0 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
	}
	if queued != 1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Wait a bit for eviction to run and clean up any leftovers, and ensure only the local remains
	time.Sleep(2 * config.Lifetime)

	pending, queued = pool.Stats()
	if pending != 0 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// remove current transactions and increase nonce to prepare for a reset and cleanup
	statedb.SetNonce(crypto.PubkeyToAddress(remote.PublicKey), 2, tracing.NonceChangeUnspecified)
	<-pool.requestReset(nil, nil)

	// make sure queue, pending are cleared
	pending, queued = pool.Stats()
	if pending != 0 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Queue gapped transactions
	if err := pool.addRemoteSync(pricedTransaction(4, 100000, big.NewInt(1), remote)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	time.Sleep(5 * evictionInterval) // A half lifetime pass

	// Queue executable transactions, the life cycle should be restarted.
	if err := pool.addRemoteSync(pricedTransaction(2, 100000, big.NewInt(1), remote)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	time.Sleep(6 * evictionInterval)

	// All gapped transactions shouldn't be kicked out
	pending, queued = pool.Stats()
	if pending != 1 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 1)
	}
	if queued != 1 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
	}
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// The whole life time pass after last promotion, kick out stale transactions
	time.Sleep(2 * config.Lifetime)
	pending, queued = pool.Stats()
	if pending != 1 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 1)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that even if the transaction count belonging to a single account goes
// above some threshold, as long as the transactions are executable, they are
// accepted.
func TestPendingLimiting(t *testing.T) {
	t.Parallel()

	// Create a test account and fund it
	pool, key := setupPool()
	defer pool.Close()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000000000))

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan core.NewTxsEvent, testTxPoolConfig.AccountQueue+5)

	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Keep queuing up transactions and make sure all above a limit are dropped
	for i := uint64(0); i < testTxPoolConfig.AccountQueue+5; i++ {
		if err := pool.addRemoteSync(transaction(i, 100000, key)); err != nil {
			t.Fatalf("tx %d: failed to add transaction: %v", i, err)
		}

		// pool.pendingMu.RLock()
		if pool.pending[account].Len() != int(i)+1 {
			t.Errorf("tx %d: pending pool size mismatch: have %d, want %d", i, pool.pending[account].Len(), i+1)
		}
		// pool.pendingMu.RUnlock()

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

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that if the transaction count belonging to multiple accounts go above
// some hard threshold, the higher transactions are dropped to prevent DOS
// attacks.
func TestPendingGlobalLimiting(t *testing.T) {
	t.Parallel()

	// Create the pool to test the limit enforcement with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	config := testTxPoolConfig
	config.GlobalSlots = config.AccountSlots * 10

	pool := New(config, blockchain)
	pool.Init(config.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

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
	pool.addRemotesSync(txs)

	pending := 0

	// pool.pendingMu.RLock()
	for _, list := range pool.pending {
		pending += list.Len()
	}
	// pool.pendingMu.RUnlock()

	if pending > int(config.GlobalSlots) {
		t.Fatalf("total pending transactions overflow allowance: %d > %d", pending, config.GlobalSlots)
	}

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Test the limit on transaction size is enforced correctly.
// This test verifies every transaction having allowed size
// is added to the pool, and longer transactions are rejected.
func TestAllowedTxSize(t *testing.T) {
	t.Parallel()

	// Create a test account and fund it
	pool, key := setupPool()
	defer pool.Close()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000000))

	// Find the maximum data length for the kind of transaction which will
	// be generated in the pool.addRemoteSync calls below.
	const largeDataLength = txMaxSize - 200 // enough to have a 5 bytes RLP encoding of the data length number
	txWithLargeData := pricedDataTransaction(0, pool.currentHead.Load().GasLimit, big.NewInt(1), key, largeDataLength)
	maxTxLengthWithoutData := txWithLargeData.Size() - largeDataLength // 103 bytes
	maxTxDataLength := txMaxSize - maxTxLengthWithoutData              // 131072 - 103 = 130953 bytes

	// Try adding a transaction with maximal allowed size
	tx := pricedDataTransaction(0, pool.currentHead.Load().GasLimit, big.NewInt(1), key, maxTxDataLength)
	if err := pool.addRemoteSync(tx); err != nil {
		t.Fatalf("failed to add transaction of size %d, close to maximal: %v", int(tx.Size()), err)
	}
	// Try adding a transaction with random allowed size
	if err := pool.addRemoteSync(pricedDataTransaction(1, pool.currentHead.Load().GasLimit, big.NewInt(1), key, uint64(rand.Intn(int(maxTxDataLength+1))))); err != nil {
		t.Fatalf("failed to add transaction of random allowed size: %v", err)
	}
	// Try adding a transaction above maximum size by one
	if err := pool.addRemoteSync(pricedDataTransaction(2, pool.currentHead.Load().GasLimit, big.NewInt(1), key, maxTxDataLength+1)); err == nil {
		t.Fatalf("expected rejection on slightly oversize transaction")
	}
	// Try adding a transaction above maximum size by more than one
	if err := pool.addRemoteSync(pricedDataTransaction(2, pool.currentHead.Load().GasLimit, big.NewInt(1), key, maxTxDataLength+1+uint64(rand.Intn(10*txMaxSize)))); err == nil {
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

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that if transactions start being capped, transactions are also removed from 'all'
func TestCapClearsFromAll(t *testing.T) {
	t.Parallel()

	// Create the pool to test the limit enforcement with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	config := testTxPoolConfig
	config.AccountSlots = 2
	config.AccountQueue = 2
	config.GlobalSlots = 8

	pool := New(config, blockchain)
	pool.Init(config.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	// Create a number of test accounts and fund them
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, addr, big.NewInt(1000000))

	txs := types.Transactions{}
	for j := 0; j < int(config.GlobalSlots)*2; j++ {
		txs = append(txs, transaction(uint64(j), 100000, key))
	}
	// Import the batch and verify that limits have been enforced
	pool.addRemotes(txs)
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that if the transaction count belonging to multiple accounts go above
// some hard threshold, if they are under the minimum guaranteed slot count then
// the transactions are still kept.
func TestPendingMinimumAllowance(t *testing.T) {
	t.Parallel()

	// Create the pool to test the limit enforcement with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	config := testTxPoolConfig
	config.GlobalSlots = 1

	pool := New(config, blockchain)
	pool.Init(config.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

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
	pool.addRemotesSync(txs)

	// pool.pendingMu.RLock()
	for addr, list := range pool.pending {
		if list.Len() != int(config.AccountSlots) {
			t.Errorf("addr %x: total pending transactions mismatch: have %d, want %d", addr, list.Len(), config.AccountSlots)
		}
	}
	// pool.pendingMu.RUnlock()

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that setting the transaction pool gas price to a higher value correctly
// discards everything cheaper than that and moves any gapped transactions back
// from the pending pool to the queue.
func TestRepricing(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	pool := New(testTxPoolConfig, blockchain)
	pool.Init(testTxPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan core.NewTxsEvent, 32)

	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 3)
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

	// Import the batch and that both pending and queued transactions match up
	pool.addRemotesSync(txs)

	pending, queued := pool.Stats()
	if pending != 6 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 6)
	}

	if queued != 3 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 3)
	}
	if err := validateEvents(events, 6); err != nil {
		t.Fatalf("original event firing failed: %v", err)
	}

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Reprice the pool and check that underpriced transactions get dropped
	pool.SetGasTip(big.NewInt(2))

	pending, queued = pool.Stats()
	if pending != 1 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 1)
	}

	if queued != 5 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 5)
	}

	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("reprice event firing failed: %v", err)
	}

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Check that we can't add the old transactions back
	if err := pool.addRemote(pricedTransaction(1, 100000, big.NewInt(1), keys[0])); !errors.Is(err, txpool.ErrTxGasPriceTooLow) {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, txpool.ErrTxGasPriceTooLow)
	}
	if err := pool.addRemote(pricedTransaction(0, 100000, big.NewInt(1), keys[1])); !errors.Is(err, txpool.ErrTxGasPriceTooLow) {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, txpool.ErrTxGasPriceTooLow)
	}
	if err := pool.addRemote(pricedTransaction(2, 100000, big.NewInt(1), keys[2])); !errors.Is(err, txpool.ErrTxGasPriceTooLow) {
		t.Fatalf("adding underpriced queued transaction error mismatch: have %v, want %v", err, txpool.ErrTxGasPriceTooLow)
	}

	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("post-reprice event firing failed: %v", err)
	}

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// we can fill gaps with properly priced transactions
	if err := pool.addRemote(pricedTransaction(1, 100000, big.NewInt(2), keys[0])); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}
	if err := pool.addRemote(pricedTransaction(0, 100000, big.NewInt(2), keys[1])); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(2, 100000, big.NewInt(2), keys[2])); err != nil {
		t.Fatalf("failed to add queued transaction: %v", err)
	}

	if err := validateEvents(events, 5); err != nil {
		t.Fatalf("post-reprice event firing failed: %v", err)
	}

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

func TestMinGasPriceEnforced(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(eip1559Config, 10000000, statedb, new(event.Feed))

	txPoolConfig := DefaultConfig
	txPoolConfig.NoLocals = true
	pool := New(txPoolConfig, blockchain)
	pool.Init(txPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	key, _ := crypto.GenerateKey()
	testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000))

	tx := pricedTransaction(0, 100000, big.NewInt(2), key)
	pool.SetGasTip(big.NewInt(tx.GasPrice().Int64() + 1))

	if err := pool.Add([]*types.Transaction{tx}, true)[0]; !errors.Is(err, txpool.ErrTxGasPriceTooLow) {
		t.Fatalf("Min tip not enforced")
	}

	tx = dynamicFeeTx(0, 100000, big.NewInt(3), big.NewInt(2), key)
	pool.SetGasTip(big.NewInt(tx.GasTipCap().Int64() + 1))

	if err := pool.Add([]*types.Transaction{tx}, true)[0]; !errors.Is(err, txpool.ErrTxGasPriceTooLow) {
		t.Fatalf("Min tip not enforced")
	}
}

// Tests that setting the transaction pool gas price to a higher value correctly
// discards everything cheaper (legacy & dynamic fee) than that and moves any
// gapped transactions back from the pending pool to the queue.
//
// Note, local transactions are never allowed to be dropped.
func TestRepricingDynamicFee(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	pool, _ := setupPoolWithConfig(eip1559Config)
	defer pool.Close()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan core.NewTxsEvent, 32)

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

	// Import the batch and that both pending and queued transactions match up
	pool.addRemotesSync(txs)

	pending, queued := pool.Stats()
	if pending != 6 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 6)
	}

	if queued != 3 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 3)
	}
	if err := validateEvents(events, 6); err != nil {
		t.Fatalf("original event firing failed: %v", err)
	}

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Reprice the pool and check that underpriced transactions get dropped
	pool.SetGasTip(big.NewInt(2))

	pending, queued = pool.Stats()
	if pending != 1 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 1)
	}

	if queued != 5 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 5)
	}

	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("reprice event firing failed: %v", err)
	}

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Check that we can't add the old transactions back
	tx := pricedTransaction(1, 100000, big.NewInt(1), keys[0])
	if err := pool.addRemote(tx); !errors.Is(err, txpool.ErrTxGasPriceTooLow) {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, txpool.ErrTxGasPriceTooLow)
	}

	tx = dynamicFeeTx(0, 100000, big.NewInt(2), big.NewInt(1), keys[1])
	if err := pool.addRemote(tx); !errors.Is(err, txpool.ErrTxGasPriceTooLow) {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, txpool.ErrTxGasPriceTooLow)
	}

	tx = dynamicFeeTx(2, 100000, big.NewInt(1), big.NewInt(1), keys[2])
	if err := pool.addRemote(tx); !errors.Is(err, txpool.ErrTxGasPriceTooLow) {
		t.Fatalf("adding underpriced queued transaction error mismatch: have %v, want %v", err, txpool.ErrTxGasPriceTooLow)
	}

	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("post-reprice event firing failed: %v", err)
	}

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// And we can fill gaps with properly priced transactions
	tx = pricedTransaction(1, 100000, big.NewInt(2), keys[0])
	if err := pool.addRemote(tx); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}

	tx = dynamicFeeTx(0, 100000, big.NewInt(3), big.NewInt(2), keys[1])
	if err := pool.addRemote(tx); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}

	tx = dynamicFeeTx(2, 100000, big.NewInt(2), big.NewInt(2), keys[2])
	if err := pool.addRemoteSync(tx); err != nil {
		t.Fatalf("failed to add queued transaction: %v", err)
	}

	if err := validateEvents(events, 5); err != nil {
		t.Fatalf("post-reprice event firing failed: %v", err)
	}

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that when the pool reaches its global transaction limit, underpriced
// transactions are gradually shifted out for more expensive ones and any gapped
// pending transactions are moved into the queue.
//
// Note, local transactions are never allowed to be dropped.
func TestUnderpricing(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	config := testTxPoolConfig
	config.GlobalSlots = 2
	config.GlobalQueue = 2

	pool := New(config, blockchain)
	pool.Init(config.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan core.NewTxsEvent, 32)

	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(10000000))
	}

	// Generate and queue a batch of transactions, both pending and queued
	txs := types.Transactions{}

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(1), keys[0])) // pending
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(2), keys[0])) // pending
	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(1), keys[2])) // pending

	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(1), keys[1])) // queued
	// Import the batch and that both pending and queued transactions match up
	pool.addRemotesSync(txs)

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

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Ensure that adding an underpriced transaction on block limit fails
	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(1), keys[1])); !errors.Is(err, txpool.ErrUnderpriced) {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, txpool.ErrUnderpriced)
	}
	// Replace a future transaction with a future transaction
	if err := pool.addRemoteSync(pricedTransaction(1, 100000, big.NewInt(5), keys[1])); err != nil { // +K1:1 => -K1:1 => Pend K0:0, K0:1, K2:0; Que K1:1
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	// Ensure that adding high priced transactions drops cheap ones, but not own
	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(3), keys[1])); err != nil { // +K1:0 => -K1:1 => Pend K0:0, K0:1, K1:0, K2:0; Que -
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(2, 100000, big.NewInt(4), keys[1])); err != nil { // +K1:2 => -K0:0 => Pend K1:0, K2:0; Que K0:1 K1:2
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(3, 100000, big.NewInt(5), keys[1])); err != nil { // +K1:3 => -K0:1 => Pend K1:0, K2:0; Que K1:2 K1:3
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	// Ensure that replacing a pending transaction with a future transaction fails
	if err := pool.addRemoteSync(pricedTransaction(5, 100000, big.NewInt(6), keys[1])); !errors.Is(err, ErrFutureReplacePending) {
		t.Fatalf("adding future replace transaction error mismatch: have %v, want %v", err, ErrFutureReplacePending)
	}

	pending, queued = pool.Stats()
	if pending != 4 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 4)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateEvents(events, 4); err != nil {
		t.Fatalf("additional event firing failed: %v", err)
	}

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that more expensive transactions push out cheap ones from the pool, but
// without producing instability by creating gaps that start jumping transactions
// back and forth between queued/pending.
func TestStableUnderpricing(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	config := testTxPoolConfig
	config.GlobalSlots = 128
	config.GlobalQueue = 0

	pool := New(config, blockchain)
	pool.Init(config.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan core.NewTxsEvent, 32)

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
	pool.addRemotesSync(txs)

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

	if err := validatePoolInternals(pool); err != nil {
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

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that when the pool reaches its global transaction limit, underpriced
// transactions (legacy & dynamic fee) are gradually shifted out for more
// expensive ones and any gapped pending transactions are moved into the queue.
func TestUnderpricingDynamicFee(t *testing.T) {
	t.Parallel()

	pool, _ := setupPoolWithConfig(eip1559Config)
	defer pool.Close()

	pool.config.GlobalSlots = 2
	pool.config.GlobalQueue = 2

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan core.NewTxsEvent, 32)

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

	txs = append(txs, dynamicFeeTx(0, 100000, big.NewInt(3), big.NewInt(2), keys[0])) // pending
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(2), keys[0]))           // pending
	txs = append(txs, dynamicFeeTx(1, 100000, big.NewInt(2), big.NewInt(1), keys[1])) // queued
	txs = append(txs, dynamicFeeTx(0, 100000, big.NewInt(2), big.NewInt(1), keys[2])) // pending

	// Import the batch and check that both pending and queued transactions match up
	pool.addRemotesSync(txs) // Pend K0:0, K0:1; Que K1:1

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

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Ensure that adding an underpriced transaction fails
	tx := dynamicFeeTx(0, 100000, big.NewInt(2), big.NewInt(1), keys[1])
	if err := pool.addRemoteSync(tx); !errors.Is(err, txpool.ErrUnderpriced) { // Pend K0:0, K0:1, K2:0; Que K1:1
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, txpool.ErrUnderpriced)
	}

	// Ensure that adding high priced transactions drops cheap ones, but not own
	tx = pricedTransaction(0, 100000, big.NewInt(2), keys[1])
	if err := pool.addRemoteSync(tx); err != nil { // +K1:0, -K1:1 => Pend K0:0, K0:1, K1:0, K2:0; Que -
		t.Fatalf("failed to add well priced transaction: %v", err)
	}

	tx = pricedTransaction(1, 100000, big.NewInt(3), keys[1])
	if err := pool.addRemoteSync(tx); err != nil { // +K1:2, -K0:1 => Pend K0:0 K1:0, K2:0; Que K1:2
		t.Fatalf("failed to add well priced transaction: %v", err)
	}

	tx = dynamicFeeTx(2, 100000, big.NewInt(4), big.NewInt(1), keys[1])
	if err := pool.addRemoteSync(tx); err != nil { // +K1:3, -K1:0 => Pend K0:0 K2:0; Que K1:2 K1:3
		t.Fatalf("failed to add well priced transaction: %v", err)
	}

	pending, queued = pool.Stats()
	if pending != 4 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 4)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validateEvents(events, 3); err != nil {
		t.Fatalf("additional event firing failed: %v", err)
	}

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests whether highest fee cap transaction is retained after a batch of high effective
// tip transactions are added and vice versa
func TestDualHeapEviction(t *testing.T) {
	t.Parallel()

	pool, _ := setupPoolWithConfig(eip1559Config)
	defer pool.Close()

	pool.config.GlobalSlots = 10
	pool.config.GlobalQueue = 10

	var (
		highTip, highCap *types.Transaction
		baseFee          int
	)

	check := func(tx *types.Transaction, name string) {
		if pool.all.Get(tx.Hash()) == nil {
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
			pool.addRemotesSync([]*types.Transaction{tx})
		}
		pending, queued := pool.Stats()
		if pending+queued != 20 {
			t.Fatalf("transaction count mismatch: have %d, want %d", pending+queued, 10)
		}
	}

	add(false)
	for baseFee = 0; baseFee <= 1000; baseFee += 100 {
		pool.priced.SetBaseFee(big.NewInt(int64(baseFee)))
		add(true)
		check(highCap, "fee cap")
		add(false)
		check(highTip, "effective tip")
	}

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that the pool rejects duplicate transactions.
func TestDeduplication(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	pool := New(testTxPoolConfig, blockchain)
	pool.Init(testTxPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

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
	errs := pool.addRemotesSync(firsts)
	if len(errs) != len(firsts) {
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
	errs = pool.addRemotesSync(txs)
	if len(errs) != len(txs) {
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

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that the pool rejects replacement transactions that don't meet the minimum
// price bump required.
func TestReplacement(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	pool := New(testTxPoolConfig, blockchain)
	pool.Init(testTxPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan core.NewTxsEvent, 32)

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
	if err := pool.addRemote(pricedTransaction(0, 100001, big.NewInt(1), key)); !errors.Is(err, txpool.ErrReplaceUnderpriced) {
		t.Fatalf("original cheap pending transaction replacement error mismatch: have %v, want %v", err, txpool.ErrReplaceUnderpriced)
	}
	if err := pool.addRemote(pricedTransaction(0, 100000, big.NewInt(2), key)); err != nil {
		t.Fatalf("failed to replace original cheap pending transaction: %v", err)
	}

	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("cheap replacement event firing failed: %v", err)
	}

	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(price), key)); err != nil {
		t.Fatalf("failed to add original proper pending transaction: %v", err)
	}
	if err := pool.addRemote(pricedTransaction(0, 100001, big.NewInt(threshold-1), key)); !errors.Is(err, txpool.ErrReplaceUnderpriced) {
		t.Fatalf("original proper pending transaction replacement error mismatch: have %v, want %v", err, txpool.ErrReplaceUnderpriced)
	}
	if err := pool.addRemote(pricedTransaction(0, 100000, big.NewInt(threshold), key)); err != nil {
		t.Fatalf("failed to replace original proper pending transaction: %v", err)
	}

	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("proper replacement event firing failed: %v", err)
	}

	// Add queued transactions, ensuring the minimum price bump is enforced for replacement (for ultra low prices too)
	if err := pool.addRemote(pricedTransaction(2, 100000, big.NewInt(1), key)); err != nil {
		t.Fatalf("failed to add original cheap queued transaction: %v", err)
	}
	if err := pool.addRemote(pricedTransaction(2, 100001, big.NewInt(1), key)); !errors.Is(err, txpool.ErrReplaceUnderpriced) {
		t.Fatalf("original cheap queued transaction replacement error mismatch: have %v, want %v", err, txpool.ErrReplaceUnderpriced)
	}
	if err := pool.addRemote(pricedTransaction(2, 100000, big.NewInt(2), key)); err != nil {
		t.Fatalf("failed to replace original cheap queued transaction: %v", err)
	}

	if err := pool.addRemote(pricedTransaction(2, 100000, big.NewInt(price), key)); err != nil {
		t.Fatalf("failed to add original proper queued transaction: %v", err)
	}
	if err := pool.addRemote(pricedTransaction(2, 100001, big.NewInt(threshold-1), key)); !errors.Is(err, txpool.ErrReplaceUnderpriced) {
		t.Fatalf("original proper queued transaction replacement error mismatch: have %v, want %v", err, txpool.ErrReplaceUnderpriced)
	}
	if err := pool.addRemote(pricedTransaction(2, 100000, big.NewInt(threshold), key)); err != nil {
		t.Fatalf("failed to replace original proper queued transaction: %v", err)
	}

	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("queued replacement event firing failed: %v", err)
	}

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that the pool rejects replacement dynamic fee transactions that don't
// meet the minimum price bump required.
func TestReplacementDynamicFee(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	pool, key := setupPoolWithConfig(eip1559Config)
	defer pool.Close()
	testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1000000000))

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan core.NewTxsEvent, 32)

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
		// into pending immediately. 2 nonce txs are "gapped"
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
		if err := pool.addRemote(tx); !errors.Is(err, txpool.ErrReplaceUnderpriced) {
			t.Fatalf("original cheap %s transaction replacement error mismatch: have %v, want %v", stage, err, txpool.ErrReplaceUnderpriced)
		}
		// 3.  Bump both more than min => accept
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(3), big.NewInt(2), key)
		if err := pool.addRemote(tx); err != nil {
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
		if err := pool.addRemote(tx); !errors.Is(err, txpool.ErrReplaceUnderpriced) {
			t.Fatalf("original proper %s transaction replacement error mismatch: have %v, want %v", stage, err, txpool.ErrReplaceUnderpriced)
		}
		// 7.  Bump fee cap max allowed so it's still underpriced => discard
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(feeCapThreshold-1), big.NewInt(gasTipCap), key)
		if err := pool.addRemote(tx); !errors.Is(err, txpool.ErrReplaceUnderpriced) {
			t.Fatalf("original proper %s transaction replacement error mismatch: have %v, want %v", stage, err, txpool.ErrReplaceUnderpriced)
		}
		// 8.  Bump tip min for acceptance => accept
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(gasFeeCap), big.NewInt(tipThreshold), key)
		if err := pool.addRemote(tx); !errors.Is(err, txpool.ErrReplaceUnderpriced) {
			t.Fatalf("original proper %s transaction replacement error mismatch: have %v, want %v", stage, err, txpool.ErrReplaceUnderpriced)
		}
		// 9.  Bump fee cap min for acceptance => accept
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(feeCapThreshold), big.NewInt(gasTipCap), key)
		if err := pool.addRemote(tx); !errors.Is(err, txpool.ErrReplaceUnderpriced) {
			t.Fatalf("original proper %s transaction replacement error mismatch: have %v, want %v", stage, err, txpool.ErrReplaceUnderpriced)
		}
		// 10. Check events match expected (3 new executable txs during pending, 0 during queue)
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(feeCapThreshold), big.NewInt(tipThreshold), key)
		if err := pool.addRemote(tx); err != nil {
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

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// TestStatusCheck tests that the pool can correctly retrieve the
// pending status of individual transactions.
func TestStatusCheck(t *testing.T) {
	t.Parallel()

	// Create the pool to test the status retrievals with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	pool := New(testTxPoolConfig, blockchain)
	pool.Init(testTxPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

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
	pool.addRemotesSync(txs)

	pending, queued := pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}

	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}

	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Retrieve the status of each transaction and validate them
	hashes := make([]common.Hash, len(txs))
	for i, tx := range txs {
		hashes[i] = tx.Hash()
	}

	hashes = append(hashes, common.Hash{})
	expect := []txpool.TxStatus{txpool.TxStatusPending, txpool.TxStatusPending, txpool.TxStatusQueued, txpool.TxStatusQueued, txpool.TxStatusUnknown}

	for i := 0; i < len(hashes); i++ {
		if status := pool.Status(hashes[i]); status != expect[i] {
			t.Errorf("transaction %d: status mismatch: have %v, want %v", i, status, expect[i])
		}
	}
}

// Test the transaction slots consumption is computed correctly
func TestSlotCount(t *testing.T) {
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

// TestSetCodeTransactions tests a few scenarios regarding the EIP-7702
// SetCodeTx.
func TestSetCodeTransactions(t *testing.T) {
	t.Parallel()

	// Create the pool to test the status retrievals with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(params.MergedTestChainConfig, 1000000, statedb, new(event.Feed))

	pool := New(testTxPoolConfig, blockchain)
	pool.Init(testTxPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	// Create the test accounts
	var (
		keyA, _ = crypto.GenerateKey()
		keyB, _ = crypto.GenerateKey()
		keyC, _ = crypto.GenerateKey()
		addrA   = crypto.PubkeyToAddress(keyA.PublicKey)
		addrB   = crypto.PubkeyToAddress(keyB.PublicKey)
		addrC   = crypto.PubkeyToAddress(keyC.PublicKey)
	)
	testAddBalance(pool, addrA, big.NewInt(params.Ether))
	testAddBalance(pool, addrB, big.NewInt(params.Ether))
	testAddBalance(pool, addrC, big.NewInt(params.Ether))

	for _, tt := range []struct {
		name    string
		pending int
		queued  int
		run     func(string)
	}{
		{
			// Check that only one in-flight transaction is allowed for accounts
			// with delegation set.
			name:    "accept-one-inflight-tx-of-delegated-account",
			pending: 1,
			run: func(name string) {
				aa := common.Address{0xaa, 0xaa}
				statedb.SetCode(addrA, append(types.DelegationPrefix, aa.Bytes()...))
				statedb.SetCode(aa, []byte{byte(vm.ADDRESS), byte(vm.PUSH0), byte(vm.SSTORE)})

				// Send gapped transaction, it should be rejected.
				if err := pool.addRemoteSync(pricedTransaction(2, 100000, big.NewInt(1), keyA)); !errors.Is(err, ErrOutOfOrderTxFromDelegated) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, ErrOutOfOrderTxFromDelegated, err)
				}
				// Send transactions. First is accepted, second is rejected.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(1), keyA)); err != nil {
					t.Fatalf("%s: failed to add remote transaction: %v", name, err)
				}
				// Second and further transactions shall be rejected
				if err := pool.addRemoteSync(pricedTransaction(1, 100000, big.NewInt(1), keyA)); !errors.Is(err, txpool.ErrInflightTxLimitReached) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, txpool.ErrInflightTxLimitReached, err)
				}
				// Check gapped transaction again.
				if err := pool.addRemoteSync(pricedTransaction(2, 100000, big.NewInt(1), keyA)); !errors.Is(err, txpool.ErrInflightTxLimitReached) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, txpool.ErrInflightTxLimitReached, err)
				}
				// Replace by fee.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(10), keyA)); err != nil {
					t.Fatalf("%s: failed to replace with remote transaction: %v", name, err)
				}

				// Reset the delegation, avoid leaking state into the other tests
				statedb.SetCode(addrA, nil)
			},
		},
		{
			// This test is analogous to the previous one, but the delegation is pending
			// instead of set.
			name:    "allow-one-tx-from-pooled-delegation",
			pending: 2,
			run: func(name string) {
				// Create a pending delegation request from B.
				if err := pool.addRemoteSync(setCodeTx(0, keyA, []unsignedAuth{{0, keyB}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				// First transaction from B is accepted.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(1), keyB)); err != nil {
					t.Fatalf("%s: failed to add remote transaction: %v", name, err)
				}
				// Second transaction fails due to limit.
				if err := pool.addRemoteSync(pricedTransaction(1, 100000, big.NewInt(1), keyB)); !errors.Is(err, txpool.ErrInflightTxLimitReached) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, txpool.ErrInflightTxLimitReached, err)
				}
				// Replace by fee for first transaction from B works.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(2), keyB)); err != nil {
					t.Fatalf("%s: failed to add remote transaction: %v", name, err)
				}
			},
		},
		{
			// This is the symmetric case of the previous one, where the delegation request
			// is received after the transaction. The resulting state shall be the same.
			name:    "accept-authorization-from-sender-of-one-inflight-tx",
			pending: 2,
			run: func(name string) {
				// The first in-flight transaction is accepted.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(1), keyB)); err != nil {
					t.Fatalf("%s: failed to add with pending delegation: %v", name, err)
				}
				// Delegation is accepted.
				if err := pool.addRemoteSync(setCodeTx(0, keyA, []unsignedAuth{{0, keyB}})); err != nil {
					t.Fatalf("%s: failed to add remote transaction: %v", name, err)
				}
				// The second in-flight transaction is rejected.
				if err := pool.addRemoteSync(pricedTransaction(1, 100000, big.NewInt(1), keyB)); !errors.Is(err, txpool.ErrInflightTxLimitReached) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, txpool.ErrInflightTxLimitReached, err)
				}
			},
		},
		{
			name:    "reject-authorization-from-sender-with-more-than-one-inflight-tx",
			pending: 2,
			run: func(name string) {
				// Submit two transactions.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(1), keyB)); err != nil {
					t.Fatalf("%s: failed to add with pending delegation: %v", name, err)
				}
				if err := pool.addRemoteSync(pricedTransaction(1, 100000, big.NewInt(1), keyB)); err != nil {
					t.Fatalf("%s: failed to add with pending delegation: %v", name, err)
				}
				// Delegation rejected since two txs are already in-flight.
				if err := pool.addRemoteSync(setCodeTx(0, keyA, []unsignedAuth{{0, keyB}})); !errors.Is(err, ErrAuthorityReserved) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, ErrAuthorityReserved, err)
				}
			},
		},
		{
			name:    "allow-setcode-tx-with-pending-authority-tx",
			pending: 2,
			run: func(name string) {
				// Send two transactions where the first has no conflicting delegations and
				// the second should be allowed despite conflicting with the authorities in the first.
				if err := pool.addRemoteSync(setCodeTx(0, keyA, []unsignedAuth{{1, keyC}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				if err := pool.addRemoteSync(setCodeTx(0, keyB, []unsignedAuth{{1, keyC}})); err != nil {
					t.Fatalf("%s: failed to add conflicting delegation: %v", name, err)
				}
			},
		},
		{
			name:    "replace-by-fee-setcode-tx",
			pending: 1,
			run: func(name string) {
				if err := pool.addRemoteSync(setCodeTx(0, keyB, []unsignedAuth{{1, keyC}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, uint256.NewInt(2000), uint256.NewInt(2), keyB, []unsignedAuth{{0, keyC}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
			},
		},
		{
			name:    "allow-more-than-one-tx-from-replaced-authority",
			pending: 3,
			run: func(name string) {
				// Send transaction from A with B as an authority.
				if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, uint256.NewInt(10), uint256.NewInt(3), keyA, []unsignedAuth{{0, keyB}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				// Replace transaction with another having C as an authority.
				if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, uint256.NewInt(3000), uint256.NewInt(300), keyA, []unsignedAuth{{0, keyC}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				// B should not be considred as having an in-flight delegation, so
				// should allow more than one pooled transaction.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(10), keyB)); err != nil {
					t.Fatalf("%s: failed to replace with remote transaction: %v", name, err)
				}
				if err := pool.addRemoteSync(pricedTransaction(1, 100000, big.NewInt(10), keyB)); err != nil {
					t.Fatalf("%s: failed to replace with remote transaction: %v", name, err)
				}
			},
		},
		{
			// This test is analogous to the previous one, but the the replaced
			// transaction is self-sponsored.
			name:    "allow-tx-from-replaced-self-sponsor-authority",
			pending: 3,
			run: func(name string) {
				// Send transaction from A with A as an authority.
				if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, uint256.NewInt(10), uint256.NewInt(3), keyA, []unsignedAuth{{0, keyA}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				// Replace transaction with a transaction with B as an authority.
				if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, uint256.NewInt(30), uint256.NewInt(30), keyA, []unsignedAuth{{0, keyB}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				// The one in-flight transaction limit from A no longer applies, so we
				// can stack a second transaction for the account.
				if err := pool.addRemoteSync(pricedTransaction(1, 100000, big.NewInt(1000), keyA)); err != nil {
					t.Fatalf("%s: failed to replace with remote transaction: %v", name, err)
				}
				// B should still be able to send transactions.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(1000), keyB)); err != nil {
					t.Fatalf("%s: failed to replace with remote transaction: %v", name, err)
				}
				// However B still has the limitation to one in-flight transaction.
				if err := pool.addRemoteSync(pricedTransaction(1, 100000, big.NewInt(1), keyB)); !errors.Is(err, txpool.ErrInflightTxLimitReached) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, txpool.ErrInflightTxLimitReached, err)
				}
			},
		},
		{
			name:    "replacements-respect-inflight-tx-count",
			pending: 2,
			run: func(name string) {
				// Send transaction from A with B as an authority.
				if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, uint256.NewInt(10), uint256.NewInt(3), keyA, []unsignedAuth{{0, keyB}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				// Send two transactions from B. Only the first should be accepted due
				// to in-flight limit.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(1), keyB)); err != nil {
					t.Fatalf("%s: failed to add remote transaction: %v", name, err)
				}
				if err := pool.addRemoteSync(pricedTransaction(1, 100000, big.NewInt(1), keyB)); !errors.Is(err, txpool.ErrInflightTxLimitReached) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, txpool.ErrInflightTxLimitReached, err)
				}
				// Replace the in-flight transaction from B.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(30), keyB)); err != nil {
					t.Fatalf("%s: failed to replace with remote transaction: %v", name, err)
				}
				// Ensure the in-flight limit for B is still in place.
				if err := pool.addRemoteSync(pricedTransaction(1, 100000, big.NewInt(1), keyB)); !errors.Is(err, txpool.ErrInflightTxLimitReached) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, txpool.ErrInflightTxLimitReached, err)
				}
			},
		},
		{
			// Since multiple authorizations can be pending simultaneously, replacing
			// one of them should not break the one in-flight-transaction limit.
			name:    "track-multiple-conflicting-delegations",
			pending: 3,
			run: func(name string) {
				// Send two setcode txs both with C as an authority.
				if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, uint256.NewInt(10), uint256.NewInt(3), keyA, []unsignedAuth{{0, keyC}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, uint256.NewInt(30), uint256.NewInt(30), keyB, []unsignedAuth{{0, keyC}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				// Replace the tx from A with a non-setcode tx.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(1000), keyA)); err != nil {
					t.Fatalf("%s: failed to replace with remote transaction: %v", name, err)
				}
				// Make sure we can only pool one tx from keyC since it is still a
				// pending authority.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(1000), keyC)); err != nil {
					t.Fatalf("%s: failed to added single pooled for account with pending delegation: %v", name, err)
				}
				if err, want := pool.addRemoteSync(pricedTransaction(1, 100000, big.NewInt(1000), keyC)), txpool.ErrInflightTxLimitReached; !errors.Is(err, want) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, want, err)
				}
			},
		},
		{
			name:    "remove-hash-from-authority-tracker",
			pending: 10,
			run: func(name string) {
				var keys []*ecdsa.PrivateKey
				for i := 0; i < 30; i++ {
					key, _ := crypto.GenerateKey()
					keys = append(keys, key)
					addr := crypto.PubkeyToAddress(key.PublicKey)
					testAddBalance(pool, addr, big.NewInt(params.Ether))
				}
				// Create a transactions with 3 unique auths so the lookup's auth map is
				// filled with addresses.
				for i := 0; i < 30; i += 3 {
					if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, uint256.NewInt(10), uint256.NewInt(3), keys[i], []unsignedAuth{{0, keys[i]}, {0, keys[i+1]}, {0, keys[i+2]}})); err != nil {
						t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
					}
				}
				// Replace one of the transactions with a normal transaction so that the
				// original hash is removed from the tracker. The hash should be
				// associated with 3 different authorities.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(1000), keys[0])); err != nil {
					t.Fatalf("%s: failed to replace with remote transaction: %v", name, err)
				}
			},
		},
	} {
		tt.run(tt.name)
		pending, queued := pool.Stats()
		if pending != tt.pending {
			t.Fatalf("%s: pending transactions mismatched: have %d, want %d", tt.name, pending, tt.pending)
		}
		if queued != tt.queued {
			t.Fatalf("%s: queued transactions mismatched: have %d, want %d", tt.name, queued, tt.queued)
		}
		if err := validatePoolInternals(pool); err != nil {
			t.Fatalf("%s: pool internal state corrupted: %v", tt.name, err)
		}
		pool.Clear()
	}
}

func TestSetCodeTransactionsReorg(t *testing.T) {
	t.Parallel()

	// Create the pool to test the status retrievals with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(params.MergedTestChainConfig, 1000000, statedb, new(event.Feed))

	pool := New(testTxPoolConfig, blockchain)
	pool.Init(testTxPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	// Create the test accounts
	var (
		keyA, _ = crypto.GenerateKey()
		addrA   = crypto.PubkeyToAddress(keyA.PublicKey)
	)
	testAddBalance(pool, addrA, big.NewInt(params.Ether))
	// Send an authorization for 0x42
	var authList []types.SetCodeAuthorization
	auth, _ := types.SignSetCode(keyA, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(params.TestChainConfig.ChainID),
		Address: common.Address{0x42},
		Nonce:   0,
	})
	authList = append(authList, auth)
	if err := pool.addRemoteSync(pricedSetCodeTxWithAuth(0, 250000, uint256.NewInt(10), uint256.NewInt(3), keyA, authList)); err != nil {
		t.Fatalf("failed to add with remote setcode transaction: %v", err)
	}
	// Simulate the chain moving
	blockchain.statedb.SetNonce(addrA, 1, tracing.NonceChangeAuthorization)
	blockchain.statedb.SetCode(addrA, types.AddressToDelegation(auth.Address))
	<-pool.requestReset(nil, nil)
	// Set an authorization for 0x00
	auth, _ = types.SignSetCode(keyA, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(params.TestChainConfig.ChainID),
		Address: common.Address{},
		Nonce:   0,
	})
	authList = append(authList, auth)
	if err := pool.addRemoteSync(pricedSetCodeTxWithAuth(1, 250000, uint256.NewInt(10), uint256.NewInt(3), keyA, authList)); err != nil {
		t.Fatalf("failed to add with remote setcode transaction: %v", err)
	}
	// Try to add a transactions in
	if err := pool.addRemoteSync(pricedTransaction(2, 100000, big.NewInt(1000), keyA)); !errors.Is(err, txpool.ErrInflightTxLimitReached) {
		t.Fatalf("unexpected error %v, expecting %v", err, txpool.ErrInflightTxLimitReached)
	}
	// Simulate the chain moving
	blockchain.statedb.SetNonce(addrA, 2, tracing.NonceChangeAuthorization)
	blockchain.statedb.SetCode(addrA, nil)
	<-pool.requestReset(nil, nil)
	// Now send two transactions from addrA
	if err := pool.addRemoteSync(pricedTransaction(2, 100000, big.NewInt(1000), keyA)); err != nil {
		t.Fatalf("failed to added single transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(3, 100000, big.NewInt(1000), keyA)); err != nil {
		t.Fatalf("failed to added single transaction: %v", err)
	}
}

// Benchmarks the speed of validating the contents of the pending queue of the
// transaction pool.
func BenchmarkPendingDemotion100(b *testing.B)   { benchmarkPendingDemotion(b, 100) }
func BenchmarkPendingDemotion1000(b *testing.B)  { benchmarkPendingDemotion(b, 1000) }
func BenchmarkPendingDemotion10000(b *testing.B) { benchmarkPendingDemotion(b, 10000) }

func benchmarkPendingDemotion(b *testing.B, size int) {
	// Add a batch of transactions to a pool one by one
	pool, key := setupPool()
	defer pool.Close()

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
	pool, key := setupPool()
	defer pool.Close()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000))

	for i := 0; i < size; i++ {
		tx := transaction(uint64(1+i), 100000, key)
		pool.enqueueTx(tx.Hash(), tx, true)
	}
	// Benchmark the speed of pool validation
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pool.promoteExecutables(nil)
	}
}

// Benchmarks the speed of batched transaction insertion.
func BenchmarkBatchInsert100(b *testing.B)   { benchmarkBatchInsert(b, 100) }
func BenchmarkBatchInsert1000(b *testing.B)  { benchmarkBatchInsert(b, 1000) }
func BenchmarkBatchInsert10000(b *testing.B) { benchmarkBatchInsert(b, 10000) }

func benchmarkBatchInsert(b *testing.B, size int) {
	// Generate a batch of transactions to enqueue into the pool
	pool, key := setupPool()
	defer pool.Close()

	account := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, account, big.NewInt(1000000000000000000))

	batches := make([]types.Transactions, b.N)
	for i := 0; i < b.N; i++ {
		batches[i] = make(types.Transactions, size)
		for j := 0; j < size; j++ {
			batches[i][j] = transaction(uint64(size*i+j), 100000, key)
		}
	}
	// Benchmark importing the transactions into the queue
	b.ResetTimer()
	for _, batch := range batches {
		pool.addRemotes(batch)
	}
}

// Benchmarks the speed of batch transaction insertion in case of multiple accounts.
func BenchmarkPoolAccountMultiBatchInsert(b *testing.B) {
	// Generate a batch of transactions to enqueue into the pool
	pool, _ := setupPool()
	defer pool.Close()

	batches := make(types.Transactions, b.N)

	for i := 0; i < b.N; i++ {
		key, _ := crypto.GenerateKey()
		account := crypto.PubkeyToAddress(key.PublicKey)

		pool.currentState.AddBalance(account, uint256.NewInt(1000000), tracing.BalanceChangeTransfer)

		tx := transaction(uint64(0), 100000, key)

		batches[i] = tx
	}

	// Benchmark importing the transactions into the queue
	b.ReportAllocs()
	b.ResetTimer()

	for _, tx := range batches {
		pool.addRemotesSync([]*types.Transaction{tx})
	}
}

func BenchmarkPoolAccountMultiBatchInsertRace(b *testing.B) {
	// Generate a batch of transactions to enqueue into the pool
	pool, _ := setupPool()
	defer pool.Close()

	batches := make(types.Transactions, b.N)

	for i := 0; i < b.N; i++ {
		key, _ := crypto.GenerateKey()
		account := crypto.PubkeyToAddress(key.PublicKey)
		tx := transaction(uint64(0), 100000, key)

		pool.currentState.AddBalance(account, uint256.NewInt(1000000), tracing.BalanceChangeTransfer)

		batches[i] = tx
	}

	done := make(chan struct{})

	go func() {
		t := time.NewTicker(time.Microsecond)
		defer t.Stop()

		var pending map[common.Address][]*txpool.LazyTransaction

	loop:
		for {
			select {
			case <-t.C:
				pending = pool.Pending(txpool.PendingFilter{})
			case <-done:
				break loop
			}
		}

		fmt.Fprint(io.Discard, pending)
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for _, tx := range batches {
		pool.addRemotesSync([]*types.Transaction{tx})
	}

	close(done)
}

func MakeWithPromoteTxCh(ch chan struct{}) func(*LegacyPool) {
	return func(pool *LegacyPool) {
		pool.promoteTxCh = ch
	}
}

func BenchmarkPoolAccountMultiBatchInsertNoLockRace(b *testing.B) {
	// Generate a batch of transactions to enqueue into the pool
	pendingAddedCh := make(chan struct{}, 1024)

	pool, localKey := setupPoolWithConfig(params.TestChainConfig, MakeWithPromoteTxCh(pendingAddedCh))
	defer pool.Close()

	_ = localKey

	batches := make(types.Transactions, b.N)

	for i := 0; i < b.N; i++ {
		key, _ := crypto.GenerateKey()
		account := crypto.PubkeyToAddress(key.PublicKey)
		tx := transaction(uint64(0), 100000, key)

		pool.currentState.AddBalance(account, uint256.NewInt(1000000), tracing.BalanceChangeTransfer)

		batches[i] = tx
	}

	done := make(chan struct{})

	go func() {
		t := time.NewTicker(time.Microsecond)
		defer t.Stop()

		var pending map[common.Address][]*txpool.LazyTransaction

		for range t.C {
			pending = pool.Pending(txpool.PendingFilter{})

			if len(pending) >= b.N/2 {
				close(done)

				return
			}
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for _, tx := range batches {
		pool.addRemotes([]*types.Transaction{tx})
	}

	<-done
}

func BenchmarkPoolAccountsBatchInsert(b *testing.B) {
	// Generate a batch of transactions to enqueue into the pool
	pool, _ := setupPool()
	defer pool.Close()

	batches := make(types.Transactions, b.N)

	for i := 0; i < b.N; i++ {
		key, _ := crypto.GenerateKey()
		account := crypto.PubkeyToAddress(key.PublicKey)

		pool.currentState.AddBalance(account, uint256.NewInt(1000000), tracing.BalanceChangeTransfer)

		tx := transaction(uint64(0), 100000, key)

		batches[i] = tx
	}

	// Benchmark importing the transactions into the queue
	b.ReportAllocs()
	b.ResetTimer()

	for _, tx := range batches {
		_ = pool.addRemoteSync(tx)
	}
}

func BenchmarkPoolAccountsBatchInsertRace(b *testing.B) {
	// Generate a batch of transactions to enqueue into the pool
	pool, _ := setupPool()
	defer pool.Close()

	batches := make(types.Transactions, b.N)

	for i := 0; i < b.N; i++ {
		key, _ := crypto.GenerateKey()
		account := crypto.PubkeyToAddress(key.PublicKey)
		tx := transaction(uint64(0), 100000, key)

		pool.currentState.AddBalance(account, uint256.NewInt(1000000), tracing.BalanceChangeTransfer)

		batches[i] = tx
	}

	done := make(chan struct{})

	go func() {
		t := time.NewTicker(time.Microsecond)
		defer t.Stop()

		var pending map[common.Address][]*txpool.LazyTransaction

	loop:
		for {
			select {
			case <-t.C:
				pending = pool.Pending(txpool.PendingFilter{})
			case <-done:
				break loop
			}
		}

		fmt.Fprint(io.Discard, pending)
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for _, tx := range batches {
		_ = pool.addRemoteSync(tx)
	}

	close(done)
}

func BenchmarkPoolAccountsBatchInsertNoLockRace(b *testing.B) {
	// Generate a batch of transactions to enqueue into the pool
	pendingAddedCh := make(chan struct{}, 1024)

	pool, localKey := setupPoolWithConfig(params.TestChainConfig, MakeWithPromoteTxCh(pendingAddedCh))
	defer pool.Close()

	_ = localKey

	batches := make(types.Transactions, b.N)

	for i := 0; i < b.N; i++ {
		key, _ := crypto.GenerateKey()
		account := crypto.PubkeyToAddress(key.PublicKey)
		tx := transaction(uint64(0), 100000, key)

		pool.currentState.AddBalance(account, uint256.NewInt(1000000), tracing.BalanceChangeTransfer)

		batches[i] = tx
	}

	done := make(chan struct{})

	go func() {
		t := time.NewTicker(time.Microsecond)
		defer t.Stop()

		var pending map[common.Address][]*txpool.LazyTransaction

		for range t.C {
			pending = pool.Pending(txpool.PendingFilter{})

			if len(pending) >= b.N/2 {
				close(done)

				return
			}
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for _, tx := range batches {
		_ = pool.addRemote(tx)
	}

	<-done
}

func TestPoolMultiAccountBatchInsertRace(t *testing.T) {
	t.Parallel()

	// Generate a batch of transactions to enqueue into the pool
	pool, _ := setupPool()
	defer pool.Close()

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
			pending map[common.Address][]*txpool.LazyTransaction
			total   int
		)

		for range t.C {
			pending = pool.Pending(txpool.PendingFilter{})
			total = len(pending)

			if total >= n {
				close(done)

				return
			}
		}
	}()

	for _, tx := range batches {
		pool.addRemotesSync([]*types.Transaction{tx})
	}

	for _, tx := range batchesSecond {
		pool.addRemotes([]*types.Transaction{tx})
	}

	<-done
}

func newTxs(pool *LegacyPool) *types.Transaction {
	key, _ := crypto.GenerateKey()
	account := crypto.PubkeyToAddress(key.PublicKey)
	tx := transaction(uint64(0), 100000, key)

	pool.currentState.AddBalance(account, uint256.NewInt(1_000_000_000), tracing.BalanceChangeTransfer)

	return tx
}

// type acc struct {
// 	nonce   uint64
// 	key     *ecdsa.PrivateKey
// 	account common.Address
// }

// type testTx struct {
// 	tx      *types.Transaction
// 	idx     int
// 	isLocal bool
// }

// const localIdx = 0

// func getTransactionGen(t *rapid.T, keys []*acc, nonces []uint64, localKey *acc, gasPriceMin, gasPriceMax, gasLimitMin, gasLimitMax uint64) *testTx {
// 	idx := rapid.IntRange(0, len(keys)-1).Draw(t, "accIdx").(int)

// 	var (
// 		isLocal bool
// 		key     *ecdsa.PrivateKey
// 	)

// 	if idx == localIdx {
// 		isLocal = true
// 		key = localKey.key
// 	} else {
// 		key = keys[idx].key
// 	}

// 	nonces[idx]++

// 	gasPriceUint := rapid.Uint64Range(gasPriceMin, gasPriceMax).Draw(t, "gasPrice").(uint64)
// 	gasPrice := big.NewInt(0).SetUint64(gasPriceUint)
// 	gasLimit := rapid.Uint64Range(gasLimitMin, gasLimitMax).Draw(t, "gasLimit").(uint64)

// 	return &testTx{
// 		tx:      pricedTransaction(nonces[idx]-1, gasLimit, gasPrice, key),
// 		idx:     idx,
// 		isLocal: isLocal,
// 	}
// }

// type transactionBatches struct {
// 	txs      []*testTx
// 	totalTxs int
// }

// func transactionsGen(keys []*acc, nonces []uint64, localKey *acc, minTxs int, maxTxs int, gasPriceMin, gasPriceMax, gasLimitMin, gasLimitMax uint64, caseParams *strings.Builder) func(t *rapid.T) *transactionBatches {
// 	return func(t *rapid.T) *transactionBatches {
// 		totalTxs := rapid.IntRange(minTxs, maxTxs).Draw(t, "totalTxs").(int)
// 		txs := make([]*testTx, totalTxs)

// 		gasValues := make([]float64, totalTxs)

// 		fmt.Fprintf(caseParams, " totalTxs = %d;", totalTxs)

// 		keys = keys[:len(nonces)]

// 		for i := 0; i < totalTxs; i++ {
// 			txs[i] = getTransactionGen(t, keys, nonces, localKey, gasPriceMin, gasPriceMax, gasLimitMin, gasLimitMax)

// 			gasValues[i] = float64(txs[i].tx.Gas())
// 		}

// 		mean, stddev := stat.MeanStdDev(gasValues, nil)
// 		fmt.Fprintf(caseParams, " gasValues mean %d, stdev %d, %d-%d);", int64(mean), int64(stddev), int64(floats.Min(gasValues)), int64(floats.Max(gasValues)))

// 		return &transactionBatches{txs, totalTxs}
// 	}
// }

// type txPoolRapidConfig struct {
// 	gasLimit    uint64
// 	avgBlockTxs uint64

// 	minTxs int
// 	maxTxs int

// 	minAccs int
// 	maxAccs int

// 	// less tweakable, more like constants
// 	gasPriceMin uint64
// 	gasPriceMax uint64

// 	gasLimitMin uint64
// 	gasLimitMax uint64

// 	balance int64

// 	blockTime      time.Duration
// 	maxEmptyBlocks int
// 	maxStuckBlocks int
// }

// func defaultTxPoolRapidConfig() txPoolRapidConfig {
// 	gasLimit := uint64(30_000_000)
// 	avgBlockTxs := gasLimit/params.TxGas + 1
// 	maxTxs := int(25 * avgBlockTxs)

// 	return txPoolRapidConfig{
// 		gasLimit: gasLimit,

// 		avgBlockTxs: avgBlockTxs,

// 		minTxs: 1,
// 		maxTxs: maxTxs,

// 		minAccs: 1,
// 		maxAccs: maxTxs,

// 		// less tweakable, more like constants
// 		gasPriceMin: 1,
// 		gasPriceMax: 1_000,

// 		gasLimitMin: params.TxGas,
// 		gasLimitMax: gasLimit / 2,

// 		balance: 0xffffffffffffff,

// 		blockTime:      2 * time.Second,
// 		maxEmptyBlocks: 10,
// 		maxStuckBlocks: 10,
// 	}
// }

// TODO - Fix Later
// TestSmallTxPool is not something to run in parallel as far it uses all CPUs
// nolint:paralleltest
// func TestSmallTxPool(t *testing.T) {
// 	t.Parallel()

// 	t.Skip("a red test to be fixed")

// 	cfg := defaultTxPoolRapidConfig()

// 	cfg.maxEmptyBlocks = 10
// 	cfg.maxStuckBlocks = 10

// 	cfg.minTxs = 1
// 	cfg.maxTxs = 2

// 	cfg.minAccs = 1
// 	cfg.maxAccs = 2

// 	testPoolBatchInsert(t, cfg)
// }

// TODO - Fix Later
// // This test is not something to run in parallel as far it uses all CPUs
// // nolint:paralleltest
// func TestBigTxPool(t *testing.T) {
// 	t.Parallel()

// 	t.Skip("a red test to be fixed")

// 	cfg := defaultTxPoolRapidConfig()

// 	testPoolBatchInsert(t, cfg)
// }

//nolint:gocognit,thelper
// func testPoolBatchInsert(t *testing.T, cfg txPoolRapidConfig) {
// 	t.Helper()

// 	t.Parallel()

// 	const debug = false

// 	initialBalance := big.NewInt(cfg.balance)

// 	keys := make([]*acc, cfg.maxAccs)

// 	var key *ecdsa.PrivateKey

// 	// prealloc keys
// 	for idx := 0; idx < cfg.maxAccs; idx++ {
// 		key, _ = crypto.GenerateKey()

// 		keys[idx] = &acc{
// 			key:     key,
// 			nonce:   0,
// 			account: crypto.PubkeyToAddress(key.PublicKey),
// 		}
// 	}

// 	var threads = runtime.NumCPU()

// 	if debug {
// 		// 1 is set only for debug
// 		threads = 1
// 	}

// 	testsDone := new(uint64)

// 	for i := 0; i < threads; i++ {
// 		t.Run(fmt.Sprintf("thread %d", i), func(t *testing.T) {
// 			t.Parallel()

// 			rapid.Check(t, func(rt *rapid.T) {
// 				caseParams := new(strings.Builder)

// 				defer func() {
// 					res := atomic.AddUint64(testsDone, 1)

// 					if res%100 == 0 {
// 						fmt.Println("case-done", res)
// 					}
// 				}()

// 				// Generate a batch of transactions to enqueue into the pool
// 				testTxPoolConfig := testTxPoolConfig

// 				// from sentry config
// 				testTxPoolConfig.AccountQueue = 16
// 				testTxPoolConfig.AccountSlots = 16
// 				testTxPoolConfig.GlobalQueue = 32768
// 				testTxPoolConfig.GlobalSlots = 32768
// 				testTxPoolConfig.Lifetime = time.Hour + 30*time.Minute //"1h30m0s"
// 				testTxPoolConfig.PriceLimit = 1

// 				now := time.Now()
// 				pendingAddedCh := make(chan struct{}, 1024)

// 				pool, key := setupPoolWithConfig(params.TestChainConfig)
// 				defer pool.Close()

// 				totalAccs := rapid.IntRange(cfg.minAccs, cfg.maxAccs).Draw(rt, "totalAccs").(int)

// 				fmt.Fprintf(caseParams, "Case params: totalAccs = %d;", totalAccs)

// 				defer func() {
// 					pending, queued := pool.Content()

// 					if len(pending) != 0 {
// 						pendingGas := make([]float64, 0, len(pending))

// 						for _, txs := range pending {
// 							for _, tx := range txs {
// 								pendingGas = append(pendingGas, float64(tx.Gas()))
// 							}
// 						}

// 						mean, stddev := stat.MeanStdDev(pendingGas, nil)
// 						fmt.Fprintf(caseParams, "\tpending mean %d, stdev %d, %d-%d;\n", int64(mean), int64(stddev), int64(floats.Min(pendingGas)), int64(floats.Max(pendingGas)))
// 					}

// 					if len(queued) != 0 {
// 						queuedGas := make([]float64, 0, len(queued))

// 						for _, txs := range queued {
// 							for _, tx := range txs {
// 								queuedGas = append(queuedGas, float64(tx.Gas()))
// 							}
// 						}

// 						mean, stddev := stat.MeanStdDev(queuedGas, nil)
// 						fmt.Fprintf(caseParams, "\tqueued mean %d, stdev %d, %d-%d);\n\n", int64(mean), int64(stddev), int64(floats.Min(queuedGas)), int64(floats.Max(queuedGas)))
// 					}

// 					rt.Log(caseParams)
// 				}()

// 				// regenerate only local key
// 				localKey := &acc{
// 					key:     key,
// 					account: crypto.PubkeyToAddress(key.PublicKey),
// 				}

// 				if err := validatePoolInternals(pool); err != nil {
// 					rt.Fatalf("pool internal state corrupted: %v", err)
// 				}

// 				var wg sync.WaitGroup

// 				wg.Add(1)

// 				go func() {
// 					defer wg.Done()

// 					now = time.Now()

// 					testAddBalance(pool, localKey.account, initialBalance)

// 					for idx := 0; idx < totalAccs; idx++ {
// 						testAddBalance(pool, keys[idx].account, initialBalance)
// 					}
// 				}()

// 				nonces := make([]uint64, totalAccs)
// 				gen := rapid.Custom(transactionsGen(keys, nonces, localKey, cfg.minTxs, cfg.maxTxs, cfg.gasPriceMin, cfg.gasPriceMax, cfg.gasLimitMin, cfg.gasLimitMax, caseParams))

// 				txs := gen.Draw(rt, "batches").(*transactionBatches)

// 				wg.Wait()

// 				var (
// 					addIntoTxPool func(tx *types.Transaction) error
// 					totalInBatch  int
// 				)

// 				for _, tx := range txs.txs {
// 					addIntoTxPool = pool.addRemoteSync

// 					if tx.isLocal {
// 						addIntoTxPool = pool.addLocal
// 					}

// 					err := addIntoTxPool(tx.tx)
// 					if err != nil {
// 						rt.Log("on adding a transaction to the tx pool", err, tx.tx.Gas(), tx.tx.GasPrice(), tx.tx.GasPrice(), getBalance(pool, keys[tx.idx].account))
// 					}
// 				}

// 				var (
// 					block              int
// 					emptyBlocks        int
// 					stuckBlocks        int
// 					lastTxPoolStats    int
// 					currentTxPoolStats int
// 				)

// 				for {
// 					// we'd expect fulfilling block take comparable, but less than blockTime
// 					ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.maxStuckBlocks)*cfg.blockTime)

// 					select {
// 					case <-pendingAddedCh:
// 					case <-ctx.Done():
// 						pendingStat, queuedStat := pool.Stats()
// 						if pendingStat+queuedStat == 0 {
// 							cancel()

// 							break
// 						}

// 						rt.Fatalf("got %ds block timeout (expected less then %s): total accounts %d. Pending %d, queued %d)",
// 							block, 5*cfg.blockTime, txs.totalTxs, pendingStat, queuedStat)
// 					}

// 					pendingStat, queuedStat := pool.Stats()

// 					currentTxPoolStats = pendingStat + queuedStat
// 					if currentTxPoolStats == 0 {
// 						cancel()
// 						break
// 					}

// 					// check if txPool got stuck
// 					if currentTxPoolStats == lastTxPoolStats {
// 						stuckBlocks++ //todo: need something better then that
// 					} else {
// 						stuckBlocks = 0
// 						lastTxPoolStats = currentTxPoolStats
// 					}

// 					// copy-paste
// 					start := time.Now()
// 					pending := pool.Pending(true)
// 					locals := pool.Locals()

// 					// from fillTransactions
// 					removedFromPool, blockGasLeft, err := fillTransactions(ctx, pool, locals, pending, cfg.gasLimit)

// 					done := time.Since(start)

// 					if removedFromPool > 0 {
// 						emptyBlocks = 0
// 					} else {
// 						emptyBlocks++
// 					}

// 					if emptyBlocks >= cfg.maxEmptyBlocks || stuckBlocks >= cfg.maxStuckBlocks {
// 						// check for nonce gaps
// 						var lastNonce, currentNonce int

// 						pending = pool.Pending(true)

// 						for txAcc, pendingTxs := range pending {
// 							lastNonce = int(pool.Nonce(txAcc)) - len(pendingTxs) - 1

// 							isFirst := true

// 							for _, tx := range pendingTxs {
// 								currentNonce = int(tx.Nonce())
// 								if currentNonce-lastNonce != 1 {
// 									rt.Fatalf("got a nonce gap for account %q. Current pending nonce %d, previous %d %v; emptyBlocks - %v; stuckBlocks - %v",
// 										txAcc, currentNonce, lastNonce, isFirst, emptyBlocks >= cfg.maxEmptyBlocks, stuckBlocks >= cfg.maxStuckBlocks)
// 								}

// 								lastNonce = currentNonce
// 							}
// 						}
// 					}

// 					if emptyBlocks >= cfg.maxEmptyBlocks {
// 						rt.Fatalf("got %d empty blocks in a row(expected less then %d): total time %s, total accounts %d. Pending %d, locals %d)",
// 							emptyBlocks, cfg.maxEmptyBlocks, done, txs.totalTxs, len(pending), len(locals))
// 					}

// 					if stuckBlocks >= cfg.maxStuckBlocks {
// 						rt.Fatalf("got %d empty blocks in a row(expected less then %d): total time %s, total accounts %d. Pending %d, locals %d)",
// 							emptyBlocks, cfg.maxEmptyBlocks, done, txs.totalTxs, len(pending), len(locals))
// 					}

// 					if err != nil {
// 						rt.Fatalf("took too long: total time %s(expected %s), total accounts %d. Pending %d, locals %d)",
// 							done, cfg.blockTime, txs.totalTxs, len(pending), len(locals))
// 					}

// 					rt.Log("current_total", txs.totalTxs, "in_batch", totalInBatch, "removed", removedFromPool, "emptyBlocks", emptyBlocks, "blockGasLeft", blockGasLeft, "pending", len(pending), "locals", len(locals),
// 						"locals+pending", done)

// 					rt.Log("block", block, "pending", pendingStat, "queued", queuedStat, "elapsed", done)

// 					block++

// 					cancel()
// 					// time.Sleep(time.Second)
// 				}

// 				rt.Logf("case completed totalTxs %d %v\n\n", txs.totalTxs, time.Since(now))
// 			})
// 		})
// 	}

// 	t.Log("done test cases", atomic.LoadUint64(testsDone))
// }

// func fillTransactions(ctx context.Context, pool *LegacyPool, locals []common.Address, pending map[common.Address][]*txpool.LazyTransaction, gasLimit uint64) (int, uint64, error) {
// 	localTxs := make(map[common.Address]types.Transactions)
// 	remoteTxs := pending

// 	for _, txAcc := range locals {
// 		if txs := remoteTxs[txAcc]; len(txs) > 0 {
// 			delete(remoteTxs, txAcc)

// 			localTxs[txAcc] = txs
// 		}
// 	}

// 	// fake signer
// 	signer := types.NewLondonSigner(big.NewInt(1))

// 	// fake baseFee
// 	baseFee := uint256.NewInt(1)

// 	blockGasLimit := gasLimit

// 	var (
// 		txLocalCount  int
// 		txRemoteCount int
// 	)

// 	if len(localTxs) > 0 {
// 		txs := types.NewTransactionsByPriceAndNonce(signer, localTxs, baseFee)

// 		select {
// 		case <-ctx.Done():
// 			return txLocalCount + txRemoteCount, blockGasLimit, ctx.Err()
// 		default:
// 		}

// 		blockGasLimit, txLocalCount = commitTransactions(pool, txs, blockGasLimit)
// 	}

// 	select {
// 	case <-ctx.Done():
// 		return txLocalCount + txRemoteCount, blockGasLimit, ctx.Err()
// 	default:
// 	}

// 	if len(remoteTxs) > 0 {
// 		txs := types.NewTransactionsByPriceAndNonce(signer, remoteTxs, baseFee)

// 		select {
// 		case <-ctx.Done():
// 			return txLocalCount + txRemoteCount, blockGasLimit, ctx.Err()
// 		default:
// 		}

// 		blockGasLimit, txRemoteCount = commitTransactions(pool, txs, blockGasLimit)
// 	}

// 	return txLocalCount + txRemoteCount, blockGasLimit, nil
// }

// func commitTransactions(pool *LegacyPool, txs *types.TransactionsByPriceAndNonce, blockGasLimit uint64) (uint64, int) {
// 	var (
// 		tx      *types.Transaction
// 		txCount int
// 	)

// 	for {
// 		tx = txs.Peek()

// 		if tx == nil {
// 			return blockGasLimit, txCount
// 		}

// 		if tx.Gas() <= blockGasLimit {
// 			blockGasLimit -= tx.Gas()

// 			pool.mu.Lock()
// 			pool.removeTx(tx.Hash(), false, false)
// 			pool.mu.Unlock()

// 			txCount++
// 		} else {
// 			// we don't maximize fulfillment of the block. just fill somehow
// 			return blockGasLimit, txCount
// 		}
// 	}
// }

// func MakeWithPromoteTxCh(ch chan struct{}) func(*LegacyPool) {
// 	return func(pool *LegacyPool) {
// 		pool.promoteTxCh = ch
// 	}
// }

func BenchmarkBigs(b *testing.B) {
	// max 256-bit
	max := new(big.Int)
	max.Exp(big.NewInt(2), big.NewInt(256), nil).Sub(max, big.NewInt(1))

	ints := make([]*big.Int, 1000000)
	intUs := make([]*uint256.Int, 1000000)

	var over bool

	for i := 0; i < len(ints); i++ {
		ints[i] = crand2.BigInt(max)
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
// func mining(tb testing.TB, pool *LegacyPool, signer types.Signer, baseFee *uint256.Int, blockGasLimit uint64, totalBlocks int) (int, time.Duration, time.Duration) {
// 	var (
// 		localTxsCount  int
// 		remoteTxsCount int
// 		localTxs       = make(map[common.Address]types.Transactions)
// 		remoteTxs      map[common.Address]types.Transactions
// 		total          int
// 	)

// 	start := time.Now()

// 	pending := pool.Pending(true)

// 	pendingDuration := time.Since(start)

// 	remoteTxs = pending

// 	locals := pool.Locals()

// 	pendingLen, queuedLen := pool.Stats()

// 	for _, account := range locals {
// 		if txs := remoteTxs[account]; len(txs) > 0 {
// 			delete(remoteTxs, account)

// 			localTxs[account] = txs
// 		}
// 	}

// 	localTxsCount = len(localTxs)
// 	remoteTxsCount = len(remoteTxs)

// 	var txLocalCount int

// 	if localTxsCount > 0 {
// 		txs := miner.newTransactionsByPriceAndNonce(signer, localTxs, baseFee)

// 		blockGasLimit, txLocalCount = commitTransactions(pool, txs, blockGasLimit)

// 		total += txLocalCount
// 	}

// 	var txRemoteCount int

// 	if remoteTxsCount > 0 {
// 		txs := types.NewTransactionsByPriceAndNonce(signer, remoteTxs, baseFee)

// 		_, txRemoteCount = commitTransactions(pool, txs, blockGasLimit)

// 		total += txRemoteCount
// 	}

// 	miningDuration := time.Since(start)

// 	tb.Logf("[%s] mining block. block %d. total %d: pending %d(added %d), local %d(added %d), queued %d, localTxsCount %d, remoteTxsCount %d, pending %v, mining %v",
// 		common.NowMilliseconds(), totalBlocks, total, pendingLen, txRemoteCount, localTxsCount, txLocalCount, queuedLen, localTxsCount, remoteTxsCount, pendingDuration, miningDuration)

// 	return total, pendingDuration, miningDuration
// }

// TODO - Fix Later
//nolint:paralleltest
// func TestPoolMiningDataRaces(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("only for data race testing")
// 	}

// 	const format = "size %d, txs ticker %v, api ticker %v"

// 	cases := []struct {
// 		name              string
// 		size              int
// 		txsTickerDuration time.Duration
// 		apiTickerDuration time.Duration
// 	}{
// 		{
// 			size:              1,
// 			txsTickerDuration: 200 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 		{
// 			size:              1,
// 			txsTickerDuration: 400 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 		{
// 			size:              1,
// 			txsTickerDuration: 600 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 		{
// 			size:              1,
// 			txsTickerDuration: 800 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},

// 		{
// 			size:              5,
// 			txsTickerDuration: 200 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 		{
// 			size:              5,
// 			txsTickerDuration: 400 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 		{
// 			size:              5,
// 			txsTickerDuration: 600 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 		{
// 			size:              5,
// 			txsTickerDuration: 800 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},

// 		{
// 			size:              10,
// 			txsTickerDuration: 200 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 		{
// 			size:              10,
// 			txsTickerDuration: 400 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 		{
// 			size:              10,
// 			txsTickerDuration: 600 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 		{
// 			size:              10,
// 			txsTickerDuration: 800 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},

// 		{
// 			size:              20,
// 			txsTickerDuration: 200 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 		{
// 			size:              20,
// 			txsTickerDuration: 400 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 		{
// 			size:              20,
// 			txsTickerDuration: 600 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 		{
// 			size:              20,
// 			txsTickerDuration: 800 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},

// 		{
// 			size:              30,
// 			txsTickerDuration: 200 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 		{
// 			size:              30,
// 			txsTickerDuration: 400 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 		{
// 			size:              30,
// 			txsTickerDuration: 600 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 		{
// 			size:              30,
// 			txsTickerDuration: 800 * time.Millisecond,
// 			apiTickerDuration: 10 * time.Millisecond,
// 		},
// 	}

// 	for i := range cases {
// 		cases[i].name = fmt.Sprintf(format, cases[i].size, cases[i].txsTickerDuration, cases[i].apiTickerDuration)
// 	}

// 	//nolint:paralleltest
// 	for _, testCase := range cases {
// 		singleCase := testCase

// 		t.Run(singleCase.name, func(t *testing.T) {
// 			defer goleak.VerifyNone(t, leak.IgnoreList()...)

// 			const (
// 				blocks          = 300
// 				blockGasLimit   = 40_000_000
// 				blockPeriod     = time.Second
// 				threads         = 10
// 				batchesSize     = 10_000
// 				timeoutDuration = 10 * blockPeriod

// 				balanceStr = "1_000_000_000_000"
// 			)

// 			apiWithMining(t, balanceStr, batchesSize, singleCase, timeoutDuration, threads, blockPeriod, blocks, blockGasLimit)
// 		})
// 	}
// }

// //nolint:gocognit,thelper
// func apiWithMining(tb testing.TB, balanceStr string, batchesSize int, singleCase struct {
// 	name              string
// 	size              int
// 	txsTickerDuration time.Duration
// 	apiTickerDuration time.Duration
// }, timeoutDuration time.Duration, threads int, blockPeriod time.Duration, blocks int, blockGasLimit uint64) {
// 	done := make(chan struct{})

// 	var wg sync.WaitGroup

// 	defer func() {
// 		close(done)

// 		tb.Logf("[%s] finishing apiWithMining", common.NowMilliseconds())

// 		wg.Wait()

// 		tb.Logf("[%s] apiWithMining finished", common.NowMilliseconds())
// 	}()

// 	// Generate a batch of transactions to enqueue into the pool
// 	pendingAddedCh := make(chan struct{}, 1024)

// 	pool, localKey := setupPoolWithConfig(params.TestChainConfig, testTxPoolConfig, txPoolGasLimit, MakeWithPromoteTxCh(pendingAddedCh))
// 	defer pool.Close()

// 	localKeyPub := localKey.PublicKey
// 	account := crypto.PubkeyToAddress(localKeyPub)

// 	balance, ok := big.NewInt(0).SetString(balanceStr, 0)
// 	if !ok {
// 		tb.Fatal("incorrect initial balance", balanceStr)
// 	}

// 	testAddBalance(pool, account, balance)

// 	signer := types.NewEIP155Signer(big.NewInt(1))
// 	baseFee := uint256.NewInt(1)

// 	batchesLocal := make([]types.Transactions, batchesSize)
// 	batchesRemote := make([]types.Transactions, batchesSize)
// 	batchesRemotes := make([]types.Transactions, batchesSize)
// 	batchesRemoteSync := make([]types.Transactions, batchesSize)
// 	batchesRemotesSync := make([]types.Transactions, batchesSize)

// 	for i := 0; i < batchesSize; i++ {
// 		batchesLocal[i] = make(types.Transactions, singleCase.size)

// 		for j := 0; j < singleCase.size; j++ {
// 			batchesLocal[i][j] = pricedTransaction(uint64(singleCase.size*i+j), 100_000, big.NewInt(int64(i+1)), localKey)
// 		}

// 		batchesRemote[i] = make(types.Transactions, singleCase.size)

// 		remoteKey, _ := crypto.GenerateKey()
// 		remoteAddr := crypto.PubkeyToAddress(remoteKey.PublicKey)
// 		testAddBalance(pool, remoteAddr, balance)

// 		for j := 0; j < singleCase.size; j++ {
// 			batchesRemote[i][j] = pricedTransaction(uint64(j), 100_000, big.NewInt(int64(i+1)), remoteKey)
// 		}

// 		batchesRemotes[i] = make(types.Transactions, singleCase.size)

// 		remotesKey, _ := crypto.GenerateKey()
// 		remotesAddr := crypto.PubkeyToAddress(remotesKey.PublicKey)
// 		testAddBalance(pool, remotesAddr, balance)

// 		for j := 0; j < singleCase.size; j++ {
// 			batchesRemotes[i][j] = pricedTransaction(uint64(j), 100_000, big.NewInt(int64(i+1)), remotesKey)
// 		}

// 		batchesRemoteSync[i] = make(types.Transactions, singleCase.size)

// 		remoteSyncKey, _ := crypto.GenerateKey()
// 		remoteSyncAddr := crypto.PubkeyToAddress(remoteSyncKey.PublicKey)
// 		testAddBalance(pool, remoteSyncAddr, balance)

// 		for j := 0; j < singleCase.size; j++ {
// 			batchesRemoteSync[i][j] = pricedTransaction(uint64(j), 100_000, big.NewInt(int64(i+1)), remoteSyncKey)
// 		}

// 		batchesRemotesSync[i] = make(types.Transactions, singleCase.size)

// 		remotesSyncKey, _ := crypto.GenerateKey()
// 		remotesSyncAddr := crypto.PubkeyToAddress(remotesSyncKey.PublicKey)
// 		testAddBalance(pool, remotesSyncAddr, balance)

// 		for j := 0; j < singleCase.size; j++ {
// 			batchesRemotesSync[i][j] = pricedTransaction(uint64(j), 100_000, big.NewInt(int64(i+1)), remotesSyncKey)
// 		}
// 	}

// 	tb.Logf("[%s] starting goroutines", common.NowMilliseconds())

// 	txsTickerDuration := singleCase.txsTickerDuration
// 	apiTickerDuration := singleCase.apiTickerDuration

// 	// locals
// 	wg.Add(1)

// 	go func() {
// 		defer func() {
// 			tb.Logf("[%s] stopping addLocal(s)", common.NowMilliseconds())

// 			wg.Done()

// 			tb.Logf("[%s] stopped addLocal(s)", common.NowMilliseconds())
// 		}()

// 		tb.Logf("[%s] starting addLocal(s)", common.NowMilliseconds())

// 		for _, batch := range batchesLocal {
// 			batch := batch

// 			select {
// 			case <-done:
// 				return
// 			default:
// 			}

// 			if rand.Int()%2 == 0 {
// 				runWithTimeout(tb, func(_ chan struct{}) {
// 					errs := pool.addLocals(batch)
// 					if len(errs) != 0 {
// 						tb.Logf("[%s] addLocals error, %v", common.NowMilliseconds(), errs)
// 					}
// 				}, done, "addLocals", timeoutDuration, 0, 0)
// 			} else {
// 				for _, tx := range batch {
// 					tx := tx

// 					runWithTimeout(tb, func(_ chan struct{}) {
// 						err := pool.addLocal(tx)
// 						if err != nil {
// 							tb.Logf("[%s] addLocal error %s", common.NowMilliseconds(), err)
// 						}
// 					}, done, "addLocal", timeoutDuration, 0, 0)

// 					time.Sleep(txsTickerDuration)
// 				}
// 			}

// 			time.Sleep(txsTickerDuration)
// 		}
// 	}()

// 	// remotes
// 	wg.Add(1)

// 	go func() {
// 		defer func() {
// 			tb.Logf("[%s] stopping addRemotes", common.NowMilliseconds())

// 			wg.Done()

// 			tb.Logf("[%s] stopped addRemotes", common.NowMilliseconds())
// 		}()

// 		addTransactionsBatches(tb, batchesRemotes, getFnForBatches(pool.addRemotes), done, timeoutDuration, txsTickerDuration, "addRemotes", 0)
// 	}()

// 	// remote
// 	wg.Add(1)

// 	go func() {
// 		defer func() {
// 			tb.Logf("[%s] stopping addRemote", common.NowMilliseconds())

// 			wg.Done()

// 			tb.Logf("[%s] stopped addRemote", common.NowMilliseconds())
// 		}()

// 		addTransactions(tb, batchesRemote, pool.addRemote, done, timeoutDuration, txsTickerDuration, "addRemote", 0)
// 	}()

// 	// sync
// 	// remotes
// 	wg.Add(1)

// 	go func() {
// 		defer func() {
// 			tb.Logf("[%s] stopping addRemotesSync", common.NowMilliseconds())

// 			wg.Done()

// 			tb.Logf("[%s] stopped addRemotesSync", common.NowMilliseconds())
// 		}()

// 		addTransactionsBatches(tb, batchesRemotesSync, getFnForBatches(pool.addRemotesSync), done, timeoutDuration, txsTickerDuration, "addRemotesSync", 0)
// 	}()

// 	// remote
// 	wg.Add(1)

// 	go func() {
// 		defer func() {
// 			tb.Logf("[%s] stopping addRemoteSync", common.NowMilliseconds())

// 			wg.Done()

// 			tb.Logf("[%s] stopped addRemoteSync", common.NowMilliseconds())
// 		}()

// 		addTransactions(tb, batchesRemoteSync, pool.addRemoteSync, done, timeoutDuration, txsTickerDuration, "addRemoteSync", 0)
// 	}()

// 	// tx pool API
// 	for i := 0; i < threads; i++ {
// 		i := i

// 		wg.Add(1)

// 		go func() {
// 			defer func() {
// 				tb.Logf("[%s] stopping Pending-no-tips, thread %d", common.NowMilliseconds(), i)

// 				wg.Done()

// 				tb.Logf("[%s] stopped Pending-no-tips, thread %d", common.NowMilliseconds(), i)
// 			}()

// 			runWithTicker(tb, func(_ chan struct{}) {
// 				p := pool.Pending(false)
// 				fmt.Fprint(io.Discard, p)
// 			}, done, "Pending-no-tips", apiTickerDuration, timeoutDuration, i)
// 		}()

// 		wg.Add(1)

// 		go func() {
// 			defer func() {
// 				tb.Logf("[%s] stopping Pending-with-tips, thread %d", common.NowMilliseconds(), i)

// 				wg.Done()

// 				tb.Logf("[%s] stopped Pending-with-tips, thread %d", common.NowMilliseconds(), i)
// 			}()

// 			runWithTicker(tb, func(_ chan struct{}) {
// 				p := pool.Pending(true)
// 				fmt.Fprint(io.Discard, p)
// 			}, done, "Pending-with-tips", apiTickerDuration, timeoutDuration, i)
// 		}()

// 		wg.Add(1)

// 		go func() {
// 			defer func() {
// 				tb.Logf("[%s] stopping Locals, thread %d", common.NowMilliseconds(), i)

// 				wg.Done()

// 				tb.Logf("[%s] stopped Locals, thread %d", common.NowMilliseconds(), i)
// 			}()

// 			runWithTicker(tb, func(_ chan struct{}) {
// 				l := pool.Locals()
// 				fmt.Fprint(io.Discard, l)
// 			}, done, "Locals", apiTickerDuration, timeoutDuration, i)
// 		}()

// 		wg.Add(1)

// 		go func() {
// 			defer func() {
// 				tb.Logf("[%s] stopping Content, thread %d", common.NowMilliseconds(), i)

// 				wg.Done()

// 				tb.Logf("[%s] stopped Content, thread %d", common.NowMilliseconds(), i)
// 			}()

// 			runWithTicker(tb, func(_ chan struct{}) {
// 				p, q := pool.Content()
// 				fmt.Fprint(io.Discard, p, q)
// 			}, done, "Content", apiTickerDuration, timeoutDuration, i)
// 		}()

// 		wg.Add(1)

// 		go func() {
// 			defer func() {
// 				tb.Logf("[%s] stopping GasPriceUint256, thread %d", common.NowMilliseconds(), i)

// 				wg.Done()

// 				tb.Logf("[%s] stopped GasPriceUint256, thread %d", common.NowMilliseconds(), i)
// 			}()

// 			runWithTicker(tb, func(_ chan struct{}) {
// 				res := pool.GasPriceUint256()
// 				fmt.Fprint(io.Discard, res)
// 			}, done, "GasPriceUint256", apiTickerDuration, timeoutDuration, i)
// 		}()

// 		wg.Add(1)

// 		go func() {
// 			defer func() {
// 				tb.Logf("[%s] stopping GasPrice, thread %d", common.NowMilliseconds(), i)

// 				wg.Done()

// 				tb.Logf("[%s] stopped GasPrice, thread %d", common.NowMilliseconds(), i)
// 			}()

// 			runWithTicker(tb, func(_ chan struct{}) {
// 				res := pool.GasPrice()
// 				fmt.Fprint(io.Discard, res)
// 			}, done, "GasPrice", apiTickerDuration, timeoutDuration, i)
// 		}()

// 		wg.Add(1)

// 		go func() {
// 			defer func() {
// 				tb.Logf("[%s] stopping SetGasPrice, thread %d", common.NowMilliseconds(), i)

// 				wg.Done()

// 				tb.Logf("[%s] stopped SetGasPrice, , thread %d", common.NowMilliseconds(), i)
// 			}()

// 			runWithTicker(tb, func(_ chan struct{}) {
// 				pool.SetGasPrice(pool.GasPrice())
// 			}, done, "SetGasPrice", apiTickerDuration, timeoutDuration, i)
// 		}()

// 		wg.Add(1)

// 		go func() {
// 			defer func() {
// 				tb.Logf("[%s] stopping ContentFrom, thread %d", common.NowMilliseconds(), i)

// 				wg.Done()

// 				tb.Logf("[%s] stopped ContentFrom, thread %d", common.NowMilliseconds(), i)
// 			}()

// 			runWithTicker(tb, func(_ chan struct{}) {
// 				p, q := pool.ContentFrom(account)
// 				fmt.Fprint(io.Discard, p, q)
// 			}, done, "ContentFrom", apiTickerDuration, timeoutDuration, i)
// 		}()

// 		wg.Add(1)

// 		go func() {
// 			defer func() {
// 				tb.Logf("[%s] stopping Has, thread %d", common.NowMilliseconds(), i)

// 				wg.Done()

// 				tb.Logf("[%s] stopped Has, thread %d", common.NowMilliseconds(), i)
// 			}()

// 			runWithTicker(tb, func(_ chan struct{}) {
// 				res := pool.Has(batchesRemotes[0][0].Hash())
// 				fmt.Fprint(io.Discard, res)
// 			}, done, "Has", apiTickerDuration, timeoutDuration, i)
// 		}()

// 		wg.Add(1)

// 		go func() {
// 			defer func() {
// 				tb.Logf("[%s] stopping Get, thread %d", common.NowMilliseconds(), i)

// 				wg.Done()

// 				tb.Logf("[%s] stopped Get, thread %d", common.NowMilliseconds(), i)
// 			}()

// 			runWithTicker(tb, func(_ chan struct{}) {
// 				tx := pool.Get(batchesRemotes[0][0].Hash())
// 				fmt.Fprint(io.Discard, tx == nil)
// 			}, done, "Get", apiTickerDuration, timeoutDuration, i)
// 		}()

// 		wg.Add(1)

// 		go func() {
// 			defer func() {
// 				tb.Logf("[%s] stopping Nonce, thread %d", common.NowMilliseconds(), i)

// 				wg.Done()

// 				tb.Logf("[%s] stopped Nonce, thread %d", common.NowMilliseconds(), i)
// 			}()

// 			runWithTicker(tb, func(_ chan struct{}) {
// 				res := pool.Nonce(account)
// 				fmt.Fprint(io.Discard, res)
// 			}, done, "Nonce", apiTickerDuration, timeoutDuration, i)
// 		}()

// 		wg.Add(1)

// 		go func() {
// 			defer func() {
// 				tb.Logf("[%s] stopping Stats, thread %d", common.NowMilliseconds(), i)

// 				wg.Done()

// 				tb.Logf("[%s] stopped Stats, thread %d", common.NowMilliseconds(), i)
// 			}()

// 			runWithTicker(tb, func(_ chan struct{}) {
// 				p, q := pool.Stats()
// 				fmt.Fprint(io.Discard, p, q)
// 			}, done, "Stats", apiTickerDuration, timeoutDuration, i)
// 		}()

// 		wg.Add(1)

// 		go func() {
// 			defer func() {
// 				tb.Logf("[%s] stopping Status, thread %d", common.NowMilliseconds(), i)

// 				wg.Done()

// 				tb.Logf("[%s] stopped Status, thread %d", common.NowMilliseconds(), i)
// 			}()

// 			runWithTicker(tb, func(_ chan struct{}) {
// 				st := pool.Status([]common.Hash{batchesRemotes[1][0].Hash()})
// 				fmt.Fprint(io.Discard, st)
// 			}, done, "Status", apiTickerDuration, timeoutDuration, i)
// 		}()

// 		wg.Add(1)

// 		go func() {
// 			defer func() {
// 				tb.Logf("[%s] stopping SubscribeNewTxsEvent, thread %d", common.NowMilliseconds(), i)

// 				wg.Done()

// 				tb.Logf("[%s] stopped SubscribeNewTxsEvent, thread %d", common.NowMilliseconds(), i)
// 			}()

// 			runWithTicker(tb, func(c chan struct{}) {
// 				ch := make(chan core.NewTxsEvent, 10)
// 				sub := pool.SubscribeNewTxsEvent(ch)

// 				if sub == nil {
// 					return
// 				}

// 				defer sub.Unsubscribe()

// 				select {
// 				case <-done:
// 					return
// 				case <-c:
// 				case res := <-ch:
// 					fmt.Fprint(io.Discard, res)
// 				}
// 			}, done, "SubscribeNewTxsEvent", apiTickerDuration, timeoutDuration, i)
// 		}()
// 	}

// 	// wait for the start
// 	tb.Logf("[%s] before the first propagated transaction", common.NowMilliseconds())
// 	<-pendingAddedCh
// 	tb.Logf("[%s] after the first propagated transaction", common.NowMilliseconds())

// 	var (
// 		totalTxs    int
// 		totalBlocks int
// 	)

// 	pendingDurations := make([]time.Duration, 0, blocks)

// 	var (
// 		added           int
// 		pendingDuration time.Duration
// 		miningDuration  time.Duration
// 		diff            time.Duration
// 	)

// 	for {
// 		added, pendingDuration, miningDuration = mining(tb, pool, signer, baseFee, blockGasLimit, totalBlocks)

// 		totalTxs += added

// 		pendingDurations = append(pendingDurations, pendingDuration)

// 		totalBlocks++

// 		if totalBlocks > blocks {
// 			fmt.Fprint(io.Discard, totalTxs)
// 			break
// 		}

// 		diff = blockPeriod - miningDuration
// 		if diff > 0 {
// 			time.Sleep(diff)
// 		}
// 	}

// 	pendingDurationsFloat := make([]float64, len(pendingDurations))

// 	for i, v := range pendingDurations {
// 		pendingDurationsFloat[i] = float64(v.Nanoseconds())
// 	}

// 	mean, stddev := stat.MeanStdDev(pendingDurationsFloat, nil)
// 	tb.Logf("[%s] pending mean %v, stddev %v, %v-%v",
// 		common.NowMilliseconds(), time.Duration(mean), time.Duration(stddev), time.Duration(floats.Min(pendingDurationsFloat)), time.Duration(floats.Max(pendingDurationsFloat)))
// }

// func addTransactionsBatches(tb testing.TB, batches []types.Transactions, fn func(types.Transactions) error, done chan struct{}, timeoutDuration time.Duration, tickerDuration time.Duration, name string, thread int) {
// 	tb.Helper()

// 	tb.Logf("[%s] starting %s", common.NowMilliseconds(), name)

// 	defer func() {
// 		tb.Logf("[%s] stop %s", common.NowMilliseconds(), name)
// 	}()

// 	for _, batch := range batches {
// 		batch := batch

// 		select {
// 		case <-done:
// 			return
// 		default:
// 		}

// 		runWithTimeout(tb, func(_ chan struct{}) {
// 			err := fn(batch)
// 			if err != nil {
// 				tb.Logf("[%s] %s error: %s", common.NowMilliseconds(), name, err)
// 			}
// 		}, done, name, timeoutDuration, 0, thread)

// 		time.Sleep(tickerDuration)
// 	}
// }

// func addTransactions(tb testing.TB, batches []types.Transactions, fn func(*types.Transaction) error, done chan struct{}, timeoutDuration time.Duration, tickerDuration time.Duration, name string, thread int) {
// 	tb.Helper()

// 	tb.Logf("[%s] starting %s", common.NowMilliseconds(), name)

// 	defer func() {
// 		tb.Logf("[%s] stop %s", common.NowMilliseconds(), name)
// 	}()

// 	for _, batch := range batches {
// 		for _, tx := range batch {
// 			tx := tx

// 			select {
// 			case <-done:
// 				return
// 			default:
// 			}

// 			runWithTimeout(tb, func(_ chan struct{}) {
// 				err := fn(tx)
// 				if err != nil {
// 					tb.Logf("%s error: %s", name, err)
// 				}
// 			}, done, name, timeoutDuration, 0, thread)

// 			time.Sleep(tickerDuration)
// 		}

// 		time.Sleep(tickerDuration)
// 	}
// }

// func getFnForBatches(fn func([]*types.Transaction) []error) func(types.Transactions) error {
// 	return func(batch types.Transactions) error {
// 		errs := fn(batch)
// 		if len(errs) != 0 {
// 			return errs[0]
// 		}

// 		return nil
// 	}
// }

//nolint:unparam
// func runWithTicker(tb testing.TB, fn func(c chan struct{}), done chan struct{}, name string, tickerDuration, timeoutDuration time.Duration, thread int) {
// 	tb.Helper()

// 	select {
// 	case <-done:
// 		tb.Logf("[%s] Short path. finishing outer runWithTicker for %q, thread %d", common.NowMilliseconds(), name, thread)

// 		return
// 	default:
// 	}

// 	defer func() {
// 		tb.Logf("[%s] finishing outer runWithTicker for %q, thread %d", common.NowMilliseconds(), name, thread)
// 	}()

// 	localTicker := time.NewTicker(tickerDuration)
// 	defer localTicker.Stop()

// 	n := 0

// 	for range localTicker.C {
// 		select {
// 		case <-done:
// 			return
// 		default:
// 		}

// 		runWithTimeout(tb, fn, done, name, timeoutDuration, n, thread)

// 		n++
// 	}
// }

// func runWithTimeout(tb testing.TB, fn func(chan struct{}), outerDone chan struct{}, name string, timeoutDuration time.Duration, n, thread int) {
// 	tb.Helper()

// 	select {
// 	case <-outerDone:
// 		tb.Logf("[%s] Short path. exiting inner runWithTimeout by outer exit event for %q, thread %d, iteration %d", common.NowMilliseconds(), name, thread, n)

// 		return
// 	default:
// 	}

// 	timeout := time.NewTimer(timeoutDuration)
// 	defer timeout.Stop()

// 	doneCh := make(chan struct{})

// 	isError := new(int32)
// 	*isError = 0

// 	go func() {
// 		defer close(doneCh)

// 		select {
// 		case <-outerDone:
// 			return
// 		default:
// 			fn(doneCh)
// 		}
// 	}()

// 	const isDebug = false

// 	var stack string

// 	select {
// 	case <-outerDone:
// 		tb.Logf("[%s] exiting inner runWithTimeout by outer exit event for %q, thread %d, iteration %d", common.NowMilliseconds(), name, thread, n)
// 	case <-doneCh:
// 		// only for debug
// 		//tb.Logf("[%s] exiting inner runWithTimeout by successful call for %q, thread %d, iteration %d", common.NowMilliseconds(), name, thread, n)
// 	case <-timeout.C:
// 		atomic.StoreInt32(isError, 1)

// 		if isDebug {
// 			stack = string(debug.Stack(true))
// 		}

// 		tb.Errorf("[%s] %s timeouted, thread %d, iteration %d. Stack %s", common.NowMilliseconds(), name, thread, n, stack)
// 	}
// }

// Benchmarks the speed of batch transaction insertion in case of multiple accounts.
func BenchmarkMultiAccountBatchInsert(b *testing.B) {
	// Generate a batch of transactions to enqueue into the pool
	pool, _ := setupPool()
	defer pool.Close()
	b.ReportAllocs()
	batches := make(types.Transactions, b.N)

	for i := 0; i < b.N; i++ {
		key, _ := crypto.GenerateKey()
		account := crypto.PubkeyToAddress(key.PublicKey)
		pool.currentState.AddBalance(account, uint256.NewInt(1000000), tracing.BalanceChangeUnspecified)
		tx := transaction(uint64(0), 100000, key)
		batches[i] = tx
	}
	// Benchmark importing the transactions into the queue
	b.ResetTimer()

	for _, tx := range batches {
		pool.addRemotesSync([]*types.Transaction{tx})
	}
}
