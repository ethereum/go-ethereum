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

package snap

import (
	"bytes"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// getChainWithBALs creates a minimal test chain with BALs stored for each block.
// It returns the chain, block hashes, and the stored BAL data.
func getChainWithBALs(nBlocks int, balSize int) (*core.BlockChain, []common.Hash, []rlp.RawValue) {
	gspec := &core.Genesis{
		Config: params.TestChainConfig,
	}
	db := rawdb.NewMemoryDatabase()
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, ethash.NewFaker(), nBlocks, func(i int, gen *core.BlockGen) {})
	options := &core.BlockChainConfig{
		TrieCleanLimit: 0,
		TrieDirtyLimit: 0,
		TrieTimeLimit:  5 * time.Minute,
		NoPrefetch:     true,
		SnapshotLimit:  0,
	}
	bc, err := core.NewBlockChain(db, gspec, ethash.NewFaker(), options)
	if err != nil {
		panic(err)
	}
	if _, err := bc.InsertChain(blocks); err != nil {
		panic(err)
	}

	// Store BALs for each block
	var hashes []common.Hash
	var bals []rlp.RawValue
	for _, block := range blocks {
		hash := block.Hash()
		number := block.NumberU64()
		bal := make(rlp.RawValue, balSize)

		// Fill with data based on block number
		for j := range bal {
			bal[j] = byte(number + uint64(j))
		}
		rawdb.WriteAccessListRLP(db, hash, number, bal)
		hashes = append(hashes, hash)
		bals = append(bals, bal)
	}
	return bc, hashes, bals
}

// TestServiceGetAccessListsQuery verifies that known block hashes return the
// correct BALs with positional correspondence.
func TestServiceGetAccessListsQuery(t *testing.T) {
	t.Parallel()
	bc, hashes, bals := getChainWithBALs(5, 100)
	defer bc.Stop()
	req := &GetAccessListsPacket{
		ID:     1,
		Hashes: hashes,
	}
	result := ServiceGetAccessListsQuery(bc, req)

	// Verify the results
	if len(result) != len(hashes) {
		t.Fatalf("expected %d results, got %d", len(hashes), len(result))
	}
	for i, bal := range result {
		if !bytes.Equal(bal, bals[i]) {
			t.Errorf("BAL %d mismatch: got %x, want %x", i, bal, bals[i])
		}
	}
}

// TestServiceGetAccessListsQueryEmpty verifies that unknown block hashes return
// nil placeholders and that mixed known/unknown hashes preserve alignment.
func TestServiceGetAccessListsQueryEmpty(t *testing.T) {
	t.Parallel()
	bc, hashes, bals := getChainWithBALs(3, 100)
	defer bc.Stop()
	unknown := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	mixed := []common.Hash{hashes[0], unknown, hashes[1], unknown, hashes[2]}
	req := &GetAccessListsPacket{
		ID:     2,
		Hashes: mixed,
	}
	result := ServiceGetAccessListsQuery(bc, req)

	// Verify length
	if len(result) != len(mixed) {
		t.Fatalf("expected %d results, got %d", len(mixed), len(result))
	}

	// Check positional correspondence
	if !bytes.Equal(result[0], bals[0]) {
		t.Errorf("index 0: expected known BAL, got %x", result[0])
	}
	if result[1] != nil {
		t.Errorf("index 1: expected nil for unknown hash, got %x", result[1])
	}
	if !bytes.Equal(result[2], bals[1]) {
		t.Errorf("index 2: expected known BAL, got %x", result[2])
	}
	if result[3] != nil {
		t.Errorf("index 3: expected nil for unknown hash, got %x", result[3])
	}
	if !bytes.Equal(result[4], bals[2]) {
		t.Errorf("index 4: expected known BAL, got %x", result[4])
	}
}

// TestServiceGetAccessListsQueryCap verifies that requests exceeding
// maxAccessListLookups are capped.
func TestServiceGetAccessListsQueryCap(t *testing.T) {
	t.Parallel()

	bc, _, _ := getChainWithBALs(2, 100)
	defer bc.Stop()

	// Create a request with more hashes than the cap
	hashes := make([]common.Hash, maxAccessListLookups+100)
	for i := range hashes {
		hashes[i] = common.BytesToHash([]byte{byte(i), byte(i >> 8)})
	}
	req := &GetAccessListsPacket{
		ID:     3,
		Hashes: hashes,
	}
	result := ServiceGetAccessListsQuery(bc, req)

	// Can't get more than maxAccessListLookups results
	if len(result) > maxAccessListLookups {
		t.Fatalf("expected at most %d results, got %d", maxAccessListLookups, len(result))
	}
}

// TestServiceGetAccessListsQueryByteLimit verifies that the response stops
// once the byte limit is exceeded. The handler appends the entry that crosses
// the limit before breaking, so the total size will exceed the limit by at
// most one BAL.
func TestServiceGetAccessListsQueryByteLimit(t *testing.T) {
	t.Parallel()

	// The handler will return 3/5 entries (3MB total) then break.
	balSize := 1024 * 1024
	nBlocks := 5
	bc, hashes, _ := getChainWithBALs(nBlocks, balSize)
	defer bc.Stop()
	req := &GetAccessListsPacket{
		ID:     0,
		Hashes: hashes,
	}
	result := ServiceGetAccessListsQuery(bc, req)

	// Should have stopped before returning all blocks
	if len(result) >= nBlocks {
		t.Fatalf("expected fewer than %d results due to byte limit, got %d", nBlocks, len(result))
	}

	// Should have returned at least one
	if len(result) == 0 {
		t.Fatal("expected at least one result")
	}

	// The total size should exceed the limit (the entry that crosses it is included)
	var total uint64
	for _, bal := range result {
		total += uint64(len(bal))
	}
	if total <= softResponseLimit {
		t.Errorf("total response size %d should exceed soft limit %d (includes one entry past limit)", total, softResponseLimit)
	}
}
