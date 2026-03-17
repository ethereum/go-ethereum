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
	"math"
	"math/big"
	"math/rand"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/tracing"
	"github.com/XinFinOrg/XDPoSChain/core/txpool"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/trie"
	"github.com/holiman/uint256"
)

var (
	// testTxPoolConfig is a transaction pool configuration without stateful disk
	// sideeffects used during testing.
	testTxPoolConfig Config

	// eip1559Config is a chain config with EIP-1559 enabled at block 0.
	eip1559Config *params.ChainConfig
)

func init() {
	testTxPoolConfig = DefaultConfig
	testTxPoolConfig.Journal = ""

	cpy := *params.TestChainConfig
	eip1559Config = &cpy
	eip1559Config.BerlinBlock = common.Big0
	eip1559Config.Eip1559Block = common.Big0
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

func (bc *testBlockChain) Engine() consensus.Engine {
	return nil
}

func (bc *testBlockChain) GetHeader(common.Hash, uint64) *types.Header {
	return nil
}

func (bc *testBlockChain) CurrentHeader() *types.Header {
	return nil
}

func (bc *testBlockChain) Config() *params.ChainConfig {
	return bc.config
}

func (bc *testBlockChain) CurrentBlock() *types.Header {
	return &types.Header{
		Root:       types.EmptyRootHash,
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
	tx, _ := types.SignTx(types.NewTransaction(nonce, common.Address{}, big.NewInt(100), gaslimit, gasprice, nil), types.HomesteadSigner{}, key)
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
	return pricedSetCodeTx(nonce, 250000, uint256.MustFromBig(common.MinGasPrice), uint256.NewInt(1), key, unsigned)
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

func newReserver() *txpool.Reserver {
	return txpool.NewReservationTracker().NewHandle(42)
}

func setupPoolWithConfig(config *params.ChainConfig) (*LegacyPool, *ecdsa.PrivateKey) {
	diskdb := rawdb.NewMemoryDatabase()
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(diskdb))
	blockchain := newTestBlockChain(config, 10000000, statedb, new(event.Feed))

	key, _ := crypto.GenerateKey()
	pool := New(testTxPoolConfig, blockchain)
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
	priced, remote := pool.priced.urgent.Len()+pool.priced.floating.Len(), pool.all.RemoteCount()
	if priced != remote {
		return fmt.Errorf("total priced transaction count %d != %d", priced, remote)
	}
	// Ensure the next nonce to assign is the correct one
	for addr, txs := range pool.pending {
		// Find the last transaction
		var last uint64
		for nonce := range txs.txs.items {
			if last < nonce {
				last = nonce
			}
		}
		if nonce := pool.Nonce(addr); nonce != last+1 {
			return fmt.Errorf("pending nonce mismatch: have %v, want %v", nonce, last+1)
		}
		if txs.totalcost.Sign() < 0 {
			return fmt.Errorf("totalcost went negative: %v", txs.totalcost)
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

func TestPromoteSpecialTxUpdatesTotalCost(t *testing.T) {
	pool, key := setupPool()
	defer pool.Close()

	normal := transaction(0, 21000, key)
	addr, err := deriveSender(normal)
	if err != nil {
		t.Fatalf("failed to derive sender: %v", err)
	}
	special := setCodeTx(0, key, nil)

	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.pending[addr] = newList(true)
	list := pool.pending[addr]

	inserted, _ := list.Add(normal, pool.config.PriceBump)
	if !inserted {
		t.Fatal("failed to insert baseline transaction")
	}
	if _, err := pool.promoteSpecialTx(addr, special, false); err != nil {
		t.Fatalf("promoteSpecialTx failed: %v", err)
	}
	want, overflow := uint256.FromBig(special.Cost())
	if overflow {
		t.Fatal("special tx cost overflowed uint256 in test setup")
	}
	if list.totalcost.Cmp(want) != 0 {
		t.Fatalf("totalcost mismatch after special promotion: have %v want %v", list.totalcost, want)
	}

	// Removing the pending tx should not underflow totalcost.
	list.Forward(1)
	if list.totalcost.Sign() != 0 {
		t.Fatalf("totalcost should be zero after removal, have %v", list.totalcost)
	}
}

func TestListAddReplacementAvoidsIntermediateOverflow(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	max := new(big.Int).Sub(new(big.Int).Lsh(common.Big1, 256), common.Big1)
	oldPrice := new(big.Int).Sub(new(big.Int).Rsh(new(big.Int).Set(max), 1), big.NewInt(100))
	newPrice := new(big.Int).Add(oldPrice, common.Big1)

	oldTx, err := types.SignTx(types.NewTransaction(0, common.Address{}, common.Big0, 1, oldPrice, nil), types.HomesteadSigner{}, key)
	if err != nil {
		t.Fatalf("failed to sign old tx: %v", err)
	}
	newTx, err := types.SignTx(types.NewTransaction(0, common.Address{}, common.Big0, 1, newPrice, nil), types.HomesteadSigner{}, key)
	if err != nil {
		t.Fatalf("failed to sign replacement tx: %v", err)
	}

	list := newList(true)
	inserted, _ := list.Add(oldTx, 0)
	if !inserted {
		t.Fatal("failed to insert baseline transaction")
	}
	inserted, replaced := list.Add(newTx, 0)
	if !inserted {
		t.Fatal("replacement transaction should not overflow after subtracting old cost")
	}
	if replaced == nil || replaced.Hash() != oldTx.Hash() {
		t.Fatal("expected old transaction to be replaced")
	}
	want, overflow := uint256.FromBig(newTx.Cost())
	if overflow {
		t.Fatal("replacement tx cost overflowed uint256 in test setup")
	}
	if list.totalcost.Cmp(want) != 0 {
		t.Fatalf("totalcost mismatch after replacement: have %v want %v", list.totalcost, want)
	}
	if tx := list.txs.Get(newTx.Nonce()); tx == nil || tx.Hash() != newTx.Hash() {
		t.Fatal("replacement transaction was not stored in list")
	}
	list.Forward(1)
	if list.totalcost.Sign() != 0 {
		t.Fatalf("totalcost should be zero after removal, have %v", list.totalcost)
	}
}

func TestPromoteSpecialTxReplacementAvoidsIntermediateOverflow(t *testing.T) {
	pool, key := setupPool()
	defer pool.Close()

	max := new(big.Int).Sub(new(big.Int).Lsh(common.Big1, 256), common.Big1)
	oldPrice := new(big.Int).Sub(new(big.Int).Rsh(new(big.Int).Set(max), 1), big.NewInt(100))
	newPrice := new(big.Int).Add(oldPrice, common.Big1)
	oldTx, err := types.SignTx(types.NewTransaction(0, common.Address{}, common.Big0, 1, oldPrice, nil), types.HomesteadSigner{}, key)
	if err != nil {
		t.Fatalf("failed to sign old tx: %v", err)
	}
	addr, err := deriveSender(oldTx)
	if err != nil {
		t.Fatalf("failed to derive sender: %v", err)
	}
	special := pricedSetCodeTx(0, 1, uint256.MustFromBig(newPrice), uint256.NewInt(1), key, nil)

	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.pending[addr] = newList(true)
	list := pool.pending[addr]
	inserted, _ := list.Add(oldTx, 0)
	if !inserted {
		t.Fatal("failed to insert baseline transaction")
	}
	inserted, err = pool.promoteSpecialTx(addr, special, false)
	if err != nil {
		t.Fatalf("promoteSpecialTx failed: %v", err)
	}
	if !inserted {
		t.Fatal("special replacement should not overflow after subtracting old cost")
	}
	want, overflow := uint256.FromBig(special.Cost())
	if overflow {
		t.Fatal("special tx cost overflowed uint256 in test setup")
	}
	if list.totalcost.Cmp(want) != 0 {
		t.Fatalf("totalcost mismatch after special replacement: have %v want %v", list.totalcost, want)
	}
	if tx := list.txs.Get(special.Nonce()); tx == nil || tx.Hash() != special.Hash() {
		t.Fatal("special replacement transaction was not stored in list")
	}
	list.Forward(1)
	if list.totalcost.Sign() != 0 {
		t.Fatalf("totalcost should be zero after removal, have %v", list.totalcost)
	}
}

func TestPromoteSpecialTxOverflowReturnsErrorWithoutMutation(t *testing.T) {
	pool, key := setupPool()
	defer pool.Close()

	normal := transaction(0, 21000, key)
	addr, err := deriveSender(normal)
	if err != nil {
		t.Fatalf("failed to derive sender: %v", err)
	}
	max := new(big.Int).Sub(new(big.Int).Lsh(common.Big1, 256), common.Big1)
	special := pricedSetCodeTx(0, 2, uint256.MustFromBig(max), uint256.NewInt(1), key, nil)

	pool.mu.Lock()
	defer pool.mu.Unlock()

	inserted, err := pool.promoteSpecialTx(addr, special, false)
	if inserted {
		t.Fatal("overflowing special tx must not be inserted")
	}
	if !errors.Is(err, txpool.ErrSpecialTxCostOverflow) {
		t.Fatalf("wrong error: have %v, want %v", err, txpool.ErrSpecialTxCostOverflow)
	}
	if _, ok := pool.pending[addr]; ok {
		t.Fatal("pending list created for rejected special tx")
	}
	if pool.all.Get(special.Hash()) != nil {
		t.Fatal("rejected special tx should not be tracked in lookup")
	}
	if pool.pendingNonces.get(addr) != 0 {
		t.Fatalf("pending nonce changed for rejected special tx: have %d want 0", pool.pendingNonces.get(addr))
	}
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
		c.statedb, _ = state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
		// simulate that the new head block included tx0 and tx1
		c.statedb.SetNonce(c.address, 2)
		c.statedb.SetBalance(c.address, new(big.Int).SetUint64(params.Ether), tracing.BalanceChangeUnspecified)
		*c.trigger = false
	}
	return stdb, nil
}

// This test simulates a scenario where a new block is imported during a
// state reset and tests whether the pending state is in sync with the
// block head event that initiated the resetState().
func TestStateChangeDuringReset(t *testing.T) {
	t.Parallel()

	var (
		db         = rawdb.NewMemoryDatabase()
		key, _     = crypto.GenerateKey()
		address    = crypto.PubkeyToAddress(key.PublicKey)
		statedb, _ = state.New(types.EmptyRootHash, state.NewDatabase(db))
		trigger    = false
	)

	// setup pool with 2 transaction in it
	statedb.SetBalance(address, new(big.Int).SetUint64(params.Ether), tracing.BalanceChangeUnspecified)
	blockchain := &testChain{newTestBlockChain(params.TestChainConfig, 1000000000, statedb, new(event.Feed)), address, &trigger}

	tx0 := pricedTransaction(0, 100000, big.NewInt(250000000), key)
	tx1 := pricedTransaction(1, 100000, big.NewInt(250000000), key)

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
	pool.currentState.AddBalance(addr, amount, tracing.BalanceChangeUnspecified)
	pool.mu.Unlock()
}

func testSetNonce(pool *LegacyPool, addr common.Address, nonce uint64) {
	pool.mu.Lock()
	pool.currentState.SetNonce(addr, nonce)
	pool.mu.Unlock()
}

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

	// Test underpriced: set pool gasTip first, then create transaction with lower gas price
	// MinGasPrice is 250000000 (0.25 Gwei), so use gas price 300000000 (0.3 Gwei)
	// which is higher than MinGasPrice but lower than pool's gasTip (1 Gwei)
	pool.gasTip.Store(uint256.NewInt(1000000000)) // Set pool gasTip to 1 Gwei (1000000000)
	tx = pricedTransaction(1, 100000, big.NewInt(300000000), key)
	if err, want := pool.addRemote(tx), txpool.ErrUnderpriced; !errors.Is(err, want) {
		t.Errorf("want %v have %v", want, err)
	}
	// Local transactions should be accepted even if underpriced
	if err := pool.addLocal(tx); err != nil {
		t.Error("expected", nil, "got", err)
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

	pool.enqueueTx(tx.Hash(), tx, false, true)
	<-pool.requestPromoteExecutables(newAccountSet(pool.signer, from))
	if len(pool.pending) != 1 {
		t.Error("expected valid txs to be 1 is", len(pool.pending))
	}

	tx = transaction(1, 100, key)
	from, _ = deriveSender(tx)
	testSetNonce(pool, from, 2)
	pool.enqueueTx(tx.Hash(), tx, false, true)

	<-pool.requestPromoteExecutables(newAccountSet(pool.signer, from))
	if _, ok := pool.pending[from].txs.items[tx.Nonce()]; ok {
		t.Error("expected transaction to be in tx pool")
	}

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

	pool.enqueueTx(tx1.Hash(), tx1, false, true)
	pool.enqueueTx(tx2.Hash(), tx2, false, true)
	pool.enqueueTx(tx3.Hash(), tx3, false, true)

	pool.promoteExecutables([]common.Address{from})
	if len(pool.pending) != 1 {
		t.Error("expected pending length to be 1, got", len(pool.pending))
	}
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
	if err := pool.addRemote(tx); err != txpool.ErrNegativeValue {
		t.Error("expected", txpool.ErrNegativeValue, "got", err)
	}
}

// TestValidateTransactionEIP2681 tests that the pool correctly validates transactions
// according to EIP-2681, which limits the nonce to a maximum value of 2^64 - 1.
func TestValidateTransactionEIP2681(t *testing.T) {
	t.Parallel()

	pool, key := setupPool()
	defer pool.Close()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, addr, big.NewInt(2e18))

	to := common.HexToAddress("0x0000000000000000000000000000000000000001")

	tests := []struct {
		name     string
		nonce    uint64
		value    *big.Int
		gas      uint64
		gasPrice *big.Int
		wantErr  error
	}{
		{
			"nonce 0",
			0,
			big.NewInt(1e18),
			21000,
			big.NewInt(2e10),
			nil,
		},
		{
			"nonce 1",
			1,
			big.NewInt(1e18),
			21000,
			big.NewInt(2e10),
			nil,
		},
		{
			"EIP-2681 overflow",
			math.MaxUint64,
			big.NewInt(1e18),
			21000,
			big.NewInt(2e10),
			core.ErrNonceMax,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := types.NewTx(&types.LegacyTx{
				Nonce:    tt.nonce,
				To:       &to,
				Value:    tt.value,
				Gas:      tt.gas,
				GasPrice: tt.gasPrice,
			})
			signedTx, _ := types.SignTx(tx, types.HomesteadSigner{}, key)
			err := pool.validateTxBasics(signedTx, true)

			if tt.wantErr == nil && err != nil {
				t.Errorf("expected nil, got %v", err)
			} else if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestTipAboveFeeCap(t *testing.T) {
	t.Parallel()

	pool, key := setupPoolWithConfig(eip1559Config)
	defer pool.Close()

	tx := dynamicFeeTx(0, 100, big.NewInt(1), big.NewInt(2), key)

	if err := pool.addRemote(tx); err != core.ErrTipAboveFeeCap {
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
	if err := pool.addRemote(tx); err != core.ErrTipVeryHigh {
		t.Error("expected", core.ErrTipVeryHigh, "got", err)
	}

	tx2 := dynamicFeeTx(0, 100, veryBigNumber, big.NewInt(1), key)
	if err := pool.addRemote(tx2); err != core.ErrFeeCapVeryHigh {
		t.Error("expected", core.ErrFeeCapVeryHigh, "got", err)
	}
}

func TestChainFork(t *testing.T) {
	t.Parallel()

	pool, key := setupPool()
	defer pool.Close()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	resetState := func() {
		statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
		statedb.AddBalance(addr, big.NewInt(100000000000000), tracing.BalanceChangeUnspecified)

		pool.chain = newTestBlockChain(pool.chainconfig, 1000000, statedb, new(event.Feed))
		<-pool.requestReset(nil, nil)
	}
	resetState()

	tx := pricedTransaction(0, 100000, big.NewInt(300000000), key)
	if _, err := pool.add(tx, false); err != nil {
		t.Error("didn't expect error", err)
	}
	pool.removeTx(tx.Hash(), true, true)

	// reset the pool's internal state
	resetState()
	if _, err := pool.add(tx, false); err != nil {
		t.Error("didn't expect error", err)
	}
}

func TestDoubleNonce(t *testing.T) {
	t.Parallel()

	pool, key := setupPool()
	defer pool.Close()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	resetState := func() {
		statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
		statedb.AddBalance(addr, big.NewInt(600000000000000), tracing.BalanceChangeUnspecified)

		pool.chain = newTestBlockChain(pool.chainconfig, 1000000, statedb, new(event.Feed))
		<-pool.requestReset(nil, nil)
	}
	resetState()

	signer := types.HomesteadSigner{}
	// Gas prices are chosen to keep the original 1:2 price ratio (250,000,000 vs. 500,000,000 wei)
	// while satisfying the chain's MinGasPrice requirement of 250,000,000 wei.
	tx1, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(100), 100000, big.NewInt(250000000), nil), signer, key)
	tx2, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(100), 1000000, big.NewInt(500000000), nil), signer, key)
	tx3, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(100), 1000000, big.NewInt(250000000), nil), signer, key)

	// Add the first two transaction, ensure higher priced stays only
	if replace, err := pool.add(tx1, false); err != nil || replace {
		t.Errorf("first transaction insert failed (%v) or reported replacement (%v)", err, replace)
	}
	if replace, err := pool.add(tx2, false); err != nil || !replace {
		t.Errorf("second transaction insert failed (%v) or not reported replacement (%v)", err, replace)
	}
	<-pool.requestPromoteExecutables(newAccountSet(signer, addr))
	if pool.pending[addr].Len() != 1 {
		t.Error("expected 1 pending transactions, got", pool.pending[addr].Len())
	}
	if tx := pool.pending[addr].txs.items[0]; tx.Hash() != tx2.Hash() {
		t.Errorf("transaction mismatch: have %x, want %x", tx.Hash(), tx2.Hash())
	}

	// Add the third transaction and ensure it's not saved (smaller price)
	pool.add(tx3, false)
	<-pool.requestPromoteExecutables(newAccountSet(signer, addr))
	if pool.pending[addr].Len() != 1 {
		t.Error("expected 1 pending transactions, got", pool.pending[addr].Len())
	}
	if tx := pool.pending[addr].txs.items[0]; tx.Hash() != tx2.Hash() {
		t.Errorf("transaction mismatch: have %x, want %x", tx.Hash(), tx2.Hash())
	}
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
	tx := pricedTransaction(1, 100000, big.NewInt(300000000), key)
	if _, err := pool.add(tx, false); err != nil {
		t.Error("didn't expect error", err)
	}
	if len(pool.pending) != 0 {
		t.Error("expected 0 pending transactions, got", len(pool.pending))
	}
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

	tx := pricedTransaction(n, 100000, big.NewInt(300000000), key)
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
	if pool.pending[account].Len() != 3 {
		t.Errorf("pending transaction mismatch: have %d, want %d", pool.pending[account].Len(), 3)
	}
	if pool.queue[account].Len() != 3 {
		t.Errorf("queued transaction mismatch: have %d, want %d", pool.queue[account].Len(), 3)
	}
	if pool.all.Count() != 6 {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), 6)
	}
	<-pool.requestReset(nil, nil)
	if pool.pending[account].Len() != 3 {
		t.Errorf("pending transaction mismatch: have %d, want %d", pool.pending[account].Len(), 3)
	}
	if pool.queue[account].Len() != 3 {
		t.Errorf("queued transaction mismatch: have %d, want %d", pool.queue[account].Len(), 3)
	}
	if pool.all.Count() != 6 {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), 6)
	}
	// Reduce the balance of the account, and check that invalidated transactions are dropped
	testAddBalance(pool, account, big.NewInt(-650))
	<-pool.requestReset(nil, nil)

	if _, ok := pool.pending[account].txs.items[tx0.Nonce()]; !ok {
		t.Errorf("funded pending transaction missing: %v", tx0)
	}
	if _, ok := pool.pending[account].txs.items[tx1.Nonce()]; !ok {
		t.Errorf("funded pending transaction missing: %v", tx0)
	}
	if _, ok := pool.pending[account].txs.items[tx2.Nonce()]; ok {
		t.Errorf("out-of-fund pending transaction present: %v", tx1)
	}
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

	if _, ok := pool.pending[account].txs.items[tx0.Nonce()]; !ok {
		t.Errorf("funded pending transaction missing: %v", tx0)
	}
	if _, ok := pool.pending[account].txs.items[tx1.Nonce()]; ok {
		t.Errorf("over-gased pending transaction present: %v", tx1)
	}
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
func TestPostponing(t *testing.T) {
	t.Parallel()

	// Create the pool to test the postponing with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
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
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(113000000001000))
	}
	// Add a batch consecutive pending transactions for validation
	txs := []*types.Transaction{}
	for i, key := range keys {
		for j := 0; j < 10; j++ {
			var tx *types.Transaction
			if (i+j)%2 == 0 {
				tx = pricedTransaction(uint64(j), 25000, big.NewInt(300000000), key)
			} else {
				tx = pricedTransaction(uint64(j), 50000, big.NewInt(300000000), key)
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
	if pending := pool.pending[accs[0]].Len() + pool.pending[accs[1]].Len(); pending != len(txs) {
		t.Errorf("pending transaction mismatch: have %d, want %d", pending, len(txs))
	}
	if len(pool.queue) != 0 {
		t.Errorf("queued accounts mismatch: have %d, want %d", len(pool.queue), 0)
	}
	if pool.all.Count() != len(txs) {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), len(txs))
	}
	<-pool.requestReset(nil, nil)
	if pending := pool.pending[accs[0]].Len() + pool.pending[accs[1]].Len(); pending != len(txs) {
		t.Errorf("pending transaction mismatch: have %d, want %d", pending, len(txs))
	}
	if len(pool.queue) != 0 {
		t.Errorf("queued accounts mismatch: have %d, want %d", len(pool.queue), 0)
	}
	if pool.all.Count() != len(txs) {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), len(txs))
	}
	// Reduce the balance of the account, and check that transactions are reorganised
	for _, addr := range accs {
		testAddBalance(pool, addr, big.NewInt(-105500000000000))
	}
	<-pool.requestReset(nil, nil)

	// The first account's first transaction remains valid, check that subsequent
	// ones are either filtered out, or queued up for later.
	if _, ok := pool.pending[accs[0]].txs.items[txs[0].Nonce()]; !ok {
		t.Errorf("tx %d: valid and funded transaction missing from pending pool: %v", 0, txs[0])
	}
	if _, ok := pool.queue[accs[0]].txs.items[txs[0].Nonce()]; ok {
		t.Errorf("tx %d: valid and funded transaction present in future queue: %v", 0, txs[0])
	}
	for i, tx := range txs[1:10] {
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
	// The second account's first transaction got invalid, check that all transactions
	// are either filtered out, or queued up for later.
	if pool.pending[accs[1]] != nil {
		t.Errorf("invalidated account still has pending transactions")
	}
	for i, tx := range txs[10:] {
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
	testAddBalance(pool, account, big.NewInt(90000000000000))

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan core.NewTxsEvent, testTxPoolConfig.AccountQueue+5)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a pending and a queued transaction with a nonce-gap in between
	pool.addRemotesSync([]*types.Transaction{
		pricedTransaction(0, 100000, big.NewInt(300000000), key),
		pricedTransaction(2, 100000, big.NewInt(300000000), key),
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
	if err := pool.addRemoteSync(pricedTransaction(1, 100000, big.NewInt(300000000), key)); err != nil {
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
	testAddBalance(pool, account, big.NewInt(300000000000000))
	testTxPoolConfig.AccountQueue = 10
	// Keep queuing up transactions and make sure all above a limit are dropped
	for i := uint64(1); i <= testTxPoolConfig.AccountQueue; i++ {
		if err := pool.addRemoteSync(pricedTransaction(i, 100000, big.NewInt(300000000), key)); err != nil {
			t.Fatalf("tx %d: failed to add transaction: %v", i, err)
		}
		if len(pool.pending) != 0 {
			t.Errorf("tx %d: pending pool size mismatch: have %d, want %d", i, len(pool.pending), 0)
		}
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

// Tests that if the transaction count belonging to multiple accounts go above
// some threshold, the higher transactions are dropped to prevent DOS attacks.
//
// This logic should not hold for local transactions, unless the local tracking
// mechanism is disabled.
func TestQueueGlobalLimiting(t *testing.T) {
	testQueueGlobalLimiting(t, false)
}

func TestQueueGlobalLimitingNoLocals(t *testing.T) {
	testQueueGlobalLimiting(t, true)
}

func testQueueGlobalLimiting(t *testing.T, nolocals bool) {
	t.Parallel()

	// Create the pool to test the limit enforcement with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	config := testTxPoolConfig
	config.NoLocals = nolocals
	config.AccountQueue = 1
	config.GlobalQueue = config.AccountQueue*3 - 1 // reduce the queue limits to shorten test time (-1 to make it non divisible)

	pool := New(config, blockchain)
	pool.Init(testTxPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	// Create a number of test accounts and fund them (last one will be the local)
	keys := make([]*ecdsa.PrivateKey, 5)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(100000000000000))
	}
	local := keys[len(keys)-1]

	// Generate and queue a batch of transactions
	nonces := make(map[common.Address]uint64)

	txs := make(types.Transactions, 0, 3*config.GlobalQueue)
	for len(txs) < cap(txs) {
		key := keys[rand.Intn(len(keys)-1)] // skip adding transactions with the local account
		addr := crypto.PubkeyToAddress(key.PublicKey)

		txs = append(txs, pricedTransaction(nonces[addr]+1, 100000, big.NewInt(300000000), key))
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
	// Generate a batch of transactions from the local account and import them
	txs = txs[:0]
	for i := uint64(0); i < 3*config.GlobalQueue; i++ {
		txs = append(txs, pricedTransaction(i+1, 100000, big.NewInt(300000000), local))
	}
	pool.addLocals(txs)

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
func TestQueueTimeLimiting(t *testing.T) { testQueueTimeLimiting(t, false) }
func TestQueueTimeLimitingNoLocals(t *testing.T) {
	testQueueTimeLimiting(t, true)
}

func testQueueTimeLimiting(t *testing.T, nolocals bool) {
	// Reduce the eviction interval to a testable amount
	defer func(old time.Duration) { evictionInterval = old }(evictionInterval)
	evictionInterval = time.Millisecond * 100

	// Create the pool to test the non-expiration enforcement
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	config := testTxPoolConfig
	config.Lifetime = time.Second
	config.NoLocals = nolocals

	pool := New(config, blockchain)
	pool.Init(config.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	// Create two test accounts to ensure remotes expire but locals do not
	local, _ := crypto.GenerateKey()
	remote, _ := crypto.GenerateKey()

	testAddBalance(pool, crypto.PubkeyToAddress(local.PublicKey), big.NewInt(100000000000000))
	testAddBalance(pool, crypto.PubkeyToAddress(remote.PublicKey), big.NewInt(100000000000000))

	// Add the two transactions and ensure they both are queued up
	if err := pool.addLocal(pricedTransaction(1, 100000, big.NewInt(300000000), local)); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}
	if err := pool.addRemote(pricedTransaction(1, 100000, big.NewInt(300000000), remote)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	pending, queued := pool.Stats()
	if pending != 0 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 0)
	}
	if queued != 2 {
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
	if queued != 2 {
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
	if nolocals {
		if queued != 0 {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
		}
	} else {
		if queued != 1 {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 1)
		}
	}
	if err := validatePoolInternals(pool); err != nil {
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
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}

	// Queue gapped transactions
	if err := pool.addLocal(pricedTransaction(4, 100000, big.NewInt(300000000), local)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(4, 100000, big.NewInt(300000000), remote)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	time.Sleep(5 * evictionInterval) // A half lifetime pass

	// Queue executable transactions, the life cycle should be restarted.
	if err := pool.addLocal(pricedTransaction(2, 100000, big.NewInt(300000000), local)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(2, 100000, big.NewInt(300000000), remote)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	time.Sleep(6 * evictionInterval)

	// All gapped transactions shouldn't be kicked out
	pending, queued = pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}
	if err := validatePoolInternals(pool); err != nil {
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
	testAddBalance(pool, account, big.NewInt(400000000000000))
	testTxPoolConfig.AccountQueue = 10

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan core.NewTxsEvent, testTxPoolConfig.AccountQueue)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Keep queuing up transactions and make sure all above a limit are dropped
	for i := uint64(0); i < testTxPoolConfig.AccountQueue; i++ {
		if err := pool.addRemoteSync(pricedTransaction(i, 100000, big.NewInt(300000000), key)); err != nil {
			t.Fatalf("tx %d: failed to add transaction: %v", i, err)
		}
		if pool.pending[account].Len() != int(i)+1 {
			t.Errorf("tx %d: pending pool size mismatch: have %d, want %d", i, pool.pending[account].Len(), i+1)
		}
		if len(pool.queue) != 0 {
			t.Errorf("tx %d: queue size mismatch: have %d, want %d", i, pool.queue[account].Len(), 0)
		}
	}
	if pool.all.Count() != int(testTxPoolConfig.AccountQueue) {
		t.Errorf("total transaction mismatch: have %d, want %d", pool.all.Count(), testTxPoolConfig.AccountQueue+5)
	}
	if err := validateEvents(events, int(testTxPoolConfig.AccountQueue)); err != nil {
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
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
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
	for _, list := range pool.pending {
		pending += list.Len()
	}
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
	minGasPrice := common.GetMinGasPrice(pool.currentHead.Load().Number)
	fundedBalance := new(big.Int).Mul(minGasPrice, new(big.Int).SetUint64(pool.currentHead.Load().GasLimit))
	fundedBalance.Mul(fundedBalance, big.NewInt(3))
	testAddBalance(pool, account, fundedBalance)

	// Find the maximum data length for the kind of transaction which will
	// be generated in the pool.addRemoteSync calls below.
	const largeDataLength = txMaxSize - 200 // enough to have a 5 bytes RLP encoding of the data length number
	txWithLargeData := pricedDataTransaction(0, pool.currentHead.Load().GasLimit, minGasPrice, key, largeDataLength)
	maxTxLengthWithoutData := txWithLargeData.Size() - largeDataLength
	maxTxDataLength := txMaxSize - maxTxLengthWithoutData
	for tx := pricedDataTransaction(0, pool.currentHead.Load().GasLimit, minGasPrice, key, maxTxDataLength); tx.Size() > txMaxSize; {
		maxTxDataLength--
		tx = pricedDataTransaction(0, pool.currentHead.Load().GasLimit, minGasPrice, key, maxTxDataLength)
	}
	minOversizedDataLength := maxTxDataLength + 1
	for tx := pricedDataTransaction(0, pool.currentHead.Load().GasLimit, minGasPrice, key, minOversizedDataLength); tx.Size() <= txMaxSize; {
		minOversizedDataLength++
		tx = pricedDataTransaction(0, pool.currentHead.Load().GasLimit, minGasPrice, key, minOversizedDataLength)
	}

	// Try adding a transaction with maximal allowed size
	tx := pricedDataTransaction(0, pool.currentHead.Load().GasLimit, minGasPrice, key, maxTxDataLength)
	if err := pool.addRemoteSync(tx); err != nil {
		t.Fatalf("failed to add transaction of size %d, close to maximal: %v", int(tx.Size()), err)
	}
	// Try adding a transaction with random allowed size
	if err := pool.addRemoteSync(pricedDataTransaction(1, pool.currentHead.Load().GasLimit, minGasPrice, key, uint64(rand.Intn(int(maxTxDataLength+1))))); err != nil {
		t.Fatalf("failed to add transaction of random allowed size: %v", err)
	}
	// Try adding a transaction with the smallest data length that is already oversized.
	if err := pool.addRemoteSync(pricedDataTransaction(2, pool.currentHead.Load().GasLimit, minGasPrice, key, minOversizedDataLength)); err == nil {
		t.Fatalf("expected rejection on slightly oversize transaction")
	}
	// Try adding a transaction above maximum size by more than one
	if err := pool.addRemoteSync(pricedDataTransaction(2, pool.currentHead.Load().GasLimit, minGasPrice, key, minOversizedDataLength+uint64(rand.Intn(10*txMaxSize)))); err == nil {
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
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
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
	pool.AddRemotes(txs)
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
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	config := testTxPoolConfig
	config.AccountSlots = 5
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

	for addr, list := range pool.pending {
		if list.Len() != int(config.AccountSlots) {
			t.Errorf("addr %x: total pending transactions mismatch: have %d, want %d", addr, list.Len(), config.AccountSlots)
		}
	}
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
}

// Tests that setting the transaction pool gas price to a higher value correctly
// discards everything cheaper than that and moves any gapped transactions back
// from the pending pool to the queue.
//
// Note, local transactions are never allowed to be dropped.
func TestRepricing(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	pool := New(testTxPoolConfig, blockchain)
	pool.Init(testTxPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan core.NewTxsEvent, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 4)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(200000000000000))
	}
	// Generate and queue a batch of transactions, both pending and queued
	txs := types.Transactions{}

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(500000000), keys[0]))
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(250000000), keys[0]))
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(500000000), keys[0]))

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(250000000), keys[1]))
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(500000000), keys[1]))
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(500000000), keys[1]))

	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(500000000), keys[2]))
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(250000000), keys[2]))
	txs = append(txs, pricedTransaction(3, 100000, big.NewInt(500000000), keys[2]))

	ltx := pricedTransaction(0, 100000, big.NewInt(250000000), keys[3])

	// Import the batch and that both pending and queued transactions match up
	pool.addRemotesSync(txs)
	pool.addLocal(ltx)

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
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Reprice the pool and check that underpriced transactions get dropped
	pool.SetGasTip(big.NewInt(500000000))

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
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Check that we can't add the old transactions back
	if err := pool.addRemote(pricedTransaction(1, 100000, big.NewInt(250000000), keys[0])); !errors.Is(err, txpool.ErrUnderpriced) {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, txpool.ErrUnderpriced)
	}
	if err := pool.addRemote(pricedTransaction(0, 100000, big.NewInt(250000000), keys[1])); !errors.Is(err, txpool.ErrUnderpriced) {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, txpool.ErrUnderpriced)
	}
	if err := pool.addRemote(pricedTransaction(2, 100000, big.NewInt(250000000), keys[2])); !errors.Is(err, txpool.ErrUnderpriced) {
		t.Fatalf("adding underpriced queued transaction error mismatch: have %v, want %v", err, txpool.ErrUnderpriced)
	}
	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("post-reprice event firing failed: %v", err)
	}
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// However we can add local underpriced transactions
	tx := pricedTransaction(1, 100000, big.NewInt(250000000), keys[3])
	if err := pool.addLocal(tx); err != nil {
		t.Fatalf("failed to add underpriced local transaction: %v", err)
	}
	if pending, _ = pool.Stats(); pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}
	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("post-reprice local event firing failed: %v", err)
	}
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// And we can fill gaps with properly priced transactions
	if err := pool.addRemote(pricedTransaction(1, 100000, big.NewInt(500000000), keys[0])); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}
	if err := pool.addRemote(pricedTransaction(0, 100000, big.NewInt(500000000), keys[1])); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(2, 100000, big.NewInt(500000000), keys[2])); err != nil {
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
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
	blockchain := newTestBlockChain(eip1559Config, 10000000, statedb, new(event.Feed))

	txPoolConfig := DefaultConfig
	txPoolConfig.NoLocals = true
	pool := New(txPoolConfig, blockchain)
	pool.Init(txPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	key, _ := crypto.GenerateKey()
	testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(1_000_000_000_000_000_000))

	minGasPrice := common.GetMinGasPrice(blockchain.CurrentBlock().Number)
	legacyPrice := new(big.Int).Add(minGasPrice, big.NewInt(1))
	dynamicTip := new(big.Int).Add(minGasPrice, big.NewInt(1))
	dynamicFeeCap := new(big.Int).Add(minGasPrice, big.NewInt(2))

	tx := pricedTransaction(0, 100000, legacyPrice, key)
	pool.SetGasTip(new(big.Int).Add(legacyPrice, big.NewInt(1)))

	if err := pool.addLocal(tx); !errors.Is(err, txpool.ErrUnderpriced) {
		t.Fatalf("Min tip not enforced")
	}

	if err := pool.Add([]*types.Transaction{tx}, true, false)[0]; !errors.Is(err, txpool.ErrUnderpriced) {
		t.Fatalf("Min tip not enforced")
	}

	tx = dynamicFeeTx(0, 100000, dynamicFeeCap, dynamicTip, key)
	pool.SetGasTip(new(big.Int).Add(dynamicTip, big.NewInt(1)))

	if err := pool.addLocal(tx); !errors.Is(err, txpool.ErrUnderpriced) {
		t.Fatalf("Min tip not enforced")
	}

	if err := pool.Add([]*types.Transaction{tx}, true, false)[0]; !errors.Is(err, txpool.ErrUnderpriced) {
		t.Fatalf("Min tip not enforced")
	}
	// Make sure the tx is accepted if locals are enabled
	pool.config.NoLocals = false
	if err := pool.Add([]*types.Transaction{tx}, true, false)[0]; err != nil {
		t.Fatalf("Min tip enforced with locals enabled, error: %v", err)
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
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(200000000000000))
	}
	// Generate and queue a batch of transactions, both pending and queued
	txs := types.Transactions{}

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(350000000), keys[0]))
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(300000000), keys[0]))
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(350000000), keys[0]))

	txs = append(txs, dynamicFeeTx(0, 100000, big.NewInt(350000000), big.NewInt(300000000), keys[1]))
	txs = append(txs, dynamicFeeTx(1, 100000, big.NewInt(400000000), big.NewInt(350000000), keys[1]))
	txs = append(txs, dynamicFeeTx(2, 100000, big.NewInt(400000000), big.NewInt(350000000), keys[1]))

	txs = append(txs, dynamicFeeTx(1, 100000, big.NewInt(350000000), big.NewInt(350000000), keys[2]))
	txs = append(txs, dynamicFeeTx(2, 100000, big.NewInt(300000000), big.NewInt(300000000), keys[2]))
	txs = append(txs, dynamicFeeTx(3, 100000, big.NewInt(350000000), big.NewInt(350000000), keys[2]))

	ltx := dynamicFeeTx(0, 100000, big.NewInt(350000000), big.NewInt(300000000), keys[3])

	// Import the batch and that both pending and queued transactions match up
	pool.addRemotesSync(txs)
	pool.addLocal(ltx)

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
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Reprice the pool and check that underpriced transactions get dropped
	pool.SetGasTip(big.NewInt(350000000))

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
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Check that we can't add the old transactions back
	tx := pricedTransaction(1, 100000, big.NewInt(300000000), keys[0])
	if err := pool.addRemote(tx); !errors.Is(err, txpool.ErrUnderpriced) {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, txpool.ErrUnderpriced)
	}
	tx = dynamicFeeTx(0, 100000, big.NewInt(350000000), big.NewInt(300000000), keys[1])
	if err := pool.addRemote(tx); !errors.Is(err, txpool.ErrUnderpriced) {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, txpool.ErrUnderpriced)
	}
	tx = dynamicFeeTx(2, 100000, big.NewInt(300000000), big.NewInt(300000000), keys[2])
	if err := pool.addRemote(tx); !errors.Is(err, txpool.ErrUnderpriced) {
		t.Fatalf("adding underpriced queued transaction error mismatch: have %v, want %v", err, txpool.ErrUnderpriced)
	}
	if err := validateEvents(events, 0); err != nil {
		t.Fatalf("post-reprice event firing failed: %v", err)
	}
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// However we can add local underpriced transactions
	tx = dynamicFeeTx(1, 100000, big.NewInt(300000000), big.NewInt(300000000), keys[3])
	if err := pool.addLocal(tx); err != nil {
		t.Fatalf("failed to add underpriced local transaction: %v", err)
	}
	if pending, _ = pool.Stats(); pending != 3 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 3)
	}
	if err := validateEvents(events, 1); err != nil {
		t.Fatalf("post-reprice local event firing failed: %v", err)
	}
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// And we can fill gaps with properly priced transactions
	tx = pricedTransaction(1, 100000, big.NewInt(350000000), keys[0])
	if err := pool.addRemote(tx); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}
	tx = dynamicFeeTx(0, 100000, big.NewInt(400000000), big.NewInt(350000000), keys[1])
	if err := pool.addRemote(tx); err != nil {
		t.Fatalf("failed to add pending transaction: %v", err)
	}
	tx = dynamicFeeTx(2, 100000, big.NewInt(350000000), big.NewInt(350000000), keys[2])
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

