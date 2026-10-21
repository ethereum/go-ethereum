// Copyright 2026 The go-ethereum Authors
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
	"fmt"
	"math/big"
	"math/rand"
	"slices"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// uncachedPending is Pending without the per-account cache: every
// transaction filtered and converted on every call.
func uncachedPending(pool *LegacyPool, filter txpool.PendingFilter) map[common.Address][]*txpool.LazyTransaction {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	pending := make(map[common.Address][]*txpool.LazyTransaction)
	for addr, list := range pool.pending {
		txs := list.Flatten()
		for i, tx := range txs {
			if filter.MinTip != nil && tx.EffectiveGasTipIntCmp(filter.MinTip, filter.BaseFee) < 0 {
				txs = txs[:i]
				break
			}
			if filter.GasLimitCap != 0 && tx.Gas() > filter.GasLimitCap {
				txs = txs[:i]
				break
			}
		}
		for _, tx := range txs {
			pending[addr] = append(pending[addr], &txpool.LazyTransaction{
				Pool:      pool,
				Hash:      tx.Hash(),
				Tx:        tx,
				Time:      tx.Time(),
				GasFeeCap: uint256.MustFromBig(tx.GasFeeCap()),
				GasTipCap: uint256.MustFromBig(tx.GasTipCap()),
				Gas:       tx.Gas(),
				BlobGas:   tx.BlobGas(),
			})
		}
	}
	return pending
}

func comparePending(got map[common.Address][]*txpool.LazyTransaction, count int, want map[common.Address][]*txpool.LazyTransaction) error {
	if len(got) != len(want) {
		return fmt.Errorf("%d accounts, want %d", len(got), len(want))
	}
	var total int
	for addr, wantTxs := range want {
		gotTxs := got[addr]
		if len(gotTxs) != len(wantTxs) {
			return fmt.Errorf("account %x: %d transactions, want %d", addr, len(gotTxs), len(wantTxs))
		}
		for i, w := range wantTxs {
			g := gotTxs[i]
			if g.Pool != w.Pool || g.Hash != w.Hash || g.Tx != w.Tx || !g.Time.Equal(w.Time) || !g.GasFeeCap.Eq(w.GasFeeCap) ||
				!g.GasTipCap.Eq(w.GasTipCap) || g.Gas != w.Gas || g.BlobGas != w.BlobGas {
				return fmt.Errorf("account %x, transaction %d: %+v, want %+v", addr, i, g, w)
			}
		}
		total += len(wantTxs)
	}
	if count != total {
		return fmt.Errorf("count %d, want %d", count, total)
	}
	return nil
}

