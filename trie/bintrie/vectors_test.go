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

package bintrie

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

// vectorsFile mirrors testdata/eip8297_vectors.json, exported from the EELS
// reference implementation by testdata/export_vectors.py.
type vectorsFile struct {
	Meta        map[string]string `json:"meta"`
	EmptyRoot   string            `json:"empty_root"`
	TrieVectors []struct {
		Name    string `json:"name"`
		Entries []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"entries"`
		Root string `json:"root"`
	} `json:"trie_vectors"`
	SequenceVectors []struct {
		Seed int `json:"seed"`
		Ops  []struct {
			Op    string `json:"op"`
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"ops"`
		RootsAfter []string `json:"roots_after"`
	} `json:"sequence_vectors"`
	EmbeddingVectors struct {
		Address      string `json:"address"`
		BasicDataKey string `json:"basic_data_key"`
		CodeHashKey  string `json:"code_hash_key"`
		Slots        []struct {
			Slot json.Number `json:"slot"`
			Key  string      `json:"key"`
		} `json:"slots"`
		Chunks []struct {
			Chunk uint64 `json:"chunk"`
			Key   string `json:"key"`
		} `json:"chunks"`
	} `json:"embedding_vectors"`
	BasicDataVectors []struct {
		CodeSize uint32 `json:"code_size"`
		Nonce    uint64 `json:"nonce"`
		Balance  string `json:"balance"`
		Value    string `json:"value"`
	} `json:"basic_data_vectors"`
	ChunkifyVectors []struct {
		Name   string   `json:"name"`
		Code   string   `json:"code"`
		Chunks []string `json:"chunks"`
	} `json:"chunkify_vectors"`
}