// Tests that setting the transaction pool gas price to a higher value does not
// remove local transactions (legacy & dynamic fee).
func TestRepricingKeepsLocals(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
	blockchain := newTestBlockChain(eip1559Config, 1000000, statedb, new(event.Feed))

	pool := New(testTxPoolConfig, blockchain)
	pool.Init(testTxPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	// Create a number of test accounts and fund them
	keys := make([]*ecdsa.PrivateKey, 3)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1500000000000000))
	}
	// Create transaction (both pending and queued) with a linearly growing gasprice
	// common.LimitThresholdNonceInQueue = 10
	for i := uint64(0); i < 5; i++ {
		// Add pending transaction.
		pendingTx := pricedTransaction(i, 100000, big.NewInt(int64((i+1)*250000000)), keys[2])
		if err := pool.addLocal(pendingTx); err != nil {
			t.Fatal(err)
		}
		// Add queued transaction.
		queuedTx := pricedTransaction(i+6, 100000, big.NewInt(int64((i+1)*250000000)), keys[2])
		if err := pool.addLocal(queuedTx); err != nil {
			t.Fatal(err)
		}

		// Add pending dynamic fee transaction.
		pendingTx = dynamicFeeTx(i, 100000, big.NewInt(int64((i+1)*250000000)), big.NewInt(int64((i+1)*250000000)), keys[1])
		if err := pool.addLocal(pendingTx); err != nil {
			t.Fatal(err)
		}
		// Add queued dynamic fee transaction.
		queuedTx = dynamicFeeTx(i+6, 100000, big.NewInt(int64((i+1)*250000000)), big.NewInt(int64((i+1)*250000000)), keys[1])
		if err := pool.addLocal(queuedTx); err != nil {
			t.Fatal(err)
		}
	}
	pending, queued := pool.Stats()
	expPending, expQueued := 10, 10
	validate := func() {
		pending, queued = pool.Stats()
		if pending != expPending {
			t.Fatalf("pending transactions mismatched: have %d, want %d", pending, expPending)
		}
		if queued != expQueued {
			t.Fatalf("queued transactions mismatched: have %d, want %d", queued, expQueued)
		}

		if err := validatePoolInternals(pool); err != nil {
			t.Fatalf("pool internal state corrupted: %v", err)
		}
	}
	validate()

	// Reprice the pool and check that nothing is dropped
	pool.SetGasTip(big.NewInt(500000000))
	validate()

	pool.SetGasTip(big.NewInt(500000000))
	pool.SetGasTip(big.NewInt(1000000000))
	pool.SetGasTip(big.NewInt(2000000000))
	pool.SetGasTip(big.NewInt(25000000000))
	validate()
}

