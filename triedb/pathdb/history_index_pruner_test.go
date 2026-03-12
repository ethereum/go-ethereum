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
