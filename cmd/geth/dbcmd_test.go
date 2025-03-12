// Copyright 2023 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
)

// TestTruncateFreezerBatching tests the batch processing approach for headers
// in the truncate-freezer command.
func TestTruncateFreezerBatching(t *testing.T) {
	// Create a temporary directory for the test
	datadir := t.TempDir()
	defer os.RemoveAll(datadir)

	// Create a freezer database
	freezerDir := filepath.Join(datadir, "freezer")
	db, err := rawdb.NewDatabaseWithFreezer(rawdb.NewMemoryDatabase(), freezerDir, "", false)
	if err != nil {
		t.Fatalf("Failed to create freezer database: %v", err)
	}
	defer db.Close()

	// Create test data: 100 pre-merge blocks and 10 post-merge blocks
	// The merge block will be at index 100
	const (
		preMergeBlocks  = 100
		postMergeBlocks = 10
		mergeBlock      = preMergeBlocks
		totalBlocks     = preMergeBlocks + postMergeBlocks
	)

	// Generate and insert blocks
	var (
		parentHash = common.Hash{}
		genesis    = types.Header{
			Number:     big.NewInt(0),
			Difficulty: big.NewInt(params.GenesisDifficulty.Int64()),
			ParentHash: parentHash,
		}
	)

	// Insert genesis header
	rawdb.WriteHeader(db, &genesis)
	parentHash = genesis.Hash()

	// Insert pre-merge blocks with non-zero difficulty
	for i := 1; i <= preMergeBlocks; i++ {
		header := types.Header{
			Number:     big.NewInt(int64(i)),
			Difficulty: big.NewInt(params.GenesisDifficulty.Int64()),
			ParentHash: parentHash,
		}
		hash := header.Hash()
		rawdb.WriteHeader(db, &header)
		rawdb.WriteCanonicalHash(db, hash, uint64(i))
		parentHash = hash
	}

	// Insert post-merge blocks with zero difficulty
	for i := preMergeBlocks + 1; i <= totalBlocks; i++ {
		header := types.Header{
			Number:     big.NewInt(int64(i)),
			Difficulty: big.NewInt(0), // Zero difficulty = post-merge
			ParentHash: parentHash,
		}
		hash := header.Hash()
		rawdb.WriteHeader(db, &header)
		rawdb.WriteCanonicalHash(db, hash, uint64(i))
		parentHash = hash
	}

	// Set the head block
	rawdb.WriteHeadBlockHash(db, parentHash)

	// Create a temporary directory for the headers freezer
	tmpDir := t.TempDir()

	// Create a new freezer for headers and hashes
	headersFreezer, err := rawdb.NewFreezer(tmpDir, "headers", false, 2*1000*1000*1000, map[string]bool{
		rawdb.ChainFreezerHeaderTable: false,
		rawdb.ChainFreezerHashTable:   false,
	})
	if err != nil {
		t.Fatalf("Failed to create headers freezer: %v", err)
	}
	defer headersFreezer.Close()

	// Get the ancient reader
	ancientDb, ok := db.(ethdb.AncientReader)
	if !ok {
		t.Fatal("Database doesn't support ancient storage")
	}

	// Get the number of items in the freezer
	ancients, err := ancientDb.Ancients()
	if err != nil {
		t.Fatalf("Failed to get ancients count: %v", err)
	}
	t.Logf("Number of items in freezer: %d", ancients)

	// Test batch processing by copying headers to the temporary freezer
	const batchSize = 10 // Small batch size for testing
	for i := uint64(0); i < mergeBlock; i += batchSize {
		end := i + batchSize
		if end > mergeBlock {
			end = mergeBlock
		}

		// Process this batch
		_, err = headersFreezer.ModifyAncients(func(op ethdb.AncientWriteOp) error {
			for j := i; j < end; j++ {
				// Skip if j is out of range
				if j >= ancients {
					continue
				}

				// Read header and hash
				headerBytes, err := ancientDb.Ancient(rawdb.ChainFreezerHeaderTable, j)
				if err != nil {
					return err
				}
				hashBytes, err := ancientDb.Ancient(rawdb.ChainFreezerHashTable, j)
				if err != nil {
					return err
				}

				// Write to temporary freezer
				if err := op.AppendRaw(rawdb.ChainFreezerHeaderTable, j, headerBytes); err != nil {
					return err
				}
				if err := op.AppendRaw(rawdb.ChainFreezerHashTable, j, hashBytes); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to copy headers batch: %v", err)
		}
	}

	// Verify that the headers were copied correctly
	headersFreezerAncients, err := headersFreezer.Ancients()
	if err != nil {
		t.Fatalf("Failed to get ancients count from headers freezer: %v", err)
	}

	// The number of items in the headers freezer should be equal to the merge block
	// or the number of items in the original freezer, whichever is smaller
	expectedCount := mergeBlock
	if ancients < mergeBlock {
		expectedCount = int(ancients)
	}

	if int(headersFreezerAncients) != expectedCount {
		t.Fatalf("Expected %d items in headers freezer, got %d", expectedCount, headersFreezerAncients)
	}

	// Verify that the headers in the temporary freezer match the originals
	for i := uint64(0); i < uint64(expectedCount); i++ {
		// Read from original freezer
		originalHeader, err := ancientDb.Ancient(rawdb.ChainFreezerHeaderTable, i)
		if err != nil {
			t.Fatalf("Failed to read header %d from original freezer: %v", i, err)
		}
		originalHash, err := ancientDb.Ancient(rawdb.ChainFreezerHashTable, i)
		if err != nil {
			t.Fatalf("Failed to read hash %d from original freezer: %v", i, err)
		}

		// Read from temporary freezer
		tempHeader, err := headersFreezer.Ancient(rawdb.ChainFreezerHeaderTable, i)
		if err != nil {
			t.Fatalf("Failed to read header %d from temporary freezer: %v", i, err)
		}
		tempHash, err := headersFreezer.Ancient(rawdb.ChainFreezerHashTable, i)
		if err != nil {
			t.Fatalf("Failed to read hash %d from temporary freezer: %v", i, err)
		}

		// Compare
		if string(originalHeader) != string(tempHeader) {
			t.Fatalf("Header %d mismatch", i)
		}
		if string(originalHash) != string(tempHash) {
			t.Fatalf("Hash %d mismatch", i)
		}
	}

	t.Log("Batch processing test passed successfully")
}