// Tests that when the pool reaches its global transaction limit, underpriced
// transactions are gradually shifted out for more expensive ones and any gapped
// pending transactions are moved into the queue.
//
// Note, local transactions are never allowed to be dropped.
func TestUnderpricing(t *testing.T) {
	t.Parallel()

	// Create the pool to test the pricing enforcement with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
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
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(600000000000000))
	}
	// Generate and queue a batch of transactions, both pending and queued
	// Gas prices follow an intentional stepped pattern (250M, 500M, 750M, 1000M, 1250M) to test
	// transaction replacement and eviction logic based on price competition. The 1:5 ratio
	// (250M to 1250M) ensures clear differentiation when testing underpricing behavior.
	txs := types.Transactions{}

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(250000000), keys[0]))
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(500000000), keys[0]))

	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(250000000), keys[1]))

	ltx := pricedTransaction(0, 100000, big.NewInt(250000000), keys[2])

	// Import the batch and that both pending and queued transactions match up
	pool.AddRemotes(txs)
	pool.addLocal(ltx)

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
	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(250000000), keys[1])); !errors.Is(err, txpool.ErrUnderpriced) {
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, txpool.ErrUnderpriced)
	}
	// Replace a future transaction with a future transaction
	if err := pool.addRemoteSync(pricedTransaction(1, 100000, big.NewInt(500000000), keys[1])); err != nil { // +K1:1 => -K1:1 => Pend K0:0, K0:1, K2:0; Que K1:1
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	// Ensure that adding high priced transactions drops cheap ones, but not own
	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(750000000), keys[1])); err != nil { // +K1:0 => -K1:1 => Pend K0:0, K0:1, K1:0, K2:0; Que -
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(2, 100000, big.NewInt(1000000000), keys[1])); err != nil { // +K1:2 => -K0:0 => Pend K1:0, K2:0; Que K0:1 K1:2
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	if err := pool.addRemote(pricedTransaction(3, 100000, big.NewInt(1250000000), keys[1])); err != nil { // +K1:3 => -K0:1 => Pend K1:0, K2:0; Que K1:2 K1:3
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	// Ensure that replacing a pending transaction with a future transaction fails
	if err := pool.addRemote(pricedTransaction(5, 100000, big.NewInt(1500000000), keys[1])); err != ErrFutureReplacePending {
		t.Fatalf("adding future replace transaction error mismatch: have %v, want %v", err, ErrFutureReplacePending)
	}
	pending, queued = pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}
	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("additional event firing failed: %v", err)
	}
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Ensure that adding local transactions can push out even higher priced ones
	ltx = pricedTransaction(1, 100000, big.NewInt(250000000), keys[2])
	if err := pool.addLocal(ltx); err != nil {
		t.Fatalf("failed to append underpriced local transaction: %v", err)
	}
	ltx = pricedTransaction(0, 100000, big.NewInt(250000000), keys[3])
	if err := pool.addLocal(ltx); err != nil {
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
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	config := testTxPoolConfig
	config.GlobalSlots = common.LimitThresholdNonceInQueue
	config.GlobalQueue = 0
	config.AccountSlots = config.GlobalSlots - 1

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
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(400000000000000))
	}
	// Fill up the entire queue with the same transaction price points
	txs := types.Transactions{}
	for i := uint64(0); i < config.GlobalSlots; i++ {
		txs = append(txs, pricedTransaction(i, 100000, big.NewInt(300000000), keys[0]))
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
	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(350000000), keys[1])); err != nil {
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
//
// Note, local transactions are never allowed to be dropped.
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
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(200000000000000))
	}

	// Generate and queue a batch of transactions, both pending and queued
	txs := types.Transactions{}

	txs = append(txs, dynamicFeeTx(0, 100000, big.NewInt(270000000), big.NewInt(260000000), keys[0]))
	txs = append(txs, pricedTransaction(1, 100000, big.NewInt(260000000), keys[0]))
	txs = append(txs, dynamicFeeTx(1, 100000, big.NewInt(260000000), big.NewInt(250000000), keys[1]))

	ltx := dynamicFeeTx(0, 100000, big.NewInt(260000000), big.NewInt(250000000), keys[2])

	// Import the batch and that both pending and queued transactions match up
	pool.AddRemotes(txs) // Pend K0:0, K0:1; Que K1:1
	pool.addLocal(ltx)   // +K2:0 => Pend K0:0, K0:1, K2:0; Que K1:1

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
	tx := dynamicFeeTx(0, 100000, big.NewInt(260000000), big.NewInt(250000000), keys[1])
	if err := pool.addRemote(tx); !errors.Is(err, txpool.ErrUnderpriced) { // Pend K0:0, K0:1, K2:0; Que K1:1
		t.Fatalf("adding underpriced pending transaction error mismatch: have %v, want %v", err, txpool.ErrUnderpriced)
	}

	// Ensure that adding high priced transactions drops cheap ones, but not own
	tx = pricedTransaction(0, 100000, big.NewInt(260000000), keys[1])
	if err := pool.addRemote(tx); err != nil { // +K1:0, -K1:1 => Pend K0:0, K0:1, K1:0, K2:0; Que -
		t.Fatalf("failed to add well priced transaction: %v", err)
	}

	tx = pricedTransaction(1, 100000, big.NewInt(270000000), keys[1])
	if err := pool.addRemoteSync(tx); err != nil { // +K1:2, -K0:1 => Pend K0:0 K1:0, K2:0; Que K1:2
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	tx = dynamicFeeTx(2, 100000, big.NewInt(280000000), big.NewInt(250000000), keys[1])
	if err := pool.addRemoteSync(tx); err != nil { // +K1:3, -K1:0 => Pend K0:0 K2:0; Que K1:2 K1:3
		t.Fatalf("failed to add well priced transaction: %v", err)
	}
	pending, queued = pool.Stats()
	if pending != 2 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 2)
	}
	if queued != 2 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 2)
	}
	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("additional event firing failed: %v", err)
	}
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Ensure that adding local transactions can push out even higher priced ones
	ltx = dynamicFeeTx(1, 100000, big.NewInt(250000000), big.NewInt(250000000), keys[2])
	if err := pool.addLocal(ltx); err != nil {
		t.Fatalf("failed to append underpriced local transaction: %v", err)
	}
	ltx = dynamicFeeTx(0, 100000, big.NewInt(250000000), big.NewInt(250000000), keys[3])
	if err := pool.addLocal(ltx); err != nil {
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
		if pool.all.GetRemote(tx.Hash()) == nil {
			t.Fatalf("highest %s transaction evicted from the pool", name)
		}
	}

	add := func(urgent bool) {
		for i := 0; i < 20; i++ {
			var tx *types.Transaction
			// Create a test accounts and fund it
			key, _ := crypto.GenerateKey()
			testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(100000000000000))
			if urgent {
				tx = dynamicFeeTx(0, 100000, big.NewInt(int64(250000000+baseFee+1+i)), big.NewInt(int64(250000000+1+i)), key)
				highTip = tx
			} else {
				tx = dynamicFeeTx(0, 100000, big.NewInt(int64(250000000+baseFee+200+i)), big.NewInt(250000000), key)
				highCap = tx
			}
			pool.addRemotesSync([]*types.Transaction{tx})
		}
		pending, queued := pool.Stats()
		if pending+queued != 20 {
			t.Fatalf("transaction count mismatch: have %d, want %d", pending+queued, 20)
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
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	pool := New(testTxPoolConfig, blockchain)
	pool.Init(testTxPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	// Create a test account to add transactions with
	key, _ := crypto.GenerateKey()
	testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(300000000000000))

	// Create a batch of transactions and add a few of them
	txs := make([]*types.Transaction, common.LimitThresholdNonceInQueue)
	for i := 0; i < len(txs); i++ {
		txs[i] = pricedTransaction(uint64(i), 100000, big.NewInt(300000000), key)
	}
	var firsts []*types.Transaction
	for i := 0; i < len(txs); i += 2 {
		firsts = append(firsts, txs[i])
	}
	errs := pool.addRemotesSync(firsts)
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
	errs = pool.addRemotesSync(txs)
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
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
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
	testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(500000000000000))

	// Add pending transactions, ensuring the minimum price bump is enforced for replacement (for ultra low prices too)
	price := int64(500000000)
	threshold := (price * (100 + int64(testTxPoolConfig.PriceBump))) / 100

	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(250000000), key)); err != nil {
		t.Fatalf("failed to add original cheap pending transaction: %v", err)
	}
	if err := pool.addRemote(pricedTransaction(0, 100001, big.NewInt(250000000), key)); err != txpool.ErrReplaceUnderpriced {
		t.Fatalf("original cheap pending transaction replacement error mismatch: have %v, want %v", err, txpool.ErrReplaceUnderpriced)
	}
	if err := pool.addRemote(pricedTransaction(0, 100000, big.NewInt(300000000), key)); err != nil {
		t.Fatalf("failed to replace original cheap pending transaction: %v", err)
	}
	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("cheap replacement event firing failed: %v", err)
	}

	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(price), key)); err != nil {
		t.Fatalf("failed to add original proper pending transaction: %v", err)
	}
	if err := pool.addRemote(pricedTransaction(0, 100001, big.NewInt(threshold-1), key)); err != txpool.ErrReplaceUnderpriced {
		t.Fatalf("original proper pending transaction replacement error mismatch: have %v, want %v", err, txpool.ErrReplaceUnderpriced)
	}
	if err := pool.addRemote(pricedTransaction(0, 100000, big.NewInt(threshold), key)); err != nil {
		t.Fatalf("failed to replace original proper pending transaction: %v", err)
	}
	if err := validateEvents(events, 2); err != nil {
		t.Fatalf("proper replacement event firing failed: %v", err)
	}

	// Add queued transactions, ensuring the minimum price bump is enforced for replacement (for ultra low prices too)
	if err := pool.addRemote(pricedTransaction(2, 100000, big.NewInt(250000000), key)); err != nil {
		t.Fatalf("failed to add original cheap queued transaction: %v", err)
	}
	if err := pool.addRemote(pricedTransaction(2, 100001, big.NewInt(250000000), key)); err != txpool.ErrReplaceUnderpriced {
		t.Fatalf("original cheap queued transaction replacement error mismatch: have %v, want %v", err, txpool.ErrReplaceUnderpriced)
	}
	if err := pool.addRemote(pricedTransaction(2, 100000, big.NewInt(300000000), key)); err != nil {
		t.Fatalf("failed to replace original cheap queued transaction: %v", err)
	}

	if err := pool.addRemote(pricedTransaction(2, 100000, big.NewInt(price), key)); err != nil {
		t.Fatalf("failed to add original proper queued transaction: %v", err)
	}
	if err := pool.addRemote(pricedTransaction(2, 100001, big.NewInt(threshold-1), key)); err != txpool.ErrReplaceUnderpriced {
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
	testAddBalance(pool, crypto.PubkeyToAddress(key.PublicKey), big.NewInt(500000000000000))

	// Keep track of transaction events to ensure all executables get announced
	events := make(chan core.NewTxsEvent, 32)
	sub := pool.txFeed.Subscribe(events)
	defer sub.Unsubscribe()

	// Add pending transactions, ensuring the minimum price bump is enforced for replacement (for ultra low prices too)
	gasFeeCap := int64(500000000)
	feeCapThreshold := (gasFeeCap * (100 + int64(testTxPoolConfig.PriceBump))) / 100
	gasTipCap := int64(400000000)
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
		tx := dynamicFeeTx(nonce, 100000, big.NewInt(300000000), big.NewInt(250000000), key)
		if err := pool.addRemoteSync(tx); err != nil {
			t.Fatalf("failed to add original cheap %s transaction: %v", stage, err)
		}
		// 2.  Don't bump tip or feecap => discard
		tx = dynamicFeeTx(nonce, 100001, big.NewInt(300000000), big.NewInt(250000000), key)
		if err := pool.addRemote(tx); err != txpool.ErrReplaceUnderpriced {
			t.Fatalf("original cheap %s transaction replacement error mismatch: have %v, want %v", stage, err, txpool.ErrReplaceUnderpriced)
		}
		// 3.  Bump both more than min => accept
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(350000000), big.NewInt(300000000), key)
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
		if err := pool.addRemote(tx); err != txpool.ErrReplaceUnderpriced {
			t.Fatalf("original proper %s transaction replacement error mismatch: have %v, want %v", stage, err, txpool.ErrReplaceUnderpriced)
		}
		// 7.  Bump fee cap max allowed so it's still underpriced => discard
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(feeCapThreshold-1), big.NewInt(gasTipCap), key)
		if err := pool.addRemote(tx); err != txpool.ErrReplaceUnderpriced {
			t.Fatalf("original proper %s transaction replacement error mismatch: have %v, want %v", stage, err, txpool.ErrReplaceUnderpriced)
		}
		// 8.  Bump tip min for acceptance => accept
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(gasFeeCap), big.NewInt(tipThreshold), key)
		if err := pool.addRemote(tx); err != txpool.ErrReplaceUnderpriced {
			t.Fatalf("original proper %s transaction replacement error mismatch: have %v, want %v", stage, err, txpool.ErrReplaceUnderpriced)
		}
		// 9.  Bump fee cap min for acceptance => accept
		tx = dynamicFeeTx(nonce, 100000, big.NewInt(feeCapThreshold), big.NewInt(gasTipCap), key)
		if err := pool.addRemote(tx); err != txpool.ErrReplaceUnderpriced {
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

// Tests that local transactions are journaled to disk, but remote transactions
// get discarded between restarts.
func TestJournaling(t *testing.T) { testJournaling(t, false) }

func TestJournalingNoLocals(t *testing.T) { testJournaling(t, true) }

func testJournaling(t *testing.T, nolocals bool) {
	t.Parallel()

	// Create a temporary file for the journal
	file, err := os.CreateTemp(t.TempDir(), "")
	if err != nil {
		t.Fatalf("failed to create temporary journal: %v", err)
	}
	journal := file.Name()
	defer os.Remove(journal)

	// Clean up the temporary file, we only need the path for now
	file.Close()
	os.Remove(journal)

	// Create the original pool to inject transaction into the journal
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	config := testTxPoolConfig
	config.NoLocals = nolocals
	config.Journal = journal
	config.Rejournal = time.Second

	pool := New(config, blockchain)
	pool.Init(config.PriceLimit, blockchain.CurrentBlock(), newReserver())

	// Create two test accounts to ensure remotes expire but locals do not
	local, _ := crypto.GenerateKey()
	remote, _ := crypto.GenerateKey()

	testAddBalance(pool, crypto.PubkeyToAddress(local.PublicKey), big.NewInt(100000000000000))
	testAddBalance(pool, crypto.PubkeyToAddress(remote.PublicKey), big.NewInt(100000000000000))

	// Add three local and a remote transactions and ensure they are queued up
	if err := pool.addLocal(pricedTransaction(0, 100000, big.NewInt(300000000), local)); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}
	if err := pool.addLocal(pricedTransaction(1, 100000, big.NewInt(300000000), local)); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}
	if err := pool.addLocal(pricedTransaction(2, 100000, big.NewInt(300000000), local)); err != nil {
		t.Fatalf("failed to add local transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(0, 100000, big.NewInt(300000000), remote)); err != nil {
		t.Fatalf("failed to add remote transaction: %v", err)
	}
	pending, queued := pool.Stats()
	if pending != 4 {
		t.Fatalf("pending transactions mismatched: have %d, want %d", pending, 4)
	}
	if queued != 0 {
		t.Fatalf("queued transactions mismatched: have %d, want %d", queued, 0)
	}
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Terminate the old pool, bump the local nonce, create a new pool and ensure relevant transaction survive
	pool.Close()
	statedb.SetNonce(crypto.PubkeyToAddress(local.PublicKey), 1)
	blockchain = newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	pool = New(config, blockchain)
	pool.Init(config.PriceLimit, blockchain.CurrentBlock(), newReserver())

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
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	// Bump the nonce temporarily and ensure the newly invalidated transaction is removed
	statedb.SetNonce(crypto.PubkeyToAddress(local.PublicKey), 2)
	<-pool.requestReset(nil, nil)
	time.Sleep(2 * config.Rejournal)
	pool.Close()

	statedb.SetNonce(crypto.PubkeyToAddress(local.PublicKey), 1)
	blockchain = newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))
	pool = New(config, blockchain)
	pool.Init(config.PriceLimit, blockchain.CurrentBlock(), newReserver())

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
	if err := validatePoolInternals(pool); err != nil {
		t.Fatalf("pool internal state corrupted: %v", err)
	}
	pool.Close()
}

