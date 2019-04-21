// Copyright 2016 The go-ethereum Authors
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

package light

import (
	"context"
	"errors"
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

type testTxRelay struct {
	send    chan []*types.Transaction
	discard chan []common.Hash

	sendHook func() error
}

func (t *testTxRelay) Send(txs types.Transactions) error {
	t.send <- txs

	if t.sendHook != nil {
		return t.sendHook()
	}
	return nil
}

func (t *testTxRelay) Discard(hashes []common.Hash) {
	t.discard <- hashes
}

const (
	poolTestTxs    = 1000
	poolTestBlocks = 100
)

// test tx 0..n-1
var testTx [poolTestTxs]*types.Transaction

// txs sent before block i
func sentTx(i int) int {
	return int(math.Pow(float64(i)/float64(poolTestBlocks), 0.9) * poolTestTxs)
}

// txs included in block i or before that (minedTx(i) <= sentTx(i))
func minedTx(i int) int {
	return int(math.Pow(float64(i)/float64(poolTestBlocks), 1.1) * poolTestTxs)
}

func txPoolTestChainGen(i int, block *core.BlockGen) {
	low, high := minedTx(i), minedTx(i+1)
	for i := low; i < high; i++ {
		block.AddTx(testTx[i])
	}
}

func TestTxPool(t *testing.T) {
	var (
		serverDB = rawdb.NewMemoryDatabase()
		clientDB = rawdb.NewMemoryDatabase()
		gspec    = core.Genesis{Alloc: core.GenesisAlloc{testBankAddress: {Balance: testBankFunds}}}
		genesis  = gspec.MustCommit(serverDB)
		txmap    = make(map[common.Hash]int)
	)
	// Initialize transactions
	for i := range testTx {
		testTx[i], _ = types.SignTx(types.NewTransaction(uint64(i), acc1Addr, big.NewInt(10000), params.TxGas, nil, nil), types.HomesteadSigner{}, testBankKey)
		txmap[testTx[i].Hash()] = i
	}
	// Initialize server side.
	blockchain, _ := core.NewBlockChain(serverDB, nil, params.TestChainConfig, ethash.NewFullFaker(), vm.Config{}, nil)
	gchain, _ := core.GenerateChain(params.TestChainConfig, genesis, ethash.NewFaker(), serverDB, poolTestBlocks, txPoolTestChainGen)
	if _, err := blockchain.InsertChain(gchain); err != nil {
		panic(err)
	}
	// Initialize client side.
	relay := &testTxRelay{
		send:    make(chan []*types.Transaction, 1),
		discard: make(chan []common.Hash, 1),
	}
	relayCh, discardCh := make(chan error, 1), make(chan error, 1)
	go func() {
		received, discarded := make(map[common.Hash]struct{}), make(map[common.Hash]struct{})
		for {
			select {
			case txs := <-relay.send:
				for _, tx := range txs {
					if _, exist := txmap[tx.Hash()]; !exist {
						relayCh <- errors.New("unexpected transaction")
					}
					received[tx.Hash()] = struct{}{}
				}
				if len(received) == len(testTx) {
					relayCh <- nil
				}
			case hashes := <-relay.discard:
				for _, h := range hashes {
					if _, exist := txmap[h]; !exist {
						discardCh <- errors.New("unexpected transaction")
					}
					discarded[h] = struct{}{}
				}
				if len(discarded) == len(testTx) {
					discardCh <- nil
				}
			}
		}
	}()

	gspec.MustCommit(clientDB)
	odr := &testOdr{sdb: serverDB, ldb: clientDB, indexerConfig: TestClientIndexerConfig}

	// Register some hooks for various attack testing
	var statusLock sync.Mutex
	statusMark := make(map[int]bool)
	statusCounter := make(map[int]int32)
	odr.isStatusHookTarget = func(hash common.Hash) bool {
		statusLock.Lock()
		defer statusLock.Unlock()

		txIndex := txmap[hash]

		var target bool
		if txIndex >= 20 && txIndex <= 50 {
			if !statusMark[txIndex] {
				statusMark[txIndex] = true
				target = true
			}
		} else if txIndex >= 80 && txIndex <= 100 {
			if statusCounter[txIndex] <= 10 {
				statusCounter[txIndex] += 1
				target = true
			}
		}
		return target
	}
	odr.StatusHook = func(hash common.Hash) TxStatus {
		txIndex := txmap[hash]

		if txIndex >= 20 && txIndex <= 50 {
			// Respond with a fake status information, expect client
			// can recover from this attack.
			return TxStatus{Status: core.TxStatusIncluded, Lookup: &rawdb.LegacyTxLookupEntry{BlockIndex: 0, BlockHash: common.HexToHash("deadbeef")}}
		} else {
			// Always respond with unknown status for the first 10 requests,
			// force light client to resend transaction.
			return TxStatus{Status: core.TxStatusUnknown}
		}
	}
	var fakeBlock int32
	odr.isBlockHookTarget = func(hash common.Hash) bool {
		if hash == gchain[50].Hash() && atomic.LoadInt32(&fakeBlock) <= 5 {
			atomic.AddInt32(&fakeBlock, 1)
			return true
		}
		return false
	}
	odr.blockHook = func(hash common.Hash) []byte {
		return []byte{0x00, 0x01, 0x02}
	}

	lightchain, _ := NewLightChain(odr, params.TestChainConfig, ethash.NewFullFaker())
	txPermanent, statusQueryResendDelay = 50, 10*time.Millisecond

	pool := NewTxPool(core.TxPoolConfig{}, params.TestChainConfig, lightchain, relay)
	defer pool.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	tail, _ := core.GenerateChain(params.TestChainConfig, gchain[len(gchain)-1], ethash.NewFaker(), serverDB, int(txPermanent), nil)

	for i, block := range append(gchain, tail...) {
		low, high := sentTx(i), sentTx(i+1)
		if high <= len(testTx) && low < high {
			pool.AddBatch(ctx, testTx[low:high])
		}
		if _, err := lightchain.InsertHeaderChain([]*types.Header{block.Header()}, 1); err != nil {
			panic(err)
		}
		time.Sleep(10 * time.Millisecond) // Give mainloop enough time to process all events.
	}
	for _, ch := range []chan error{relayCh, discardCh} {
		select {
		case err := <-ch:
			if err != nil {
				t.Fatalf("Unexpeated error %v", err)
			}
		case <-time.NewTimer(5 * time.Second).C:
			t.Fatalf("timeout")
		}
	}
	// Check the integrity in database.
	for _, tx := range testTx {
		dbtx, _, _, _ := rawdb.ReadTransaction(clientDB, tx.Hash())
		if dbtx == nil {
			t.Fatalf("Transaction %v(index=%d) not found, expect to find in the database", tx.Hash().Hex(), txmap[tx.Hash()])
		}
		receipt, _, _, _ := rawdb.ReadReceipt(clientDB, tx.Hash(), lightchain.Config())
		if receipt == nil {
			t.Fatalf("Receipt %v not found, expect to find in the database", tx.Hash().Hex())
		}
	}
}

func TestResend(t *testing.T) {
	var (
		serverDB = rawdb.NewMemoryDatabase()
		clientDB = rawdb.NewMemoryDatabase()
		gspec    = core.Genesis{Alloc: core.GenesisAlloc{testBankAddress: {Balance: testBankFunds}}}
	)
	// Initialize transactions
	for i := uint64(0); i < 3; i++ {
		testTx[i], _ = types.SignTx(types.NewTransaction(i, acc1Addr, big.NewInt(10000), params.TxGas, nil, nil), types.HomesteadSigner{}, testBankKey)
	}
	// Initialize client side.
	relay := &testTxRelay{
		send:    make(chan []*types.Transaction, 1),
		discard: make(chan []common.Hash, 1),
	}
	// Disable relay functionality.
	relay.sendHook = func() error {
		return errors.New("test error")
	}
	go func() {
		for {
			select {
			case <-relay.send:
			case <-relay.discard:
			}
		}
	}()

	gspec.MustCommit(clientDB)
	odr := &testOdr{sdb: serverDB, ldb: clientDB, indexerConfig: TestClientIndexerConfig}
	lightchain, _ := NewLightChain(odr, params.TestChainConfig, ethash.NewFullFaker())

	pool := NewTxPool(core.TxPoolConfig{}, params.TestChainConfig, lightchain, relay)
	defer pool.Stop()

	pool.AddBatch(context.Background(), testTx[:3])

	// Resend the transaction with higher transfer value.
	for i := uint64(0); i < 3; i++ {
		newtx, _ := types.SignTx(types.NewTransaction(i, acc1Addr, big.NewInt(20000), params.TxGas, nil, nil), types.HomesteadSigner{}, testBankKey)
		pool.Add(context.Background(), newtx)

		p, _ := pool.GetPendingTransaction(newtx.Hash())
		if p == nil {
			t.Fatalf("new transaction should be included")
		}
		p, _ = pool.GetPendingTransaction(testTx[i].Hash())
		if p != nil {
			t.Fatalf("old transaction should be discarded")
		}
	}
}

func TestInvalidTransaction(t *testing.T) {
	var (
		serverDB = rawdb.NewMemoryDatabase()
		clientDB = rawdb.NewMemoryDatabase()
		gspec    = core.Genesis{Alloc: core.GenesisAlloc{testBankAddress: {Balance: testBankFunds}}}
	)
	// Initialize client side.
	relay := &testTxRelay{
		send:    make(chan []*types.Transaction, 1),
		discard: make(chan []common.Hash, 1),
	}
	go func() {
		for {
			select {
			case <-relay.send:
			case <-relay.discard:
			}
		}
	}()

	gspec.MustCommit(clientDB)
	odr := &testOdr{sdb: serverDB, ldb: clientDB, indexerConfig: TestClientIndexerConfig}
	lightchain, _ := NewLightChain(odr, params.TestChainConfig, ethash.NewFullFaker())

	pool := NewTxPool(core.TxPoolConfig{}, params.TestChainConfig, lightchain, relay)
	defer pool.Stop()

	// Duplicated transaction
	tx, _ := types.SignTx(types.NewTransaction(0, acc1Addr, big.NewInt(10000), params.TxGas, nil, nil), types.HomesteadSigner{}, testBankKey)
	pool.Add(context.Background(), tx)
	if err := pool.Add(context.Background(), tx); err != errDuplicatedTransaction {
		t.Fatalf("duplicated transaction expected, %v", err)
	}
	// Invalid nonce
	tx, _ = types.SignTx(types.NewTransaction(2, acc1Addr, big.NewInt(10000), params.TxGas, nil, nil), types.HomesteadSigner{}, testBankKey)
	if err := pool.Add(context.Background(), tx); err != errInvalidNonce {
		t.Fatalf("invalid nonce error expected, %v", err)
	}
	// Not enough balance
	tx, _ = types.SignTx(types.NewTransaction(0, acc1Addr, big.NewInt(10000), params.TxGas, nil, nil), types.HomesteadSigner{}, acc2Key)
	if err := pool.Add(context.Background(), tx); err != core.ErrInsufficientFunds {
		t.Fatalf("insufficient funds error expected, %v", err)
	}
}

func TestPendingNonce(t *testing.T) {
	var (
		serverDB = rawdb.NewMemoryDatabase()
		clientDB = rawdb.NewMemoryDatabase()
		gspec    = core.Genesis{Alloc: core.GenesisAlloc{testBankAddress: {Balance: testBankFunds}}}
	)
	// Initialize transactions
	for i := range testTx {
		testTx[i], _ = types.SignTx(types.NewTransaction(uint64(i), acc1Addr, big.NewInt(10000), params.TxGas, nil, nil), types.HomesteadSigner{}, testBankKey)
	}
	// Initialize client side.
	relay := &testTxRelay{
		send:    make(chan []*types.Transaction, 1),
		discard: make(chan []common.Hash, 1),
	}
	closeCh := make(chan struct{})
	go func() {
		for {
			select {
			case <-relay.send:
			case <-relay.discard:
			case <-closeCh:
			}
		}
	}()

	gspec.MustCommit(clientDB)
	odr := &testOdr{sdb: serverDB, ldb: clientDB, indexerConfig: TestClientIndexerConfig}
	lightchain, _ := NewLightChain(odr, params.TestChainConfig, ethash.NewFullFaker())

	pool := NewTxPool(core.TxPoolConfig{}, params.TestChainConfig, lightchain, relay)
	defer pool.Stop()

	pool.AddBatch(context.Background(), testTx[:])

	nonce, err := pool.GetNonce(context.Background(), testBankAddress)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	if nonce != uint64(len(testTx)) {
		t.Fatalf("nonce mismatch, want %v, have %v", len(testTx), nonce)
	}
}

func TestPendingTransaction(t *testing.T) {
	var (
		serverDB = rawdb.NewMemoryDatabase()
		clientDB = rawdb.NewMemoryDatabase()
		gspec    = core.Genesis{Alloc: core.GenesisAlloc{testBankAddress: {Balance: testBankFunds}}}
	)
	// Initialize transactions
	for i := range testTx[:5] {
		testTx[i], _ = types.SignTx(types.NewTransaction(uint64(i), acc1Addr, big.NewInt(10000), params.TxGas, nil, nil), types.HomesteadSigner{}, testBankKey)
	}
	// Initialize client side.
	relay := &testTxRelay{
		send:    make(chan []*types.Transaction, 1),
		discard: make(chan []common.Hash, 1),
	}
	relay.sendHook = func() error {
		return errors.New("reject relay transaction")
	}
	closeCh := make(chan struct{})
	go func() {
		for {
			select {
			case <-relay.send:
			case <-relay.discard:
			case <-closeCh:
			}
		}
	}()

	gspec.MustCommit(clientDB)
	odr := &testOdr{sdb: serverDB, ldb: clientDB, indexerConfig: TestClientIndexerConfig}
	lightchain, _ := NewLightChain(odr, params.TestChainConfig, ethash.NewFullFaker())

	pool := NewTxPool(core.TxPoolConfig{}, params.TestChainConfig, lightchain, relay)
	defer pool.Stop()

	pool.AddBatch(context.Background(), testTx[:5])

	txs, _ := pool.GetAllPendingTransactions()
	if len(txs) != 5 {
		t.Fatalf("Pending transaction number mismatch, want %d, have %d", 5, len(txs))
	}

	tx, _ := pool.GetPendingTransaction(testTx[0].Hash())
	if tx == nil {
		t.Fatalf("Expect pending transaction exists, but not found")
	}
	if tx.Hash() != testTx[0].Hash() {
		t.Fatalf("Pending transaction mismatch, want %s, have %s", testTx[0].Hash().Hex(), tx.Hash().Hex())
	}
}