// TestPendingCached checks that Pending, served from the per-account cache,
// returns what filtering and converting every transaction on each call
// returns, through random additions, replacements, removals, new heads, balance
// drops and evictions over the pool's limits.
func TestPendingCached(t *testing.T) {
	t.Parallel()

	config := testTxPoolConfig
	config.AccountSlots = 2
	config.GlobalSlots = 16
	config.AccountQueue = 8
	config.GlobalQueue = 40

	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	blockchain := newTestBlockChain(params.TestChainConfig, 10000000, statedb, new(event.Feed))
	pool := New(config, blockchain)
	if err := pool.Init(config.PriceLimit, blockchain.CurrentBlock(), newReserver()); err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	rng := rand.New(rand.NewSource(1))
	keys := make([]*ecdsa.PrivateKey, 16)
	addrs := make([]common.Address, len(keys))
	for i := range keys {
		keys[i], _ = crypto.GenerateKey()
		addrs[i] = crypto.PubkeyToAddress(keys[i].PublicKey)
		testAddBalance(pool, addrs[i], big.NewInt(1e18))
	}
	randomFilter := func() txpool.PendingFilter {
		var f txpool.PendingFilter
		if rng.Intn(4) > 0 {
			f.MinTip = uint256.NewInt(uint64(rng.Intn(12)))
		}
		if rng.Intn(4) > 0 {
			// on both sides of the fee caps (30 to 59)
			f.BaseFee = uint256.NewInt(uint64(rng.Intn(70)))
		}
		if rng.Intn(3) == 0 {
			f.GasLimitCap = uint64(21000 + rng.Intn(30000))
		}
		return f
	}
	check := func(step int) {
		t.Helper()
		filters := []txpool.PendingFilter{
			{}, {MinTip: uint256.NewInt(1)},
			{MinTip: uint256.NewInt(1), BaseFee: uint256.NewInt(45)}, // some fee caps below the base fee
			randomFilter(), randomFilter(), randomFilter(),
		}
		for _, f := range filters {
			var first map[common.Address][]*txpool.LazyTransaction
			for range 2 {
				got, count := pool.Pending(f)
				if err := comparePending(got, count, uncachedPending(pool, f)); err != nil {
					t.Fatalf("step %d, filter %+v: %v", step, f, err)
				}
				if first == nil {
					first = got
					continue
				}
				for addr, txs := range got {
					if txs[0] != first[addr][0] {
						t.Fatalf("step %d: account %x read again without a change, not served from the cache", step, addr)
					}
				}
			}
		}
		if err := validatePoolInternals(pool); err != nil {
			t.Fatalf("step %d: %v", step, err)
		}
	}
	for step := 0; step < 1000; step++ {
		k := rng.Intn(len(keys))
		addr := addrs[k]
		switch op := rng.Intn(20); {
		case op < 10: // a transaction: the next nonce, a gap or a replacement
			nonce := pool.Nonce(addr)
			switch rng.Intn(5) {
			case 0:
				nonce += uint64(1 + rng.Intn(2))
			case 1:
				if state := pool.currentState.GetNonce(addr); nonce > state {
					nonce = state + uint64(rng.Intn(int(nonce-state)))
				}
			}
			gas := uint64(21000 + rng.Intn(3)*10000)
			var tx *types.Transaction
			if rng.Intn(4) == 0 {
				tx = pricedTransaction(nonce, gas, big.NewInt(int64(30+rng.Intn(30))), keys[k])
			} else {
				tip := int64(1 + rng.Intn(15))
				tx = dynamicFeeTx(nonce, gas, big.NewInt(int64(30+rng.Intn(20))), big.NewInt(tip), keys[k])
			}
			pool.addRemotesSync([]*types.Transaction{tx})
		case op < 13: // removal of a pending or queued transaction
			pending, queued := pool.ContentFrom(addr)
			txs := append(pending, queued...)
			if len(txs) > 0 {
				pool.mu.Lock()
				pool.removeTx(txs[rng.Intn(len(txs))].Hash(), true, true)
				pool.mu.Unlock()
			}
		case op < 17: // a new head that includes a sender's leading transactions
			testSetNonce(pool, addr, pool.currentState.GetNonce(addr)+uint64(1+rng.Intn(2)))
			<-pool.requestReset(nil, nil)
		case op < 19: // a balance drop that makes some transactions unaffordable
			pool.mu.Lock()
			pool.currentState.SetBalance(addr, uint256.NewInt(uint64(rng.Intn(4))*600000), tracing.BalanceChangeUnspecified)
			pool.mu.Unlock()
			<-pool.requestReset(nil, nil)
		default: // the balance back
			testAddBalance(pool, addr, big.NewInt(1e18))
			<-pool.requestReset(nil, nil)
		}
		check(step)
	}
}