// TestTransactionStatusCheck tests that the pool can correctly retrieve the
// pending status of individual transactions.
func TestStatusCheck(t *testing.T) {
	t.Parallel()

	// Create the pool to test the status retrievals with
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()))
	blockchain := newTestBlockChain(params.TestChainConfig, 1000000, statedb, new(event.Feed))

	pool := New(testTxPoolConfig, blockchain)
	pool.Init(testTxPoolConfig.PriceLimit, blockchain.CurrentBlock(), newReserver())
	defer pool.Close()

	// Create the test accounts to check various transaction statuses with
	keys := make([]*ecdsa.PrivateKey, 3)
	for i := 0; i < len(keys); i++ {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(100000000000000))
	}
	// Generate and queue a batch of transactions, both pending and queued
	txs := types.Transactions{}

	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(300000000), keys[0])) // Pending only
	txs = append(txs, pricedTransaction(0, 100000, big.NewInt(300000000), keys[1])) // Pending and queued
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(300000000), keys[1]))
	txs = append(txs, pricedTransaction(2, 100000, big.NewInt(300000000), keys[2])) // Queued only

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

	minGasPrice := new(big.Int).Set(common.MinGasPrice)
	minGasFee := uint256.MustFromBig(minGasPrice)
	doubleGasFee := new(uint256.Int).Lsh(new(uint256.Int).Set(minGasFee), 1)
	tripleGasFee := new(uint256.Int).Mul(new(uint256.Int).Set(minGasFee), uint256.NewInt(3))
	legacyReplacePrice := new(big.Int).Mul(minGasPrice, big.NewInt(10))

	for _, tt := range []struct {
		name    string
		pending int
		queued  int
		run     func(string)
	}{
		{
			// Check that only one in-flight transaction is allowed for accounts
			// with delegation set. Also verify the accepted transaction can be
			// replaced by fee.
			name:    "only-one-in-flight",
			pending: 1,
			run: func(name string) {
				aa := common.Address{0xaa, 0xaa}
				statedb.SetCode(addrA, append(types.DelegationPrefix, aa.Bytes()...))
				statedb.SetCode(aa, []byte{byte(vm.ADDRESS), byte(vm.PUSH0), byte(vm.SSTORE)})

				// Send gapped transaction, it should be rejected.
				if err := pool.addRemoteSync(pricedTransaction(2, 100000, minGasPrice, keyA)); !errors.Is(err, ErrOutOfOrderTxFromDelegated) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, ErrOutOfOrderTxFromDelegated, err)
				}
				// Send transactions. First is accepted, second is rejected.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, minGasPrice, keyA)); err != nil {
					t.Fatalf("%s: failed to add remote transaction: %v", name, err)
				}
				if err := pool.addRemoteSync(pricedTransaction(1, 100000, minGasPrice, keyA)); !errors.Is(err, txpool.ErrInflightTxLimitReached) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, txpool.ErrInflightTxLimitReached, err)
				}
				// Check gapped transaction again.
				if err := pool.addRemoteSync(pricedTransaction(2, 100000, minGasPrice, keyA)); !errors.Is(err, txpool.ErrInflightTxLimitReached) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, txpool.ErrInflightTxLimitReached, err)
				}
				// Replace by fee.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, legacyReplacePrice, keyA)); err != nil {
					t.Fatalf("%s: failed to replace with remote transaction: %v", name, err)
				}
			},
		},
		{
			name:    "allow-setcode-tx-with-pending-authority-tx",
			pending: 2,
			run: func(name string) {
				// Send two transactions where the first has no conflicting delegations and
				// the second should be allowed despite conflicting with the authorities in 1).
				if err := pool.addRemoteSync(setCodeTx(0, keyA, []unsignedAuth{{1, keyC}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				if err := pool.addRemoteSync(setCodeTx(0, keyB, []unsignedAuth{{1, keyC}})); err != nil {
					t.Fatalf("%s: failed to add conflicting delegation: %v", name, err)
				}
			},
		},
		{
			name:    "allow-one-tx-from-pooled-delegation",
			pending: 2,
			run: func(name string) {
				// Verify C cannot originate another transaction when it has a pooled delegation.
				if err := pool.addRemoteSync(setCodeTx(0, keyA, []unsignedAuth{{0, keyC}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, minGasPrice, keyC)); err != nil {
					t.Fatalf("%s: failed to add with pending delegation: %v", name, err)
				}
				// Also check gapped transaction is rejected.
				if err := pool.addRemoteSync(pricedTransaction(1, 100000, minGasPrice, keyC)); !errors.Is(err, txpool.ErrInflightTxLimitReached) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, txpool.ErrInflightTxLimitReached, err)
				}
			},
		},
		{
			name:    "replace-by-fee-setcode-tx",
			pending: 1,
			run: func(name string) {
				// 4. Fee bump the setcode tx send.
				if err := pool.addRemoteSync(setCodeTx(0, keyB, []unsignedAuth{{1, keyC}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, doubleGasFee, uint256.NewInt(2), keyB, []unsignedAuth{{0, keyC}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
			},
		},
		{
			name:    "allow-tx-from-replaced-authority",
			pending: 2,
			run: func(name string) {
				// Fee bump with a different auth list. Make sure that unlocks the authorities.
				if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, minGasFee, uint256.NewInt(3), keyA, []unsignedAuth{{0, keyB}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, tripleGasFee, uint256.NewInt(300), keyA, []unsignedAuth{{0, keyC}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				// Now send a regular tx from B.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, minGasPrice, keyB)); err != nil {
					t.Fatalf("%s: failed to replace with remote transaction: %v", name, err)
				}
			},
		},
		{
			name:    "allow-tx-from-replaced-self-sponsor-authority",
			pending: 2,
			run: func(name string) {
				//
				if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, minGasFee, uint256.NewInt(3), keyA, []unsignedAuth{{0, keyA}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, tripleGasFee, uint256.NewInt(30), keyA, []unsignedAuth{{0, keyB}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				// Now send a regular tx from keyA.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, new(big.Int).Set(legacyReplacePrice), keyA)); err != nil {
					t.Fatalf("%s: failed to replace with remote transaction: %v", name, err)
				}
				// Make sure we can still send from keyB.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, minGasPrice, keyB)); err != nil {
					t.Fatalf("%s: failed to replace with remote transaction: %v", name, err)
				}
			},
		},
		{
			name:    "track-multiple-conflicting-delegations",
			pending: 3,
			run: func(name string) {
				// Send two setcode txs both with C as an authority.
				if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, minGasFee, uint256.NewInt(3), keyA, []unsignedAuth{{0, keyC}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				if err := pool.addRemoteSync(pricedSetCodeTx(0, 250000, tripleGasFee, uint256.NewInt(30), keyB, []unsignedAuth{{0, keyC}})); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				// Replace the tx from A with a non-setcode tx.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, new(big.Int).Set(legacyReplacePrice), keyA)); err != nil {
					t.Fatalf("%s: failed to replace with remote transaction: %v", name, err)
				}
				// Make sure we can only pool one tx from keyC since it is still a
				// pending authority.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, minGasPrice, keyC)); err != nil {
					t.Fatalf("%s: failed to added single pooled for account with pending delegation: %v", name, err)
				}
				if err, want := pool.addRemoteSync(pricedTransaction(1, 100000, minGasPrice, keyC)), txpool.ErrInflightTxLimitReached; !errors.Is(err, want) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, want, err)
				}
			},
		},
		{
			name:    "reject-delegation-from-pending-account",
			pending: 1,
			run: func(name string) {
				// Attempt to submit a delegation from an account with a pending tx.
				if err := pool.addRemoteSync(pricedTransaction(0, 100000, minGasPrice, keyC)); err != nil {
					t.Fatalf("%s: failed to add with remote setcode transaction: %v", name, err)
				}
				if err, want := pool.addRemoteSync(setCodeTx(0, keyA, []unsignedAuth{{1, keyC}})), ErrAuthorityReserved; !errors.Is(err, want) {
					t.Fatalf("%s: error mismatch: want %v, have %v", name, want, err)
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

	minGasFee := uint256.MustFromBig(common.MinGasPrice)
	doubleGasFee := new(uint256.Int).Mul(new(uint256.Int).Set(minGasFee), uint256.NewInt(2))
	legacyPrice := new(big.Int).Mul(new(big.Int).Set(common.MinGasPrice), big.NewInt(10))

	// Send an authorization for 0x42
	var authList []types.SetCodeAuthorization
	auth, _ := types.SignSetCode(keyA, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(params.TestChainConfig.ChainID),
		Address: common.Address{0x42},
		Nonce:   0,
	})
	authList = append(authList, auth)
	if err := pool.addRemoteSync(pricedSetCodeTxWithAuth(0, 250000, minGasFee, uint256.NewInt(3), keyA, authList)); err != nil {
		t.Fatalf("failed to add with remote setcode transaction: %v", err)
	}
	// Simulate the chain moving
	blockchain.statedb.SetNonce(addrA, 1)
	blockchain.statedb.SetCode(addrA, types.AddressToDelegation(auth.Address))
	<-pool.requestReset(nil, nil)
	// Set an authorization for 0x00
	auth, _ = types.SignSetCode(keyA, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(params.TestChainConfig.ChainID),
		Address: common.Address{},
		Nonce:   0,
	})
	authList = append(authList, auth)
	if err := pool.addRemoteSync(pricedSetCodeTxWithAuth(1, 250000, doubleGasFee, uint256.NewInt(3), keyA, authList)); err != nil {
		t.Fatalf("failed to add with remote setcode transaction: %v", err)
	}
	// Try to add a transactions in
	if err := pool.addRemoteSync(pricedTransaction(2, 100000, new(big.Int).Set(legacyPrice), keyA)); !errors.Is(err, txpool.ErrInflightTxLimitReached) {
		t.Fatalf("unexpected error %v, expecting %v", err, txpool.ErrInflightTxLimitReached)
	}
	// Simulate the chain moving
	blockchain.statedb.SetNonce(addrA, 2)
	blockchain.statedb.SetCode(addrA, nil)
	<-pool.requestReset(nil, nil)
	// Now send two transactions from addrA
	if err := pool.addRemoteSync(pricedTransaction(2, 100000, new(big.Int).Set(legacyPrice), keyA)); err != nil {
		t.Fatalf("failed to added single transaction: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(3, 100000, new(big.Int).Set(legacyPrice), keyA)); err != nil {
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
		pool.enqueueTx(tx.Hash(), tx, false, true)
	}
	// Benchmark the speed of pool validation
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.promoteExecutables(nil)
	}
}

// Benchmarks the speed of batched transaction insertion.
func BenchmarkBatchInsert100(b *testing.B)   { benchmarkBatchInsert(b, 100, false) }
func BenchmarkBatchInsert1000(b *testing.B)  { benchmarkBatchInsert(b, 1000, false) }
func BenchmarkBatchInsert10000(b *testing.B) { benchmarkBatchInsert(b, 10000, false) }

func BenchmarkBatchLocalInsert100(b *testing.B)   { benchmarkBatchInsert(b, 100, true) }
func BenchmarkBatchLocalInsert1000(b *testing.B)  { benchmarkBatchInsert(b, 1000, true) }
func BenchmarkBatchLocalInsert10000(b *testing.B) { benchmarkBatchInsert(b, 10000, true) }

func benchmarkBatchInsert(b *testing.B, size int, local bool) {
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
		if local {
			pool.addLocals(batch)
		} else {
			pool.AddRemotes(batch)
		}
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
		pool, _ := setupPool()
		testAddBalance(pool, account, big.NewInt(100000000))
		for _, local := range locals {
			pool.addLocal(local)
		}
		b.StartTimer()
		// Assign a high enough balance for testing
		testAddBalance(pool, remoteAddr, big.NewInt(100000000))
		for i := 0; i < len(remotes); i++ {
			pool.AddRemotes([]*types.Transaction{remotes[i]})
		}
		pool.Close()
	}
}

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
		pool.currentState.AddBalance(account, big.NewInt(1000000), tracing.BalanceChangeUnspecified)
		tx := transaction(uint64(0), 100000, key)
		batches[i] = tx
	}
	// Benchmark importing the transactions into the queue
	b.ResetTimer()
	for _, tx := range batches {
		pool.addRemotesSync([]*types.Transaction{tx})
	}
}

func TestPendingMinTipThreshold(t *testing.T) {
	t.Parallel()

	pool, key := setupPool()
	defer pool.Close()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, addr, big.NewInt(1_000_000_000_000_000))
	threshold := new(big.Int).Add(new(big.Int).Set(common.MinGasPrice), big.NewInt(100))
	aboveThreshold := new(big.Int).Set(threshold)
	belowThreshold := new(big.Int).Sub(new(big.Int).Set(threshold), big.NewInt(1))

	if err := pool.addRemoteSync(pricedTransaction(0, 100000, aboveThreshold, key)); err != nil {
		t.Fatalf("failed to add tx nonce 0: %v", err)
	}
	if err := pool.addRemoteSync(pricedTransaction(1, 100000, belowThreshold, key)); err != nil {
		t.Fatalf("failed to add tx nonce 1: %v", err)
	}

	filtered := pool.Pending(txpool.PendingFilter{MinTip: uint256.MustFromBig(threshold)})
	txs := filtered[addr]
	if len(txs) != 1 {
		t.Fatalf("unexpected filtered tx count: have %d, want %d", len(txs), 1)
	}
	resolved := txs[0].Resolve()
	if resolved == nil || resolved.Nonce() != 0 {
		have := uint64(math.MaxUint64)
		if resolved != nil {
			have = resolved.Nonce()
		}
		t.Fatalf("unexpected surviving tx after tip filter: got nonce %d", have)
	}

	all := pool.Pending(txpool.PendingFilter{})
	if len(all[addr]) != 2 {
		t.Fatalf("unexpected unfiltered tx count: have %d, want %d", len(all[addr]), 2)
	}
}

