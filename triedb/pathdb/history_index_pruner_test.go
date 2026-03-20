// Copyright 2025 The go-ethereum Authors
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

package pathdb

import (
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

func writeMultiBlockIndex(t *testing.T, db ethdb.Database, ident stateIdent, bitmapSize int, startID uint64) []*indexBlockDesc {
	t.Helper()

	if startID == 0 {
		startID = 1
	}
	iw, _ := newIndexWriter(db, ident, 0, bitmapSize)

	for i := 0; i < 10000; i++ {
		if err := iw.append(startID+uint64(i), randomExt(bitmapSize, 5)); err != nil {
			t.Fatalf("Failed to append element %d: %v", i, err)
		}
	}
	batch := db.NewBatch()
	iw.finish(batch)
	if err := batch.Write(); err != nil {
		t.Fatalf("Failed to write batch: %v", err)
	}

	blob := readStateIndex(ident, db)
	descList, err := parseIndex(blob, bitmapSize)
	if err != nil {
		t.Fatalf("Failed to parse index: %v", err)
	}
	return descList
}

// TestPruneEntryBasic verifies that pruneEntry correctly removes leading index
// blocks whose max is below the given tail.
func TestPruneEntryBasic(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	ident := newAccountIdent(common.Hash{0xa})
	descList := writeMultiBlockIndex(t, db, ident, 0, 1)

	// Prune with a tail that is above the first block's max but below the second
	firstBlockMax := descList[0].max

	pruner := newIndexPruner(db, typeStateHistory)
	defer pruner.close()

	if err := pruner.process(firstBlockMax + 1); err != nil {
		t.Fatalf("Failed to process pruning: %v", err)
	}

	// Verify the first block was removed
	blob := readStateIndex(ident, db)
	if len(blob) == 0 {
		t.Fatal("Index metadata should not be empty after partial prune")
	}
	remaining, err := parseIndex(blob, 0)
	if err != nil {
		t.Fatalf("Failed to parse index after prune: %v", err)
	}
	if len(remaining) != len(descList)-1 {
		t.Fatalf("Expected %d blocks remaining, got %d", len(descList)-1, len(remaining))
	}
	// The first remaining block should be what was previously the second block
	if remaining[0].id != descList[1].id {
		t.Fatalf("Expected first remaining block id %d, got %d", descList[1].id, remaining[0].id)
	}

	// Verify the pruned block data is actually deleted
	blockData := readStateIndexBlock(ident, db, descList[0].id)
	if len(blockData) != 0 {
		t.Fatal("Pruned block data should have been deleted")
	}

	// Remaining blocks should still have their data
	for _, desc := range remaining {
		blockData = readStateIndexBlock(ident, db, desc.id)
		if len(blockData) == 0 {
			t.Fatalf("Block %d data should still exist", desc.id)
		}
	}
}

// TestPruneEntryBasicTrienode is the same as TestPruneEntryBasic but for
// trienode index entries with a non-zero bitmapSize.
func TestPruneEntryBasicTrienode(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	addrHash := common.Hash{0xa}
	path := string([]byte{0x0, 0x0, 0x0})
	ident := newTrienodeIdent(addrHash, path)

	descList := writeMultiBlockIndex(t, db, ident, ident.bloomSize(), 1)
	firstBlockMax := descList[0].max

	pruner := newIndexPruner(db, typeTrienodeHistory)
	defer pruner.close()

	if err := pruner.process(firstBlockMax + 1); err != nil {
		t.Fatalf("Failed to process pruning: %v", err)
	}

	blob := readStateIndex(ident, db)
	remaining, err := parseIndex(blob, ident.bloomSize())
	if err != nil {
		t.Fatalf("Failed to parse index after prune: %v", err)
	}
	if len(remaining) != len(descList)-1 {
		t.Fatalf("Expected %d blocks remaining, got %d", len(descList)-1, len(remaining))
	}
	if remaining[0].id != descList[1].id {
		t.Fatalf("Expected first remaining block id %d, got %d", descList[1].id, remaining[0].id)
	}
	blockData := readStateIndexBlock(ident, db, descList[0].id)
	if len(blockData) != 0 {
		t.Fatal("Pruned block data should have been deleted")
	}
}

// TestPruneEntryComplete verifies that when all blocks are pruned, the metadata
// entry is also deleted.
func TestPruneEntryComplete(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	ident := newAccountIdent(common.Hash{0xb})
	iw, _ := newIndexWriter(db, ident, 0, 0)

	for i := 1; i <= 10; i++ {
		if err := iw.append(uint64(i), nil); err != nil {
			t.Fatalf("Failed to append: %v", err)
		}
	}
	batch := db.NewBatch()
	iw.finish(batch)
	if err := batch.Write(); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	pruner := newIndexPruner(db, typeStateHistory)
	defer pruner.close()

	// Prune with tail above all elements
	if err := pruner.process(11); err != nil {
		t.Fatalf("Failed to process: %v", err)
	}

	// Metadata entry should be deleted
	blob := readStateIndex(ident, db)
	if len(blob) != 0 {
		t.Fatal("Index metadata should be empty after full prune")
	}
}

// TestPruneNoop verifies that pruning does nothing when the tail is below all
// block maximums.
func TestPruneNoop(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	ident := newAccountIdent(common.Hash{0xc})
	iw, _ := newIndexWriter(db, ident, 0, 0)

	for i := 100; i <= 200; i++ {
		if err := iw.append(uint64(i), nil); err != nil {
			t.Fatalf("Failed to append: %v", err)
		}
	}
	batch := db.NewBatch()
	iw.finish(batch)
	if err := batch.Write(); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	blob := readStateIndex(ident, db)
	origLen := len(blob)

	pruner := newIndexPruner(db, typeStateHistory)
	defer pruner.close()

	if err := pruner.process(50); err != nil {
		t.Fatalf("Failed to process: %v", err)
	}

	// Nothing should have changed
	blob = readStateIndex(ident, db)
	if len(blob) != origLen {
		t.Fatalf("Expected no change, original len %d, got %d", origLen, len(blob))
	}
}

// TestPrunePreservesReadability verifies that after pruning, the remaining
// index data is still readable and returns correct results.
func TestPrunePreservesReadability(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	ident := newAccountIdent(common.Hash{0xe})
	descList := writeMultiBlockIndex(t, db, ident, 0, 1)
	firstBlockMax := descList[0].max

	pruner := newIndexPruner(db, typeStateHistory)
	defer pruner.close()

	if err := pruner.process(firstBlockMax + 1); err != nil {
		t.Fatalf("Failed to process: %v", err)
	}

	// Read the remaining index and verify lookups still work
	ir, err := newIndexReader(db, ident, 0)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	// Looking for something greater than firstBlockMax should still work
	result, err := ir.readGreaterThan(firstBlockMax)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	if result != firstBlockMax+1 {
		t.Fatalf("Expected %d, got %d", firstBlockMax+1, result)
	}

	// Looking for the last element should return MaxUint64
	result, err = ir.readGreaterThan(20000)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	if result != math.MaxUint64 {
		t.Fatalf("Expected MaxUint64, got %d", result)
	}
}

// TestPrunePauseResume verifies the pause/resume mechanism:
//   - The pruner pauses mid-iteration and flushes its batch
//   - Data written while the pruner is paused (simulating indexSingle) is
//     visible after resume via a fresh iterator
//   - Pruning still completes correctly after resume
func TestPrunePauseResume(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	// Create many accounts with multi-block indexes so the pruner is still
	// iterating when the pause request arrives.
	var firstBlockMax uint64
	for i := 0; i < 200; i++ {
		hash := common.Hash{byte(i)}
		ident := newAccountIdent(hash)
		descList := writeMultiBlockIndex(t, db, ident, 0, 1)
		if i == 0 {
			firstBlockMax = descList[0].max
		}
	}
	// Target account at the end of the key space — the pruner should not
	// have visited it yet when the pause is acknowledged.
	targetIdent := newAccountIdent(common.Hash{0xff})
	targetDescList := writeMultiBlockIndex(t, db, targetIdent, 0, 1)

	tail := firstBlockMax + 1

	// Construct the pruner without starting run(). Calling process()
	// directly while run() is active would race: both run()'s main select
	// and prunePrefix's select listen on pauseReq. If run() receives it
	// (idle ack), process() runs unpaused and can overwrite data with a
	// stale iterator snapshot.
	pruner := &indexPruner{
		disk:     db,
		typ:      typeStateHistory,
		log:      log.New("type", "account"),
		closed:   make(chan struct{}),
		pauseReq: make(chan chan struct{}),
		resumeCh: make(chan struct{}),
	}

	// Run process() in the background.
	errCh := make(chan error, 1)
	go func() {
		errCh <- pruner.process(tail)
	}()

	// Pause — blocks until the pruner has flushed pending writes and
	// acknowledged. Because pauseReq is unbuffered, the send in pause()
	// blocks until prunePrefix's select receives it; the pruner checks
	// the channel on every iteration, so this always succeeds before
	// the iterator is exhausted.
	pruner.pause()

	// While paused, append a new element to the target account's index,
	// simulating what indexSingle would do during the pause window.
	lastMax := targetDescList[len(targetDescList)-1].max
	newID := lastMax + 10000
	iw, err := newIndexWriter(db, targetIdent, lastMax, 0)
	if err != nil {
		t.Fatalf("Failed to create index writer: %v", err)
	}
	if err := iw.append(newID, nil); err != nil {
		t.Fatalf("Failed to append: %v", err)
	}
	batch := db.NewBatch()
	iw.finish(batch)
	if err := batch.Write(); err != nil {
		t.Fatalf("Failed to write batch: %v", err)
	}

	// Resume the pruner.
	pruner.resume()

	// Wait for process() to complete.
	if err := <-errCh; err != nil {
		t.Fatalf("process() failed: %v", err)
	}

	// Verify: the entry written during the pause must still be accessible.
	// If the pruner used a stale iterator snapshot, it would overwrite the
	// target's metadata and lose the new entry.
	ir, err := newIndexReader(db, targetIdent, 0)
	if err != nil {
		t.Fatalf("Failed to create index reader: %v", err)
	}
	result, err := ir.readGreaterThan(newID - 1)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	if result != newID {
		t.Fatalf("Entry written during pause was lost: want %d, got %d", newID, result)
	}

	// Verify: pruning actually occurred on an early account.
	earlyIdent := newAccountIdent(common.Hash{0x00})
	earlyBlob := readStateIndex(earlyIdent, db)
	if len(earlyBlob) == 0 {
		t.Fatal("Early account index should not be completely empty")
	}
	earlyRemaining, err := parseIndex(earlyBlob, 0)
	if err != nil {
		t.Fatalf("Failed to parse early account index: %v", err)
	}
	// The first block (id=0) should have been pruned.
	if earlyRemaining[0].id == 0 {
		t.Fatal("First block of early account should have been pruned")
	}
}