// TestPendingCachedConcurrent reads the pool from several goroutines while it
// changes, for the race detector.
func TestPendingCachedConcurrent(t *testing.T) {
	t.Parallel()

	pool, _ := setupPool()
	defer pool.Close()

	keys := make([]*ecdsa.PrivateKey, 8)
	for i := range keys {
		keys[i], _ = crypto.GenerateKey()
		testAddBalance(pool, crypto.PubkeyToAddress(keys[i].PublicKey), big.NewInt(1e18))
	}
	var (
		wg   sync.WaitGroup
		stop = make(chan struct{})
	)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			filter := txpool.PendingFilter{MinTip: uint256.NewInt(1), BaseFee: uint256.NewInt(2)}
			for {
				select {
				case <-stop:
					return
				default:
				}
				pending, _ := pool.Pending(filter)
				for _, txs := range pending {
					for _, ltx := range txs {
						_ = ltx.GasFeeCap.Uint64() + ltx.Gas
					}
				}
			}
		}()
	}
	for n := uint64(0); n < 50; n++ {
		for _, key := range keys {
			pool.addRemotesSync([]*types.Transaction{dynamicFeeTx(n, 21000, big.NewInt(10), big.NewInt(5), key)})
		}
		if n%10 == 9 {
			testSetNonce(pool, crypto.PubkeyToAddress(keys[0].PublicKey), n-5)
			<-pool.requestReset(nil, nil)
		}
	}
	close(stop)
	wg.Wait()
}

// TestPendingSnapshot checks that the lists Pending returned stay as they were
// while the pool changes afterwards: a replacement, a new head that includes an
// account's first transactions, a cap, a removal and Clear; and that appending
// to a returned list leaves the pool's cache alone.
func TestPendingSnapshot(t *testing.T) {
	t.Parallel()

	pool, _ := setupPool()
	defer pool.Close()

	keys := make([]*ecdsa.PrivateKey, 4)
	addrs := make([]common.Address, len(keys))
	for i := range keys {
		keys[i], _ = crypto.GenerateKey()
		addrs[i] = crypto.PubkeyToAddress(keys[i].PublicKey)
		testAddBalance(pool, addrs[i], big.NewInt(1e18))
		var txs []*types.Transaction
		for n := uint64(0); n < 4; n++ {
			txs = append(txs, dynamicFeeTx(n, 21000, big.NewInt(int64(100+10*i)+int64(n)), big.NewInt(int64(1+i)), keys[i]))
		}
		if errs := pool.addRemotesSync(txs); errs[0] != nil {
			t.Fatal(errs[0])
		}
	}
	filter := txpool.PendingFilter{MinTip: uint256.NewInt(1), BaseFee: uint256.NewInt(10)}
	snapshot, _ := pool.Pending(filter)
	type entry struct {
		hash           common.Hash
		tx             *types.Transaction
		feeCap, tipCap uint256.Int
		gas            uint64
	}
	want := make(map[common.Address][]entry)
	for addr, txs := range snapshot {
		for _, ltx := range txs {
			want[addr] = append(want[addr], entry{ltx.Hash, ltx.Tx, *ltx.GasFeeCap, *ltx.GasTipCap, ltx.Gas})
		}
	}
	if len(want) != len(keys) {
		t.Fatalf("%d accounts pending, want %d", len(want), len(keys))
	}
	check := func(stage string) {
		t.Helper()
		for addr, txs := range snapshot {
			if len(txs) != len(want[addr]) {
				t.Fatalf("after %s: account %x has %d transactions, had %d", stage, addr, len(txs), len(want[addr]))
			}
			for i, ltx := range txs {
				w := want[addr][i]
				if ltx.Hash != w.hash || ltx.Tx != w.tx || !ltx.GasFeeCap.Eq(&w.feeCap) || !ltx.GasTipCap.Eq(&w.tipCap) || ltx.Gas != w.gas {
					t.Fatalf("after %s: account %x, transaction %d changed", stage, addr, i)
				}
			}
		}
	}
	grown := append(snapshot[addrs[0]], &txpool.LazyTransaction{})
	again, _ := pool.Pending(filter)
	if len(grown) != 5 || len(again[addrs[0]]) != 4 || again[addrs[0]][0] != snapshot[addrs[0]][0] {
		t.Fatalf("an append to a returned list reached the pool: %d transactions read again, want 4", len(again[addrs[0]]))
	}
	check("an append")

	if errs := pool.addRemotesSync([]*types.Transaction{dynamicFeeTx(1, 21000, big.NewInt(1000), big.NewInt(500), keys[0])}); errs[0] != nil {
		t.Fatal(errs[0])
	}
	check("a replacement")

	testSetNonce(pool, addrs[1], 2)
	<-pool.requestReset(nil, nil)
	check("a new head")

	pool.mu.Lock()
	pool.pending[addrs[2]].Cap(2)
	pool.removeTx(want[addrs[3]][2].hash, true, true)
	pool.mu.Unlock()
	check("a cap and a removal")

	pool.Clear()
	check("Clear")
}