func TestPendingMinTipWithBaseFee(t *testing.T) {
	t.Parallel()

	pool, key := setupPoolWithConfig(eip1559Config)
	defer pool.Close()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, addr, big.NewInt(1_000_000_000_000_000))
	minGasTip := new(big.Int).Set(common.MinGasPrice)
	tipPass := new(big.Int).Add(new(big.Int).Set(minGasTip), big.NewInt(80))
	tipPassWithoutBaseFeeOnly := new(big.Int).Add(new(big.Int).Set(minGasTip), big.NewInt(60))

	if err := pool.addRemoteSync(dynamicFeeTx(0, 100000, new(big.Int).Add(new(big.Int).Set(minGasTip), big.NewInt(300)), tipPass, key)); err != nil {
		t.Fatalf("failed to add tx nonce 0: %v", err)
	}
	if err := pool.addRemoteSync(dynamicFeeTx(1, 100000, new(big.Int).Add(new(big.Int).Set(minGasTip), big.NewInt(240)), tipPassWithoutBaseFeeOnly, key)); err != nil {
		t.Fatalf("failed to add tx nonce 1: %v", err)
	}
	if err := pool.addRemoteSync(dynamicFeeTx(2, 100000, new(big.Int).Add(new(big.Int).Set(minGasTip), big.NewInt(240)), tipPassWithoutBaseFeeOnly, key)); err != nil {
		t.Fatalf("failed to add tx nonce 2: %v", err)
	}

	minTipBig := new(big.Int).Add(new(big.Int).Set(minGasTip), big.NewInt(45))
	baseFeeBig := big.NewInt(200)
	minTip := uint256.MustFromBig(minTipBig)
	baseFee := uint256.MustFromBig(baseFeeBig)

	withBaseFee := pool.Pending(txpool.PendingFilter{MinTip: minTip, BaseFee: baseFee})
	txs := withBaseFee[addr]
	if len(txs) != 1 {
		t.Fatalf("unexpected tx count with base fee filter: have %d, want %d", len(txs), 1)
	}
	resolved := txs[0].Resolve()
	if resolved == nil || resolved.Nonce() != 0 {
		have := uint64(math.MaxUint64)
		if resolved != nil {
			have = resolved.Nonce()
		}
		t.Fatalf("unexpected surviving tx with base fee filter: got nonce %d", have)
	}

	withoutBaseFee := pool.Pending(txpool.PendingFilter{MinTip: minTip})
	if len(withoutBaseFee[addr]) != 3 {
		t.Fatalf("unexpected tx count without base fee filter: have %d, want %d", len(withoutBaseFee[addr]), 3)
	}
}

