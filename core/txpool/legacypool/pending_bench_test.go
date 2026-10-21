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
	"math/big"
	"runtime"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/txorder"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// BenchmarkPending measures what a block build pays to read the pending
// transactions of a full pool: 1250 senders with 16 transactions each, about
// what a mainnet node holds with --txpool.globalslots 20000, filtered by a
// minimum tip against a base fee as the miner asks for them.
func BenchmarkPending(b *testing.B) {
	pool, _ := setupPool()
	defer pool.Close()

	for s := 0; s < 1250; s++ {
		key, _ := crypto.GenerateKey()
		account := crypto.PubkeyToAddress(key.PublicKey)
		testAddBalance(pool, account, big.NewInt(1e18))
		for n := uint64(0); n < 16; n++ {
			tx := dynamicFeeTx(n, 100000, big.NewInt(int64(2e9+s)), big.NewInt(int64(1e9+s)), key)
			pool.promoteTx(account, tx.Hash(), tx)
		}
	}
	filter := txpool.PendingFilter{MinTip: uint256.NewInt(1), BaseFee: uint256.NewInt(1e9)}
	if _, n := pool.Pending(filter); n != 1250*16 {
		b.Fatalf("pending %d, want %d", n, 1250*16)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Pending(filter)
	}
}

// livePendingShape is how many senders hold 1, 2, ... 16 pending transactions
// in a mainnet node's full pool (--txpool.globalslots 20000, October 2026):
// 20,531 transactions from 12,550 senders, most of them with one.
var livePendingShape = []int{10736, 1113, 63, 34, 21, 26, 25, 32, 51, 50, 52, 58, 50, 56, 37, 146}

// newLivePool returns a pool shaped like a live mainnet node's
// (livePendingShape), every sender paying the base fee of liveFilter, and the
// number of its pending transactions.
func newLivePool() (*LegacyPool, int) {
	pool, _ := setupPool()

	var (
		total int
		keys  []*ecdsa.PrivateKey
	)
	for n, senders := range livePendingShape {
		for range senders {
			key, _ := crypto.GenerateKey()
			keys = append(keys, key)
			account := crypto.PubkeyToAddress(key.PublicKey)
			testAddBalance(pool, account, big.NewInt(1e18))
			for nonce := range uint64(n + 1) {
				s := int64(len(keys))
				tx := dynamicFeeTx(nonce, 100000, big.NewInt(2e9+s), big.NewInt(1e9+s), key)
				pool.promoteTx(account, tx.Hash(), tx)
				total++
			}
		}
	}
	return pool, total
}

// liveFilter is how the miner reads the pool of newLivePool.
var liveFilter = txpool.PendingFilter{MinTip: uint256.NewInt(1), BaseFee: uint256.NewInt(1e9)}

// BenchmarkPendingLive reads a pool shaped like a live mainnet node's, with
// the CPU caches warm from the previous read (hot) or evicted before every
// read (cold), as a live node's other work between builds leaves them.
func BenchmarkPendingLive(b *testing.B) {
	pool, total := newLivePool()
	defer pool.Close()

	filter := liveFilter
	if _, n := pool.Pending(filter); n != total {
		b.Fatalf("pending %d, want %d", n, total)
	}
	b.Run("hot", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pool.Pending(filter)
		}
	})
	b.Run("cold", func(b *testing.B) {
		flush := make([]byte, 256<<20)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			for i := 0; i < len(flush); i += 64 {
				flush[i]++
			}
			b.StartTimer()
			pool.Pending(filter)
		}
	})
}

// BenchmarkDemotionLive runs the pool's demotion pass after a new head that
// includes none of the pending transactions, the common case for an account
// between two of its inclusions, on a pool shaped like a live mainnet node's
// whose lists have been read, so that their caches are populated.
func BenchmarkDemotionLive(b *testing.B) {
	pool, total := newLivePool()
	defer pool.Close()

	if _, n := pool.Pending(liveFilter); n != total {
		b.Fatalf("pending %d, want %d", n, total)
	}
	pool.mu.Lock()
	defer pool.mu.Unlock()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.demoteUnexecutables()
	}
}

