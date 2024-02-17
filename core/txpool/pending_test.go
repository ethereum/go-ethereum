// Copyright 2024 The go-ethereum Authors
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
	"container/heap"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/holiman/uint256"
)

func initLists() (heads TipList, tails map[common.Address][]*LazyTransaction) {
	tails = make(map[common.Address][]*LazyTransaction)

	for i := 0; i < 25; i++ {
		var (
			addr  = common.Address{byte(i)}
			tail  []*LazyTransaction
			first = true
		)
		for j := 0; j < 25; j++ {
			tip := uint256.NewInt(uint64(100*i + j))
			lazyTx := &LazyTransaction{
				Pool:    nil,
				Hash:    common.Hash{byte(i), byte(j)},
				Fees:    *tip,
				Gas:     uint64(i),
				BlobGas: uint64(j),
			}
			if first {
				first = false
				heads = append(heads, &TxTips{
					From: addr,
					Tips: lazyTx.Fees,
					Time: 0,
				})
			}
			tail = append(tail, lazyTx)
		}
		if len(tail) > 0 {
			tails[addr] = tail
		}
	}
	// un-sort the heads
	rand.Shuffle(len(heads), func(i, j int) {
		heads[i], heads[j] = heads[j], heads[i]
	})
	return heads, tails
}

// Test the sorting and the Shift operation
func TestPendingSortAndShift(t *testing.T) {
	// Create the pending-set
	var (
		heads, tails  = initLists()
		expectedCount = 25 * 25
		txset         = NewPendingSet(heads, tails)
		haveCount     = 0
		prevFee       = uint64(math.MaxInt64)
	)
	if txset.Empty() {
		t.Fatalf("expected non-empty")
	}
	for {
		ltx, fee := txset.Peek()
		if ltx == nil {
			break
		}
		haveCount++
		if fee.Cmp(&ltx.Fees) != 0 {
			t.Fatalf("error tx %d: %v != %v", haveCount, fee, ltx.Fees)
		}
		if fee.Uint64() > prevFee {
			t.Fatalf("tx %d: fee %d  > previous fee %d", haveCount, fee, prevFee)
		}
		txset.Shift()
	}
	if haveCount != expectedCount {
		t.Errorf("expected %d transactions, found %d", expectedCount, haveCount)
	}
}

// Test the sorting and the Pop operation
func TestPendingSortAndPop(t *testing.T) {
	var (
		heads, tails  = initLists()
		expectedCount = 25 * 1
		txset         = NewPendingSet(heads, tails)
		haveCount     = 0
		prevFee       = uint64(math.MaxInt64)
	)
	for {
		ltx, fee := txset.Peek()
		if ltx == nil {
			break
		}
		haveCount++
		if fee.Cmp(&ltx.Fees) != 0 {
			t.Fatalf("error tx %d: %v != %v", haveCount, fee, ltx.Fees)
		}
		if fee.Uint64() > prevFee {
			t.Fatalf("tx %d: fee %d  > previous fee %d", haveCount, fee, prevFee)
		}
		txset.Pop()
	}
	if haveCount != expectedCount {
		t.Errorf("expected %d transactions, found %d", expectedCount, haveCount)
	}
}

// Tests that if multiple transactions have the same price, the ones seen earlier
// are prioritized to avoid network spam attacks aiming for a specific ordering.
func TestSortingByTime(t *testing.T) {
	var heads TipList
	for i := 0; i < 25; i++ {
		addr := common.Address{byte(i)}
		heads = append(heads, &TxTips{
			From: addr,
			Tips: *(uint256.NewInt(uint64(100))),
			Time: int64(i),
		})
	}
	// un-sort the heads
	rand.Shuffle(len(heads), func(i, j int) {
		heads[i], heads[j] = heads[j], heads[i]
	})
	heap.Init(&heads)
	for want := int64(0); want < 25; want++ {
		obj := heap.Pop(&heads).(*TxTips)
		if have := obj.Time; have != want {
			t.Fatalf("have %d want %d", have, want)
		}
	}
}