func TestPendingKeepsLocalAndSpecialTransactions(t *testing.T) {
	t.Parallel()

	pool, _ := setupPool()
	defer pool.Close()
	minGasTip := new(big.Int).Set(common.MinGasPrice)
	filterTip := new(big.Int).Add(new(big.Int).Set(minGasTip), big.NewInt(100))

	specialKey, _ := crypto.GenerateKey()
	specialAddr := crypto.PubkeyToAddress(specialKey.PublicKey)
	testAddBalance(pool, specialAddr, big.NewInt(1_000_000_000_000_000))

	specialTx, _ := types.SignTx(types.NewTransaction(0, common.BlockSignersBinary, big.NewInt(1), 100000, new(big.Int).Add(new(big.Int).Set(minGasTip), big.NewInt(1)), nil), types.HomesteadSigner{}, specialKey)
	if err := pool.addRemoteSync(specialTx); err != nil {
		t.Fatalf("failed to add special tx: %v", err)
	}
	normalTx := pricedTransaction(1, 100000, new(big.Int).Add(new(big.Int).Set(minGasTip), big.NewInt(200)), specialKey)
	if err := pool.addRemoteSync(normalTx); err != nil {
		t.Fatalf("failed to add normal tx after special tx: %v", err)
	}

	localKey, _ := crypto.GenerateKey()
	localAddr := crypto.PubkeyToAddress(localKey.PublicKey)
	testAddBalance(pool, localAddr, big.NewInt(1_000_000_000_000_000))
	if err := pool.addLocal(pricedTransaction(0, 100000, new(big.Int).Add(new(big.Int).Set(minGasTip), big.NewInt(1)), localKey)); err != nil {
		t.Fatalf("failed to add local tx: %v", err)
	}

	filtered := pool.Pending(txpool.PendingFilter{MinTip: uint256.MustFromBig(filterTip)})

	specialPending := filtered[specialAddr]
	if len(specialPending) != 2 {
		t.Fatalf("unexpected special account tx count: have %d, want %d", len(specialPending), 2)
	}
	first := specialPending[0].Resolve()
	if first == nil || first.Nonce() != 0 || !first.IsSpecialTransaction() {
		t.Fatalf("special tx should survive filtering at nonce 0")
	}
	second := specialPending[1].Resolve()
	if second == nil || second.Nonce() != 1 {
		t.Fatalf("unexpected tx order for special account")
	}

	localPending := filtered[localAddr]
	if len(localPending) != 1 {
		t.Fatalf("unexpected local account tx count: have %d, want %d", len(localPending), 1)
	}
	local := localPending[0].Resolve()
	if local == nil || local.Nonce() != 0 {
		t.Fatalf("unexpected local tx after filtering")
	}
}