// TestSortedMapCaches checks that after each mutation of a SortedMap whose
// caches are populated, both the sorted-order cache and the lazy one match
// lists built afresh from its transactions.
func TestSortedMapCaches(t *testing.T) {
	key, _ := crypto.GenerateKey()
	tx := func(nonce uint64, fee int64) *types.Transaction {
		return dynamicFeeTx(nonce, 21000, big.NewInt(fee), big.NewInt(1), key)
	}
	for _, mutation := range []struct {
		name string
		do   func(m *SortedMap)
	}{
		{"a new nonce", func(m *SortedMap) { m.Put(tx(8, 100)) }},
		{"a replacement", func(m *SortedMap) { m.Put(tx(3, 200)) }},
		{"Forward, removing some", func(m *SortedMap) { m.Forward(3) }},
		{"Forward, removing one", func(m *SortedMap) { m.Forward(1) }},
		{"Forward, removing none", func(m *SortedMap) { m.Forward(0) }},
		{"Filter", func(m *SortedMap) { m.Filter(func(tx *types.Transaction) bool { return tx.Nonce()%2 == 0 }) }},
		{"filter and reheap", func(m *SortedMap) {
			m.filter(func(tx *types.Transaction) bool { return tx.Nonce() > 5 })
			m.reheap()
		}},
		{"Cap", func(m *SortedMap) { m.Cap(5) }},
		{"Remove", func(m *SortedMap) { m.Remove(4) }},
		{"Ready", func(m *SortedMap) { m.Ready(0) }},
		{"Ready up to a gap", func(m *SortedMap) { m.Remove(3); m.Ready(0) }},
	} {
		m := NewSortedMap()
		for n := uint64(0); n < 8; n++ {
			m.Put(tx(n, 100+int64(n)))
		}
		m.lazies(nil) // both caches populated
		mutation.do(m)

		sorted := make(types.Transactions, 0, len(m.items))
		for _, tx := range m.items {
			sorted = append(sorted, tx)
		}
		slices.SortFunc(sorted, func(a, b *types.Transaction) int { return int(a.Nonce()) - int(b.Nonce()) })
		if cache := m.flatten(); !slices.Equal(cache, sorted) {
			t.Errorf("after %s: sorted-order cache holds %d transactions, %d expected, or another order", mutation.name, len(cache), len(sorted))
		}
		got, want := m.lazies(nil), newLazyList(nil, sorted)
		if len(got.txs) != len(want.txs) {
			t.Fatalf("after %s: lazy cache holds %d transactions, want %d", mutation.name, len(got.txs), len(want.txs))
		}
		for i := range want.txs {
			g, w := got.txs[i], want.txs[i]
			if g.Tx != w.Tx || !g.GasFeeCap.Eq(w.GasFeeCap) || !g.GasTipCap.Eq(w.GasTipCap) || g.Gas != w.Gas || got.fees[i] != want.fees[i] {
				t.Fatalf("after %s: lazy cache's transaction %d differs from a list built afresh", mutation.name, i)
			}
		}
	}
}
