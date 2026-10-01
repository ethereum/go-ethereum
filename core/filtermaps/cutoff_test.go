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

package filtermaps

import (
	"context"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestIndexerHistoryCutoff tests indexing a chain whose history before the cutoff
// is not available and no checkpoint is at or after the cutoff (the test chain
// matches no checkpoint list at all): the index starts at the cutoff with a local
// log value index base, survives a restart, follows the head through reorgs and
// drops tail epochs when the cutoff moves forward.
func TestIndexerHistoryCutoff(t *testing.T) {
	ts := newTestSetup(t)
	defer ts.close()

	ts.historyCutoff.Store(300)
	ts.chain.addBlocks(200, 5, 2, 4, true)
	ts.setHistory(0, false) // head before the cutoff: initialization waits
	ts.chain.addBlocks(800, 5, 2, 4, true)
	ts.fm.WaitIdle()
	ts.checkCutoffIndex(301, 1000)

	// restart with the same database
	ts.storeDbHash("before restart")
	ts.setHistory(0, false)
	ts.fm.WaitIdle()
	ts.checkDbHash("before restart")
	ts.checkCutoffIndex(301, 1000)

	// extend the head, then reorg near it
	ts.chain.addBlocks(100, 5, 2, 4, true)
	ts.fm.WaitIdle()
	ts.checkCutoffIndex(301, 1100)
	ts.chain.setHead(1050)
	ts.chain.addBlocks(80, 5, 2, 4, true)
	ts.fm.WaitIdle()
	ts.checkCutoffIndex(301, 1130)

	// a shorter log history drops tail epochs; restoring it renders them again
	// from the local base, giving the same database
	ts.storeDbHash("full")
	ts.setHistory(500, false)
	ts.fm.WaitIdle()
	if first := ts.fm.indexedRange.blocks.First(); first <= 301 || first > 631 {
		t.Fatalf("Invalid first indexed block with a log history of 500 blocks: %d", first)
	}
	ts.checkCutoffIndex(ts.fm.indexedRange.blocks.First(), 1130)
	ts.setHistory(0, false)
	ts.fm.WaitIdle()
	ts.checkCutoffIndex(301, 1130)
	ts.checkDbHash("full")

	// prune: the cutoff moves forward, the epochs starting before it go (at the
	// next head, like any tail change)
	ts.historyCutoff.Store(700)
	ts.chain.addBlocks(1, 5, 2, 4, true)
	ts.fm.WaitIdle()
	first := ts.fm.indexedRange.blocks.First()
	if first <= 700 || first > 800 {
		t.Fatalf("Invalid first indexed block after moving the cutoff to 700: %d", first)
	}
	ts.checkCutoffIndex(first, 1131)

	// a restart after the prune keeps the reduced index
	ts.setHistory(0, false)
	ts.fm.WaitIdle()
	ts.checkCutoffIndex(first, 1131)
}

// TestIndexerHistoryCutoffRandom changes the head of a chain with a history
// cutoff randomly between forks, checking the index after each change.
func TestIndexerHistoryCutoffRandom(t *testing.T) {
	ts := newTestSetup(t)
	defer ts.close()

	ts.historyCutoff.Store(250)
	forks := make([][]common.Hash, 8)
	ts.chain.addBlocks(600, 5, 2, 4, true)
	for i := range forks {
		if i != 0 {
			forkBlock := 300 + rand.Intn(300)
			ts.chain.setHead(forkBlock)
			ts.chain.addBlocks(600-forkBlock, 5, 2, 4, true)
		}
		forks[i] = ts.chain.getCanonicalChain()
	}
	ts.setHistory(0, false)
	ts.fm.WaitIdle()
	for i := 0; i < 100; i++ {
		head := 300 + rand.Intn(301)
		ts.chain.setCanonicalChain(forks[rand.Intn(len(forks))][:head+1])
		ts.fm.WaitIdle()
		ts.checkCutoffIndex(251, uint64(head))
	}
}

// checkCutoffIndex checks that the index has a local base, covers the given block
// range and that the matcher finds exactly the logs a scan of the receipts finds.
func (ts *testSetup) checkCutoffIndex(first, last uint64) {
	ts.t.Helper()
	fm := ts.fm
	if fm.disabled {
		ts.t.Fatalf("Log indexer disabled")
	}
	if !fm.indexedRange.initialized || !fm.indexedRange.localBase || !fm.indexedRange.headIndexed {
		ts.t.Fatalf("Unexpected index state (initialized: %v, local base: %v, head indexed: %v)",
			fm.indexedRange.initialized, fm.indexedRange.localBase, fm.indexedRange.headIndexed)
	}
	if fm.indexedRange.blocks.First() != first || fm.indexedRange.blocks.Last() != last {
		ts.t.Fatalf("Invalid indexed block range (expected %d-%d, got %d-%d)", first, last,
			fm.indexedRange.blocks.First(), fm.indexedRange.blocks.Last())
	}
	chain := ts.chain.getCanonicalChain()
	blockOf := make(map[*types.Log]uint64)
	var logs []*types.Log
	for n := first; n <= last; n++ {
		for _, receipt := range ts.chain.receipts[chain[n]] {
			for _, l := range receipt.Logs {
				blockOf[l] = n
				logs = append(logs, l)
			}
		}
	}
	if len(logs) == 0 {
		ts.t.Fatalf("No logs in blocks %d-%d", first, last)
	}
	for i := 0; i < 50; i++ {
		var (
			l          = logs[rand.Intn(len(logs))]
			from       = first + uint64(rand.Intn(int(last-first+1)))
			to         = from + uint64(rand.Intn(int(last-from+1)))
			addresses  = []common.Address{l.Address}
			topics     [][]common.Hash
			matchTopic = len(l.Topics) > 0 && rand.Intn(2) == 0
		)
		if matchTopic {
			addresses, topics = nil, [][]common.Hash{{l.Topics[0]}}
		}
		matches := func(l *types.Log) bool {
			if matchTopic {
				return len(l.Topics) > 0 && l.Topics[0] == topics[0][0]
			}
			return l.Address == addresses[0]
		}
		want := make(map[*types.Log]bool)
		for _, l := range logs {
			if n := blockOf[l]; n >= from && n <= to && matches(l) {
				want[l] = true
			}
		}
		mb := fm.NewMatcherBackend()
		potential, err := GetPotentialMatches(context.Background(), mb, from, to, addresses, topics)
		mb.Close()
		if err != nil {
			ts.t.Fatalf("Log search error: %v", err)
		}
		got := make(map[*types.Log]bool)
		for _, l := range potential {
			if n, ok := blockOf[l]; ok && n >= from && n <= to && matches(l) {
				got[l] = true
			}
		}
		if len(got) != len(want) {
			ts.t.Fatalf("Log search in blocks %d-%d returned %d matching logs, expected %d", from, to, len(got), len(want))
		}
		for l := range want {
			if !got[l] {
				ts.t.Fatalf("Log search in blocks %d-%d did not return a matching log of block %d", from, to, blockOf[l])
			}
		}
	}
}