func TestPendingDynamicFeeThresholdWithoutBaseFee(t *testing.T) {
	t.Parallel()

	pool, key := setupPoolWithConfig(eip1559Config)
	defer pool.Close()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	testAddBalance(pool, addr, big.NewInt(1_000_000_000_000_000))

	minTipBig := new(big.Int).Add(new(big.Int).Set(common.MinGasPrice), big.NewInt(50))
	equalTip := new(big.Int).Set(minTipBig)
	belowTip := new(big.Int).Sub(new(big.Int).Set(minTipBig), big.NewInt(1))

	if err := pool.addRemoteSync(dynamicFeeTx(0, 100000, new(big.Int).Add(new(big.Int).Set(equalTip), big.NewInt(100)), equalTip, key)); err != nil {
		t.Fatalf("failed to add tx nonce 0: %v", err)
	}
	if err := pool.addRemoteSync(dynamicFeeTx(1, 100000, new(big.Int).Add(new(big.Int).Set(equalTip), big.NewInt(100)), belowTip, key)); err != nil {
		t.Fatalf("failed to add tx nonce 1: %v", err)
	}

	filtered := pool.Pending(txpool.PendingFilter{MinTip: uint256.MustFromBig(minTipBig)})
	txs := filtered[addr]
	if len(txs) != 1 {
		t.Fatalf("unexpected filtered tx count: have %d, want %d", len(txs), 1)
	}
	resolved := txs[0].Resolve()
	if resolved == nil || resolved.Nonce() != 0 {
		have := uint64(math.MaxUint64)
		if resolved != nil {
			have = resolved.Nonce()
		}
		t.Fatalf("unexpected surviving tx without base fee: got nonce %d", have)
	}

	all := pool.Pending(txpool.PendingFilter{})
	if len(all[addr]) != 2 {
		t.Fatalf("unexpected unfiltered tx count: have %d, want %d", len(all[addr]), 2)
	}
}

