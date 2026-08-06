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
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

// vectorsFile mirrors testdata/eip8297_vectors.json, exported from the EELS
// reference implementation by testdata/export_vectors.py.
type vectorsFile struct {
	Meta         map[string]string `json:"meta"`
	EmptyRoot    string            `json:"empty_root"`
	StateVectors []struct {
		Name     string `json:"name"`
		Accounts []struct {
			Address string            `json:"address"`
			Nonce   uint64            `json:"nonce"`
			Balance *big.Int          `json:"balance"`
			Code    string            `json:"code"`
			Storage map[string]uint64 `json:"storage"`
		} `json:"accounts"`
		Root string `json:"root"`
	} `json:"state_vectors"`
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
		Address              string `json:"address"`
		BasicDataKey         string `json:"basic_data_key"`
		CodeHashKey          string `json:"code_hash_key"`
		DelegationKey        string `json:"delegation_key"`
		DelegationDesignator string `json:"delegation_designator"`
		DelegationValue      string `json:"delegation_value"`
		Slots                []struct {
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
	// A reader over no database, which reports every node as missing. Tests
	// that only insert never resolve anything, but one holding a hashed child
	// does - and with a nil reader that dereferences instead of reporting a
	// missing node, which is a property of this helper rather than the engine.
	reader, err := trie.NewReader(common.Hash{}, common.Hash{}, nil)
	if err != nil {
		panic(err)
	}
	return &BinaryTrie{
		root:   empty{},
		reader: reader,
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
// TestStateVectors drives whole allocations through the typed write path and
// compares the resulting root against the reference.
//
// This is the only test that pins what an embedding *writes*, rather than what
// it derives or how it hashes. The other vector tests are handed reference
// keys and values and insert them raw, so they cannot see a disagreement about
// which leaves exist at all - and that is exactly where the state layer makes
// decisions: EIP-8297 has it resolve a write of 32 zero bytes to a deletion,
// so an all-zero code chunk or an all-zero basic data leaf is absent, and an
// implementation that stores either commits to a different root while every
// read still returns the right answer.
func TestStateVectors(t *testing.T) {
	vf := loadVectors(t)
	if len(vf.StateVectors) == 0 {
		t.Fatal("no state vectors in the file; the exporter did not write them")
	}
	for _, sv := range vf.StateVectors {
		t.Run(sv.Name, func(t *testing.T) {
			tr := newTestTrie()
			for _, a := range sv.Accounts {
				addr := common.BytesToAddress(unhex(t, a.Address))
				var code []byte
				if a.Code != "" { // absent in the file for a codeless account
					code = unhex(t, a.Code)
				}
				balance := new(uint256.Int)
				if a.Balance != nil {
					balance = uint256.MustFromBig(a.Balance)
				}
				acc := &types.StateAccount{
					Nonce:    a.Nonce,
					Balance:  balance,
					Root:     types.EmptyRootHash,
					CodeHash: crypto.Keccak256(code),
				}
				// The classification the write path makes: a designator goes
				// in the header, everything else is chunked.
				var delegation []byte
				if _, ok := types.ParseDelegation(code); ok {
					delegation = code
				}
				if err := tr.UpdateAccount(addr, acc, len(code), delegation); err != nil {
					t.Fatalf("account %x: %v", addr, err)
				}
				if err := tr.UpdateContractCode(addr, common.BytesToHash(acc.CodeHash), code); err != nil {
					t.Fatalf("code for %x: %v", addr, err)
				}
				for slot, val := range a.Storage {
					key, ok := new(big.Int).SetString(slot, 10)
					if !ok {
						t.Fatalf("bad storage key %q", slot)
					}
					var k, v common.Hash
					key.FillBytes(k[:])
					new(big.Int).SetUint64(val).FillBytes(v[:])
					if err := tr.UpdateStorage(addr, k[:], common.TrimLeftZeroes(v[:])); err != nil {
						t.Fatalf("storage %s of %x: %v", slot, addr, err)
					}
				}
			}
			if got := tr.Hash(); got != common.HexToHash(sv.Root) {
				t.Fatalf("root %x, want %s", got, sv.Root)
			}
			assertNoZeroLeaf(t, tr)
		})
	}
}

// assertNoZeroLeaf walks every leaf and checks none holds 32 zero bytes.
//
// EIP-8297 states this as a property of the whole tree, not as a rule for one
// writer: "no key in the state's tree holds 32 zero bytes". Asserting it here
// catches a write path that bypasses stateWrite, which is the shape of the
// mistake rather than any particular instance of it.
func assertNoZeroLeaf(t *testing.T, tr *BinaryTrie) {
	t.Helper()
	it, err := tr.NodeIterator(nil)
	if err != nil {
		t.Fatal(err)
	}
	for it.Next(true) {
		if !it.Leaf() {
			continue
		}
		if isZeroValue(it.LeafBlob()) {
			t.Fatalf("leaf %x holds 32 zero bytes, which no key in the state's tree may", it.LeafKey())
		}
	}
	if err := it.Error(); err != nil {
		t.Fatal(err)
	}
}

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
	if got := DelegationKey(addr); !equalBytes(got, unhex(t, ev.DelegationKey)) {
		t.Fatalf("delegation key mismatch: %x", got)
	}
	// The padding is the part worth pinning: hashing the padded value rather
	// than the leading code_size bytes would disagree with EXTCODEHASH, and
	// both encodings look equally plausible from the Go side alone.
	if got := EncodeDelegation(unhex(t, ev.DelegationDesignator)); !equalBytes(got, unhex(t, ev.DelegationValue)) {
		t.Fatalf("delegation value mismatch: got %x want %s", got, ev.DelegationValue)
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
	// The code hash the chunk keys were derived from. It is not carried in the
	// vector file, so this literal has to match testdata/export_vectors.py,
	// which uses the same all-0xbb hash. Nothing checks that for us: if the
	// exporter's hash ever changes, every chunk key below fails and the reason
	// will not be obvious from the failure.
	ch := common.BytesToHash(unhexConst("bb", 32))
	for _, cv := range ev.Chunks {
		if got := CodeChunkKey(ch, cv.Chunk); !equalBytes(got, unhex(t, cv.Key)) {
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