func loadVectors(t testing.TB) *vectorsFile {
	t.Helper()
	blob, err := os.ReadFile("testdata/eip8297_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vf vectorsFile
	if err := json.Unmarshal(blob, &vf); err != nil {
		t.Fatal(err)
	}
	return &vf
}

func unhex(t testing.TB, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s[2:])
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// newTestTrie builds a database-less in-memory trie; resolution never
// triggers for tries built purely in memory.
func newTestTrie() *BinaryTrie {
	return &BinaryTrie{
		root:   empty{},
		tracer: trie.NewPrevalueTracer(),
		ops:    newOpTracer(),
	}
}

func setKey(t testing.TB, tr *BinaryTrie, key, value []byte) {
	t.Helper()
	if err := tr.UpdateStem(key[:len(key)-1], []byte{key[len(key)-1]}, [][]byte{value}); err != nil {
		t.Fatal(err)
	}
}

func deleteKey(t testing.TB, tr *BinaryTrie, key []byte) {
	t.Helper()
	if err := tr.UpdateStem(key[:len(key)-1], []byte{key[len(key)-1]}, [][]byte{nil}); err != nil {
		t.Fatal(err)
	}
}

// TestTrieVectors replays the EELS fixed-population vectors: build each
// entry set, compare the root byte for byte.
func TestTrieVectors(t *testing.T) {
	vf := loadVectors(t)
	if got := (empty{}).hashAt(0); got != common.HexToHash(vf.EmptyRoot) {
		t.Fatalf("empty root mismatch: %x", got)
	}
	for _, tv := range vf.TrieVectors {
		t.Run(tv.Name, func(t *testing.T) {
			tr := newTestTrie()
			for _, e := range tv.Entries {
				setKey(t, tr, unhex(t, e.Key), unhex(t, e.Value))
			}
			if got, want := tr.Hash(), common.HexToHash(tv.Root); got != want {
				t.Fatalf("root mismatch: got %x want %x", got, want)
			}
			// Round-trip: every inserted entry reads back.
			for _, e := range tv.Entries {
				got, err := tr.getValue(unhex(t, e.Key))
				if err != nil {
					t.Fatal(err)
				}
				if common.BytesToHash(got) != common.BytesToHash(unhex(t, e.Value)) {
					t.Fatalf("read-back mismatch for %s", e.Key)
				}
			}
		})
	}
}

// TestSequenceVectors replays the EELS randomized op sequences (inserts,
// overwrites and deletes), comparing the root after every operation.
func TestSequenceVectors(t *testing.T) {
	vf := loadVectors(t)
	for _, sv := range vf.SequenceVectors {
		tr := newTestTrie()
		for i, op := range sv.Ops {
			key := unhex(t, op.Key)
			switch op.Op {
			case "set":
				setKey(t, tr, key, unhex(t, op.Value))
			case "delete":
				deleteKey(t, tr, key)
			default:
				t.Fatalf("unknown op %q", op.Op)
			}
			if got, want := tr.Hash(), common.HexToHash(sv.RootsAfter[i]); got != want {
				t.Fatalf("seed %d: root mismatch after op %d (%s %s): got %x want %x",
					sv.Seed, i, op.Op, op.Key, got, want)
			}
		}
	}
}

// TestEmbeddingVectors pins the key-derivation helpers against the EELS
// embedding for a fixed address and code hash.
func TestEmbeddingVectors(t *testing.T) {
	vf := loadVectors(t)
	ev := vf.EmbeddingVectors
	addr := common.BytesToAddress(unhex(t, ev.Address))
	if got := BasicDataKey(addr); !equalBytes(got, unhex(t, ev.BasicDataKey)) {
		t.Fatalf("basic data key mismatch: %x", got)
	}
	if got := CodeHashKey(addr); !equalBytes(got, unhex(t, ev.CodeHashKey)) {
		t.Fatalf("code hash key mismatch: %x", got)
	}
	for _, sv := range ev.Slots {
		slotInt, err := uint256.FromDecimal(sv.Slot.String())
		if err != nil {
			t.Fatalf("bad slot %s", sv.Slot)
		}
		slot := slotInt.Bytes32()
		if got := StorageSlotKey(addr, slot[:]); !equalBytes(got, unhex(t, sv.Key)) {
			t.Fatalf("slot %s key mismatch: got %x want %s", sv.Slot, got, sv.Key)
		}
	}
	ch := common.BytesToHash(unhexConst("bb", 32))
	for _, cv := range ev.Chunks {
		if got := CodeChunkKey(addr, ch, cv.Chunk); !equalBytes(got, unhex(t, cv.Key)) {
			t.Fatalf("chunk %d key mismatch: got %x want %s", cv.Chunk, got, cv.Key)
		}
	}
}

// TestBasicDataVectors pins the BASIC_DATA packing.
func TestBasicDataVectors(t *testing.T) {
	vf := loadVectors(t)
	for _, bv := range vf.BasicDataVectors {
		bal, err := uint256.FromDecimal(bv.Balance)
		if err != nil {
			t.Fatalf("bad balance %s", bv.Balance)
		}
		got, err := EncodeBasicData(bv.CodeSize, bv.Nonce, bal)
		if err != nil {
			t.Fatal(err)
		}
		if !equalBytes(got[:], unhex(t, bv.Value)) {
			t.Fatalf("basic data mismatch: got %x want %s", got, bv.Value)
		}
		version, codeSize, nonce, balance := DecodeBasicData(got[:])
		if version != 0 || codeSize != bv.CodeSize || nonce != bv.Nonce || balance.Cmp(bal) != 0 {
			t.Fatalf("basic data decode mismatch for %s", bv.Value)
		}
	}
}

// TestChunkifyVectors pins code chunking against the EELS reference.
func TestChunkifyVectors(t *testing.T) {
	vf := loadVectors(t)
	for _, cv := range vf.ChunkifyVectors {
		code := unhex(t, cv.Code)
		chunks := ChunkifyCode(code)
		if len(chunks) != 32*len(cv.Chunks) {
			t.Fatalf("%s: chunk count mismatch: got %d want %d", cv.Name, len(chunks)/32, len(cv.Chunks))
		}
		for i, want := range cv.Chunks {
			if !equalBytes(chunks[32*i:32*(i+1)], unhex(t, want)) {
				t.Fatalf("%s: chunk %d mismatch", cv.Name, i)
			}
		}
	}
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func unhexConst(pattern string, n int) []byte {
	out := make([]byte, n)
	b, _ := hex.DecodeString(pattern)
	for i := range out {
		out[i] = b[i%len(b)]
	}
	return out
}
