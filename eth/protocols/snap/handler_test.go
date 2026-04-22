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
	"encoding/binary"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

func makeTestBAL(minSize int) *bal.BlockAccessList {
	n := minSize/33 + 1 // 33 bytes per storage read slot in RLP
	access := bal.AccountAccess{
		Address:      common.HexToAddress("0x01"),
		StorageReads: make([]*bal.EncodedStorage, n),
	}
	for i := range access.StorageReads {
		read := access.StorageReads[i].ToHash()
		binary.BigEndian.PutUint64(read[24:], uint64(i))
	}
	return &bal.BlockAccessList{access}
}

// getChainWithBALs creates a minimal test chain with BALs stored for each block.
// It returns the chain, block hashes, and the stored BAL data.
func getChainWithBALs(nBlocks int, balSize int) (*core.BlockChain, []common.Hash, []rlp.RawValue) {
	gspec := &core.Genesis{
		Config: params.MergedTestChainConfig,
	}
	db := rawdb.NewMemoryDatabase()
	engine := beacon.New(ethash.NewFaker())
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, engine, nBlocks, func(i int, gen *core.BlockGen) {})
	options := &core.BlockChainConfig{
		StateScheme:   rawdb.PathScheme,
		TrieTimeLimit: 5 * time.Minute,
		NoPrefetch:    true,
	}
	bc, err := core.NewBlockChain(db, gspec, engine, options)
	if err != nil {
		panic(err)
	}
	if _, err := bc.InsertChain(blocks); err != nil {
		panic(err)
	}

	// Store BALs for each block
	var (
		hashes []common.Hash
		bals   []rlp.RawValue
	)
	for _, block := range blocks {
		hash := block.Hash()
		number := block.NumberU64()

		// Fill with data based on block number
		bytes, err := rlp.EncodeToBytes(makeTestBAL(balSize))
		if err != nil {
			panic(err)
		}
		rawdb.WriteAccessListRLP(db, hash, number, bytes)
		hashes = append(hashes, hash)
		bals = append(bals, bytes)
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
		Bytes:  softResponseLimit,
	}
	result := ServiceGetAccessListsQuery(bc, req)

	// Verify the results
	if result.Len() != len(hashes) {
		t.Fatalf("expected %d results, got %d", len(hashes), result.Len())
	}
	var (
		index int
		it    = result.ContentIterator()
	)
	for it.Next() {
		if !bytes.Equal(it.Value(), bals[index]) {
			t.Errorf("BAL %d mismatch: got %x, want %x", index, it.Value(), bals[index])
		}
		index++
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
		Bytes:  softResponseLimit,
	}
	result := ServiceGetAccessListsQuery(bc, req)

	// Verify length
	if result.Len() != len(mixed) {
		t.Fatalf("expected %d results, got %d", len(mixed), result.Len())
	}

	// Check positional correspondence
	var expectVal = []rlp.RawValue{
		bals[0], rlp.EmptyString, bals[1], rlp.EmptyString, bals[2],
	}
	var (
		index int
		it    = result.ContentIterator()
	)
	for it.Next() {
		if !bytes.Equal(it.Value(), expectVal[index]) {
			t.Errorf("BAL %d mismatch: got %x, want %x", index, it.Value(), expectVal[index])
		}
		index++
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
		Bytes:  softResponseLimit,
	}
	result := ServiceGetAccessListsQuery(bc, req)

	// Can't get more than maxAccessListLookups results
	if result.Len() > maxAccessListLookups {
		t.Fatalf("expected at most %d results, got %d", maxAccessListLookups, result.Len())
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
		Bytes:  softResponseLimit,
	}
	result := ServiceGetAccessListsQuery(bc, req)

	// Should have stopped before returning all blocks
	if result.Len() >= nBlocks {
		t.Fatalf("expected fewer than %d results due to byte limit, got %d", nBlocks, result.Len())
	}

	// Should have returned at least one
	if result.Len() == 0 {
		t.Fatal("expected at least one result")
	}

	// The total size should exceed the limit (the entry that crosses it is included)
	if result.Size() <= softResponseLimit {
		t.Errorf("total response size %d should exceed soft limit %d (includes one entry past limit)", result.Size(), softResponseLimit)
	}
}

// TestGetAccessListResponseDecoding verifies that an AccessListsPacket
// round-trips through RLP encode/decode, preserving positional
// correspondence and correctly representing absent BALs as empty strings.
func TestGetAccessListResponseDecoding(t *testing.T) {
	t.Parallel()

	// Build two real BALs of different sizes.
	bal1 := makeTestBAL(100)
	bal2 := makeTestBAL(200)
	bytes1, _ := rlp.EncodeToBytes(bal1)
	bytes2, _ := rlp.EncodeToBytes(bal2)

	tests := []struct {
		name   string
		items  []rlp.RawValue // nil entry = unavailable BAL
		counts int            // expected decoded length
	}{
		{
			name:   "all present",
			items:  []rlp.RawValue{bytes1, bytes2},
			counts: 2,
		},
		{
			name:   "all absent",
			items:  []rlp.RawValue{rlp.EmptyString, rlp.EmptyString, rlp.EmptyString},
			counts: 3,
		},
		{
			name:   "mixed present and absent",
			items:  []rlp.RawValue{bytes1, rlp.EmptyString, bytes2, rlp.EmptyString},
			counts: 4,
		},
		{
			name:   "empty response",
			items:  []rlp.RawValue{},
			counts: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the packet using Append.
			var orig AccessListsPacket
			orig.ID = 42
			for _, item := range tt.items {
				if err := orig.AccessLists.AppendRaw(item); err != nil {
					t.Fatalf("AppendRaw failed: %v", err)
				}
			}

			// Encode -> Decode round-trip.
			enc, err := rlp.EncodeToBytes(&orig)
			if err != nil {
				t.Fatalf("encode failed: %v", err)
			}
			var dec AccessListsPacket
			if err := rlp.DecodeBytes(enc, &dec); err != nil {
				t.Fatalf("decode failed: %v", err)
			}

			// Verify ID preserved.
			if dec.ID != orig.ID {
				t.Fatalf("ID mismatch: got %d, want %d", dec.ID, orig.ID)
			}

			// Verify element count.
			if dec.AccessLists.Len() != tt.counts {
				t.Fatalf("length mismatch: got %d, want %d", dec.AccessLists.Len(), tt.counts)
			}

			// Verify each element positionally.
			it := dec.AccessLists.ContentIterator()
			for i, want := range tt.items {
				if !it.Next() {
					t.Fatalf("iterator exhausted at index %d", i)
				}
				got := it.Value()
				if !bytes.Equal(got, want) {
					t.Errorf("element %d: got %x, want %x", i, got, want)
				}
				if !bytes.Equal(got, rlp.EmptyString) {
					obj := new(bal.BlockAccessList)
					if err := rlp.DecodeBytes(got, obj); err != nil {
						t.Fatalf("decode failed: %v", err)
					}
					if bytes.Equal(got, bytes1) && !reflect.DeepEqual(obj, bal1) {
						t.Fatalf("decode failed: got %x, want %x", obj, bal1)
					}
					if bytes.Equal(got, bytes2) && !reflect.DeepEqual(obj, bal2) {
						t.Fatalf("decode failed: got %x, want %x", obj, bal2)
					}
				}
			}
			if it.Next() {
				t.Error("iterator has extra elements after expected end")
			}
		})
	}
}