// TestSetGasPrice tests the SetGasPrice validation logic using table-driven tests
func TestSetGasPrice(t *testing.T) {
	testCases := []struct {
		name        string
		tip         *big.Int
		wantErr     error
		description string
	}{
		// Invalid cases - should be rejected
		{
			name:        "nil gas tip",
			tip:         nil,
			wantErr:     errors.New("reject nil gas tip"),
			description: "nil pointer should be rejected gracefully",
		},
		{
			name:        "negative gas tip",
			tip:         big.NewInt(-1),
			wantErr:     fmt.Errorf("reject negative gas tip: %v", big.NewInt(-1)),
			description: "negative value should be rejected",
		},
		{
			name:        "exceeds maximum by 1",
			tip:         new(big.Int).Add(defaultMaxTip, big.NewInt(1)),
			wantErr:     fmt.Errorf("reject too high gas tip: %v, maximum: %v", new(big.Int).Add(defaultMaxTip, big.NewInt(1)), defaultMaxTip),
			description: "value exceeding 1000 GWei should be rejected",
		},
		{
			name:        "exceeds maximum significantly",
			tip:         big.NewInt(10000 * params.GWei),
			wantErr:     fmt.Errorf("reject too high gas tip: %v, maximum: %v", big.NewInt(10000*params.GWei), defaultMaxTip),
			description: "value far exceeding maximum should be rejected",
		},
		// Valid cases - should be accepted
		{
			name:        "zero gas tip",
			tip:         big.NewInt(0),
			wantErr:     nil,
			description: "zero is valid as it's not negative",
		},
		{
			name:        "minimum positive value",
			tip:         big.NewInt(1),
			wantErr:     nil,
			description: "minimum positive value should be accepted",
		},
		{
			name:        "1 GWei",
			tip:         big.NewInt(params.GWei),
			wantErr:     nil,
			description: "1 GWei should be accepted",
		},
		{
			name:        "100 GWei",
			tip:         big.NewInt(100 * params.GWei),
			wantErr:     nil,
			description: "100 GWei should be accepted",
		},
		{
			name:        "500 GWei",
			tip:         big.NewInt(500 * params.GWei),
			wantErr:     nil,
			description: "500 GWei should be accepted",
		},
		{
			name:        "just below maximum",
			tip:         new(big.Int).Sub(defaultMaxTip, big.NewInt(1)),
			wantErr:     nil,
			description: "value just below maximum should be accepted",
		},
		{
			name:        "exactly at maximum",
			tip:         defaultMaxTip,
			wantErr:     nil,
			description: "exactly 1000 GWei should be accepted",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fresh pool instance for each test case to ensure isolation
			pool, _ := setupPool()
			defer pool.Close()

			oldPrice := pool.gasTip.Load()
			haveErr := pool.SetGasTip(tc.tip)
			newPrice := pool.gasTip.Load()

			if tc.wantErr != nil {
				// Invalid case: should return error and price should remain unchanged
				if haveErr == nil {
					t.Errorf("%s: Expected error %q, got nil", tc.description, tc.wantErr)
				} else if haveErr.Error() != tc.wantErr.Error() {
					t.Errorf("%s: Expected error %q, got %q", tc.description, tc.wantErr, haveErr)
				}
				if newPrice.Cmp(oldPrice) != 0 {
					t.Errorf("%s: Gas price should not change. Expected %v, got %v", tc.description, oldPrice, newPrice)
				}
			} else {
				// Valid case: should return nil error and price should be updated
				if haveErr != nil {
					t.Errorf("%s: Expected no error, got: %v", tc.description, haveErr)
				}
				if newPrice.ToBig().Cmp(tc.tip) != 0 {
					t.Errorf("%s: Expected gas price to be set to %v, got %v", tc.description, tc.tip, newPrice)
				}
			}
		})
	}
}