// BenchmarkPendingPayableLive reads a pool shaped like a live mainnet node's,
// where few senders can pay the base fee (one in 64; live, 150-350 of 20,000
// transactions in October 2026, the rest waiting for it to drop), under the
// miner's filter, alone and with the miner's price ordering built on the
// result, with the CPU caches warm or evicted before every read. As live, the
// senders of two transactions pay ten times more in the second, above the base
// fee: it waits for the first.
func BenchmarkPendingPayableLive(b *testing.B) {
	pool, _ := setupPool()
	defer pool.Close()

	var (
		payable int
		keys    []*ecdsa.PrivateKey
	)
	for n, senders := range livePendingShape {
		for range senders {
			key, _ := crypto.GenerateKey()
			keys = append(keys, key)
			account := crypto.PubkeyToAddress(key.PublicKey)
			testAddBalance(pool, account, big.NewInt(1e18))
			s := int64(len(keys))
			feeCap := 3e8 + s*6e8/20000 // 0.3-0.65 of the base fee
			if s%64 == 0 {
				feeCap = 2e9 + s
				payable += n + 1
			}
			if n == 1 && feeCap < 1e9 {
				feeCap /= 2 // 0.15-0.33 of the base fee, the second 1.5-3.3 times it
			}
			for nonce := range uint64(n + 1) {
				tx := dynamicFeeTx(nonce, 100000, big.NewInt(feeCap), big.NewInt(1e8+s), key)
				pool.all.Add(tx)
				pool.promoteTx(account, tx.Hash(), tx)
				if n == 1 {
					feeCap *= 10
				}
			}
		}
	}
	var (
		baseFee = big.NewInt(1e9)
		signer  = types.LatestSigner(pool.chainconfig)
		filter  = txpool.PendingFilter{MinTip: uint256.NewInt(1), BaseFee: uint256.MustFromBig(baseFee)}
	)
	if _, n := pool.Pending(filter); n != payable {
		b.Fatalf("pending %d, want %d", n, payable)
	}
	pool.Pending(txpool.PendingFilter{}) // every account's list cached
	flush := make([]byte, 256<<20)
	for _, order := range []bool{false, true} {
		for _, cold := range []bool{false, true} {
			name := "read"
			if order {
				name += "+order"
			}
			name += map[bool]string{false: "/hot", true: "/cold"}[cold]
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if cold {
						b.StopTimer()
						for i := 0; i < len(flush); i += 64 {
							flush[i]++
						}
						b.StartTimer()
					}
					txs, _ := pool.Pending(filter)
					if order {
						txorder.NewTransactionsByPriceAndNonce(signer, txs, baseFee)
					}
				}
			})
		}
	}
}

// BenchmarkPromotion moves executable transactions from the queue to the
// pending lists, as when the transactions in between arrive: 1000 senders with
// four pending transactions each, their lists read so that their caches are
// populated, receive four more each that were queued.
func BenchmarkPromotion(b *testing.B) {
	pool, _ := setupPool()
	defer pool.Close()

	const senders, pending, queued = 1000, 4, 4
	var (
		accounts []common.Address
		txs      [][]*types.Transaction
	)
	for range senders {
		key, _ := crypto.GenerateKey()
		account := crypto.PubkeyToAddress(key.PublicKey)
		testAddBalance(pool, account, big.NewInt(1e18))
		accounts = append(accounts, account)
		var list []*types.Transaction
		for nonce := range uint64(pending + queued) {
			list = append(list, dynamicFeeTx(nonce, 100000, big.NewInt(2e9), big.NewInt(1e9), key))
		}
		txs = append(txs, list)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pool.Clear()
		pool.mu.Lock()
		for s, account := range accounts {
			if err := pool.reserver.Hold(account); err != nil { // as add does; Clear released it
				b.Fatal(err)
			}
			for _, tx := range txs[s][:pending] {
				pool.all.Add(tx)
				pool.promoteTx(account, tx.Hash(), tx)
			}
			for _, tx := range txs[s][pending:] {
				pool.enqueueTx(tx.Hash(), tx, true)
			}
		}
		pool.mu.Unlock()
		pool.Pending(liveFilter) // every pending list's cache populated
		runtime.GC()             // the setup's garbage collected outside the timing
		pool.mu.Lock()
		b.StartTimer()
		promoted := pool.promoteExecutables(accounts)
		b.StopTimer()
		pool.mu.Unlock()
		if len(promoted) != senders*queued {
			b.Fatalf("promoted %d, want %d", len(promoted), senders*queued)
		}
		b.StartTimer()
	}
}
