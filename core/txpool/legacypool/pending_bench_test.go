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
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// pendingShape is how many senders hold 1, 2, ... 16 pending transactions in a
// full 20,000-slot pool (--txpool.globalslots 20000): 20,531 transactions from
// 12,550 senders, most of them with one.
var pendingShape = []int{10736, 1113, 63, 34, 21, 26, 25, 32, 51, 50, 52, 58, 50, 56, 37, 146}

// pendingSizes scales pendingShape to geth's default of 5,120 pending slots
// (5,159 transactions from 3,140 senders) and to 20,000 slots.
var pendingSizes = []struct {
	name  string
	scale float64
}{{"5120", 0.25}, {"20000", 1}}

// pendingFilter is how the miner reads the pool.
var pendingFilter = txpool.PendingFilter{MinTip: uint256.NewInt(1), BaseFee: uint256.NewInt(1e9)}

// newFullPool returns a pool holding pendingShape scaled by scale, every
// transaction paying the base fee of pendingFilter, and the number of its
// pending transactions.
func newFullPool(scale float64) (*LegacyPool, int) {
	pool, _ := setupPool()

	var total, senders int
	for n, count := range pendingShape {
		for range int(math.Round(float64(count) * scale)) {
			key, _ := crypto.GenerateKey()
			account := crypto.PubkeyToAddress(key.PublicKey)
			testAddBalance(pool, account, big.NewInt(1e18))
			senders++
			s := int64(senders)
			for nonce := range uint64(n + 1) {
				tx := dynamicFeeTx(nonce, 100000, big.NewInt(2e9+s), big.NewInt(1e9+s), key)
				pool.promoteTx(account, tx.Hash(), tx)
				total++
			}
		}
	}
	return pool, total
}

// BenchmarkPending reads a full pool of each size under the miner's filter,
// every transaction passing it, with the CPU caches warm from the previous read
// (hot) or evicted before every read (cold).
func BenchmarkPending(b *testing.B) {
	flush := make([]byte, 256<<20)
	for _, size := range pendingSizes {
		pool, total := newFullPool(size.scale)
		if _, n := pool.Pending(pendingFilter); n != total {
			b.Fatalf("pending %d, want %d", n, total)
		}
		b.Run(size.name+"/hot", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				pool.Pending(pendingFilter)
			}
		})
		b.Run(size.name+"/cold", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				for i := 0; i < len(flush); i += 64 {
					flush[i]++
				}
				b.StartTimer()
				pool.Pending(pendingFilter)
			}
		})
		pool.Close()
	}
}

// BenchmarkPendingDemotionCached runs the pool's demotion pass after a new head
// that includes none of the pending transactions, on a full pool of each size
// whose lists have been read, so that their caches are populated.
func BenchmarkPendingDemotionCached(b *testing.B) {
	for _, size := range pendingSizes {
		pool, total := newFullPool(size.scale)
		if _, n := pool.Pending(pendingFilter); n != total {
			b.Fatalf("pending %d, want %d", n, total)
		}
		b.Run(size.name, func(b *testing.B) {
			pool.mu.Lock()
			defer pool.mu.Unlock()

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				pool.demoteUnexecutables()
			}
		})
		pool.Close()
	}
}
