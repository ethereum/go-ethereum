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
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/blake3"
)

// FuzzTrieOps decodes the fuzz input as an operation stream over a small
// key universe and checks the engine against the rebuild model after every
// operation.
func FuzzTrieOps(f *testing.F) {
	f.Add([]byte{0x01, 0x02, 0x83, 0x44, 0x05, 0xc6, 0x07, 0x88, 0x09, 0x0a})
	f.Add([]byte{0xff, 0x00, 0xff, 0x00, 0xff, 0x00})
	f.Fuzz(func(t *testing.T, data []byte) {
		tr := newTestTrie()
		model := make(map[string][]byte)
		for i := 0; i+2 < len(data); i += 3 {
			entity, sub, val := data[i]&0x07, data[i+1], data[i+2]
			var key []byte
			seed := [1]byte{entity}
			switch entity & 0x03 {
			case 0:
				h := blake3.Sum256(seed[:])
				key = append(append([]byte{AccountZone}, h[:]...), sub)
			case 1:
				h := blake3.Sum256(append(seed[:], 0xC0))
				key = append(append([]byte{CodeZone}, h[:]...), sub)
			default:
				p := blake3.Sum256(seed[:])
				q := blake3.Sum256(append(seed[:], entity>>2))
				key = append([]byte{StorageZone}, p[:]...)
				key = append(key, q[:]...)
				key = append(key, sub)
			}
			if val == 0 {
				delete(model, string(key))
				deleteKey(t, tr, key)
			} else {
				value := make([]byte, 32)
				value[31] = val
				model[string(key)] = value
				setKey(t, tr, key, value)
			}
			if got, want := tr.Hash(), modelRoot(model); got != want {
				t.Fatalf("op %d: engine %x model %x", i/3, got, want)
			}
		}
	})
}

// mutateSeeds are the records FuzzNodeMutate starts from. They must all decode:
// a record the decoder rejects makes the fuzz body return before it reaches a
// single walk, which is how this target was vacuous when first written.
func mutateSeeds() [][]byte {
	branch := func(nbits int) []byte {
		p := make([]byte, 2+(nbits+7)/8)
		binary.BigEndian.PutUint16(p, uint16(nbits))
		blob := append([]byte{tagBranch}, p...)
		return append(blob, bytes.Repeat([]byte{1}, 64)...)
	}
	return [][]byte{
		leafBlob(AccountZone, AccountKeyLength),
		leafBlob(StorageZone, StorageKeyLength),
		groupBlob(AccountZone, AccountKeyLength-1),
		groupBlob(StorageZone, StorageKeyLength-1),
		branch(0),
		branch(8),
	}
}

// TestFuzzNodeMutateSeedsAreLive guards the fuzz target below.
//
// Every seed has to survive decodeNode, or the fuzz body returns before
// touching a walk and the target proves nothing while still reporting success.
// That is not hypothetical: the first version of this target seeded only
// records the decoder rejects, so it explored nothing at all.
func TestFuzzNodeMutateSeedsAreLive(t *testing.T) {
	for i, seed := range mutateSeeds() {
		if _, err := decodeNode(seed); err != nil {
			t.Errorf("seed %d never reaches a walk, decodeNode rejects it: %v", i, err)
		}
	}
}

// FuzzNodeMutate plants a decoded record as a resident node and drives the
// walks across it. Every walk may return an error; none may panic.
//
// Surviving the decoder is not the same as surviving the engine: the walks do
// position arithmetic against a resident stem, and neither existing target
// reaches that. FuzzNodeDecode stops at decode and re-encode, and FuzzTrieOps
// only ever builds trees out of conformant keys, so the resident shapes it
// produces are the ones the engine itself would have made.
//
// What this explores is the cross product the others miss: an arbitrary record
// that is structurally valid, against an ordinary key. The keys are kept
// conformant deliberately, so a failure points at the record.
func FuzzNodeMutate(f *testing.F) {
	for _, seed := range mutateSeeds() {
		f.Add(seed, []byte{0x00, 0x01})
	}
	f.Fuzz(func(t *testing.T, record, seed []byte) {
		resident, err := decodeNode(record)
		if err != nil {
			return // the decoder is FuzzNodeDecode's job
		}
		// Build an ordinary, conformant stem from the seed, so what is being
		// fuzzed is the resident record rather than the key.
		var stem []byte
		if len(seed) > 0 && seed[0]&1 == 0 {
			stem = HeaderStem(common.BytesToAddress(seed))
		} else {
			stem = StorageBucketPrefix(common.BytesToAddress(seed))
			for len(stem) < StorageKeyLength-1 {
				stem = append(stem, 0)
			}
			stem = stem[:StorageKeyLength-1]
		}
		var sub byte
		if len(seed) > 1 {
			sub = seed[1]
		}
		value := make([]byte, 32)
		value[31] = sub | 1

		tr := newTestTrie()
		tr.root = resident

		// Each walk is driven directly rather than through the public methods,
		// which screen their arguments before the walk ever sees them.
		_, _, _ = tr.getStem(resident, stem, 0)
		_, _ = tr.insStem(resident, stem, []byte{sub}, [][]byte{value}, 0)
		_, _, _ = tr.delStem(resident, stem, 0)
		_, _, _ = tr.findPrefix(resident, stem, 0)
		_, _, _ = tr.delPrefix(resident, stem, bitstr{}, 0)

		// And once through the public surface, which is what a caller reaches.
		_ = tr.UpdateStem(stem, []byte{sub}, [][]byte{value})
		_, _ = tr.getStemGroup(stem)
		tr.Hash()
	})
}

// FuzzNodeDecode throws arbitrary bytes at the record decoder: it must never
// panic, and anything that decodes must re-encode to the same hash.
func FuzzNodeDecode(f *testing.F) {
	f.Add([]byte{0x00})
	f.Add([]byte{0x01, 0x00, 0x03, 0xa0})
	f.Add([]byte{0x02, 0x00, 0x10, 0x21})
	f.Fuzz(func(t *testing.T, data []byte) {
		n, err := decodeNode(data)
		if err != nil {
			return
		}
		switch nn := n.(type) {
		case *groupNode:
			pos := nn.cachedAt
			blob := serializeNode(nn, pos)
			redec, err := decodeNode(blob)
			if err != nil {
				t.Fatalf("re-decode failed: %v", err)
			}
			if redec.hashAt(pos) != n.hashAt(pos) {
				t.Fatal("re-encoded group hash mismatch")
			}
		case *branchNode:
			blob := serializeNode(nn, 0)
			if _, err := decodeNode(blob); err != nil {
				t.Fatalf("re-decode failed: %v", err)
			}
		}
	})
}
