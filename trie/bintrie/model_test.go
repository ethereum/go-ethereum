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
	"math/rand"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/blake3"
	"github.com/holiman/uint256"
)

// modelRoot is an independent, from-scratch port of the EELS reference
// (binarize + merkleize): rebuild the canonical structure of the whole key
// set recursively and hash it, sharing no code with the engine beyond the
// BLAKE3 dependency. It is the in-repo oracle for differential testing.
func modelRoot(entries map[string][]byte) common.Hash {
	if len(entries) == 0 {
		return common.Hash{}
	}
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return modelMerkleize(keys, entries, 0)
}

func modelBit(key string, i int) byte {
	return key[i>>3] >> (7 - i&7) & 1
}

func modelMerkleize(keys []string, entries map[string][]byte, depth int) common.Hash {
	if len(keys) == 1 {
		preimage := append([]byte{0x00}, keys[0]...)
		preimage = append(preimage, entries[keys[0]]...)
		return common.Hash(blake3.Sum256(preimage))
	}
	// Advance to the first bit where the keys diverge.
	split := depth
	for {
		bit := modelBit(keys[0], split)
		diverged := false
		for _, k := range keys[1:] {
			if modelBit(k, split) != bit {
				diverged = true
				break
			}
		}
		if diverged {
			break
		}
		split++
	}
	// Partition by the diverging bit (keys are sorted, zeros first).
	cut := len(keys)
	for i, k := range keys {
		if modelBit(k, split) == 1 {
			cut = i
			break
		}
	}
	left := modelMerkleize(keys[:cut], entries, split+1)
	right := modelMerkleize(keys[cut:], entries, split+1)

	// encode_bit_prefix of the shared run keys[0][depth:split].
	n := split - depth
	packed := make([]byte, (n+7)/8)
	for i := 0; i < n; i++ {
		if modelBit(keys[0], depth+i) == 1 {
			packed[i>>3] |= 1 << (7 - i&7)
		}
	}
	preimage := []byte{0x01, byte(n >> 8), byte(n)}
	preimage = append(preimage, packed...)
	preimage = append(preimage, left[:]...)
	preimage = append(preimage, right[:]...)
	return common.Hash(blake3.Sum256(preimage))
}

// randomConformantKey generates keys over a small universe so collisions,
// same-stem groupings and cross-zone mixes all occur: a few account stems,
// storage stems for the same "addresses", and code stems.
func randomConformantKey(rng *rand.Rand) []byte {
	entity := uint64(rng.Intn(12)) // small universe: stems recur
	sub := byte(rng.Intn(256))
	var seed [8]byte
	binary.BigEndian.PutUint64(seed[:], entity)
	switch rng.Intn(4) {
	case 0: // account header
		h := blake3.Sum256(seed[:])
		return append(append([]byte{AccountZone}, h[:]...), sub)
	case 1: // code zone
		h := blake3.Sum256(append(seed[:], 0xC0))
		return append(append([]byte{CodeZone}, h[:]...), sub)
	default: // storage: two-digest stem, few groups per entity
		group := uint64(rng.Intn(3))
		p := blake3.Sum256(seed[:])
		var gseed [16]byte
		copy(gseed[:8], seed[:])
		binary.BigEndian.PutUint64(gseed[8:], group)
		q := blake3.Sum256(gseed[:])
		key := append([]byte{StorageZone}, p[:]...)
		key = append(key, q[:]...)
		return append(key, sub)
	}
}

// TestRandomOpsVsModel runs randomized insert/overwrite/delete sequences,
// comparing the engine root against the independent rebuild model after
// every operation.
func TestRandomOpsVsModel(t *testing.T) {
	seeds := []int64{1, 7, 42, 8297, 11832, 90210, 20260728}
	if testing.Short() {
		seeds = seeds[:3]
	}
	for _, seed := range seeds {
		rng := rand.New(rand.NewSource(seed))
		tr := newTestTrie()
		model := make(map[string][]byte)
		live := make([]string, 0, 256)

		for op := 0; op < 300; op++ {
			if len(live) > 0 && rng.Intn(4) == 0 {
				// Delete a random live key.
				idx := rng.Intn(len(live))
				key := live[idx]
				live = append(live[:idx], live[idx+1:]...)
				delete(model, key)
				deleteKey(t, tr, []byte(key))
			} else {
				key := randomConformantKey(rng)
				var value [32]byte
				rng.Read(value[:])
				if _, ok := model[string(key)]; !ok {
					live = append(live, string(key))
				}
				model[string(key)] = append([]byte{}, value[:]...)
				setKey(t, tr, key, value[:])
			}
			if got, want := tr.Hash(), modelRoot(model); got != want {
				t.Fatalf("seed %d op %d: engine %x model %x", seed, op, got, want)
			}
		}
		// Drain: delete everything, checking canonical collapse throughout.
		for len(live) > 0 {
			idx := rng.Intn(len(live))
			key := live[idx]
			live = append(live[:idx], live[idx+1:]...)
			delete(model, key)
			deleteKey(t, tr, []byte(key))
			if got, want := tr.Hash(), modelRoot(model); got != want {
				t.Fatalf("seed %d drain: engine %x model %x", seed, got, want)
			}
		}
		if tr.Hash() != (common.Hash{}) {
			t.Fatalf("seed %d: non-empty root after drain", seed)
		}
	}
}

// TestDeletePrefixVsModel checks bucket drops against the model across the
// locator's shapes: bucket as group record, as prefix-straddling branch,
// multi-group buckets, absent buckets, and bucket == whole tree.
func TestDeletePrefixVsModel(t *testing.T) {
	for _, seed := range []int64{3, 9, 8297} {
		rng := rand.New(rand.NewSource(seed))
		tr := newTestTrie()
		model := make(map[string][]byte)
		for op := 0; op < 200; op++ {
			key := randomConformantKey(rng)
			var value [32]byte
			rng.Read(value[:])
			model[string(key)] = append([]byte{}, value[:]...)
			setKey(t, tr, key, value[:])
		}
		// Drop every distinct storage bucket present, one at a time.
		prefixes := make(map[string]struct{})
		for k := range model {
			if k[0] == byte(StorageZone) {
				prefixes[k[:33]] = struct{}{}
			}
		}
		for p := range prefixes {
			if err := tr.DeletePrefix([]byte(p)); err != nil {
				t.Fatal(err)
			}
			for k := range model {
				if len(k) >= 33 && k[:33] == p {
					delete(model, k)
				}
			}
			if got, want := tr.Hash(), modelRoot(model); got != want {
				t.Fatalf("seed %d bucket %x: engine %x model %x", seed, p, got, want)
			}
			// Absent-bucket drop is a no-op.
			if err := tr.DeletePrefix([]byte(p)); err != nil {
				t.Fatal(err)
			}
			if got := tr.Hash(); got != modelRoot(model) {
				t.Fatalf("seed %d: absent bucket drop changed root", seed)
			}
		}
	}
	// Bucket == whole tree.
	tr := newTestTrie()
	stem := StorageStem(common.Address{9}, uint256Zero())
	if err := tr.UpdateStem(stem, []byte{70, 80}, [][]byte{bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{2}, 32)}); err != nil {
		t.Fatal(err)
	}
	if err := tr.DeletePrefix(StorageBucketPrefix(common.Address{9})); err != nil {
		t.Fatal(err)
	}
	if tr.Hash() != (common.Hash{}) {
		t.Fatal("whole-tree bucket drop did not empty the trie")
	}
}

// TestSerializationRoundTrip decodes what it encodes across node shapes.
func TestSerializationRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	tr := newTestTrie()
	model := make(map[string][]byte)
	for op := 0; op < 150; op++ {
		key := randomConformantKey(rng)
		var value [32]byte
		rng.Read(value[:])
		model[string(key)] = append([]byte{}, value[:]...)
		setKey(t, tr, key, value[:])
	}
	tr.Hash()
	// Walk every node, encode, decode, re-hash: must match.
	var walkCheck func(n binaryNode, pos int)
	walkCheck = func(n binaryNode, pos int) {
		switch nn := n.(type) {
		case *groupNode:
			blob := serializeNode(nn, pos)
			dec, err := decodeNode(blob)
			if err != nil {
				t.Fatal(err)
			}
			if got := dec.hashAt(pos); got != nn.hashAt(pos) {
				t.Fatalf("group re-hash mismatch at pos %d", pos)
			}
			if h, err := DeserializeAndHash(blob); err != nil || h != nn.hashAt(pos) {
				t.Fatalf("DeserializeAndHash mismatch: %v", err)
			}
		case *branchNode:
			blob := serializeNode(nn, pos)
			dec, err := decodeNode(blob)
			if err != nil {
				t.Fatal(err)
			}
			if got := dec.hashAt(pos); got != nn.hashAt(pos) {
				t.Fatalf("branch re-hash mismatch at pos %d", pos)
			}
			child := pos + nn.prefix.n + 1
			walkCheck(nn.left, child)
			walkCheck(nn.right, child)
		}
	}
	walkCheck(tr.root, 0)
}

func uint256Zero() *uint256.Int { return new(uint256.Int) }

// TestStackBuilderVsIncremental checks the sorted-stream bulk builder
// against incremental insertion: same root, same record set.
func TestStackBuilderVsIncremental(t *testing.T) {
	for _, seed := range []int64{1, 5, 8297, 20260728} {
		rng := rand.New(rand.NewSource(seed))
		model := make(map[string][]byte)
		for op := 0; op < 250; op++ {
			key := randomConformantKey(rng)
			var value [32]byte
			rng.Read(value[:])
			model[string(key)] = append([]byte{}, value[:]...)
		}
		keys := make([]string, 0, len(model))
		for k := range model {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Incremental reference.
		inc := newTestTrie()
		for _, k := range keys {
			setKey(t, inc, []byte(k), model[k])
		}
		want := inc.Hash()

		// Bulk build, collecting emitted records.
		built := make(map[string][]byte)
		b := NewStackBuilder(func(path []byte, hash common.Hash, blob []byte) {
			built[string(path)] = append([]byte{}, blob...)
		})
		for _, k := range keys {
			if err := b.Add([]byte(k), model[k]); err != nil {
				t.Fatal(err)
			}
		}
		if got := b.Finish(); got != want {
			t.Fatalf("seed %d: builder root %x, incremental %x", seed, got, want)
		}

		// The emitted record set must equal what a commit of the
		// incremental trie produces.
		_, set := inc.Commit(true)
		if set == nil {
			t.Fatal("incremental commit produced no records")
		}
		if len(set.Nodes) != len(built) {
			t.Fatalf("seed %d: record count %d, builder %d", seed, len(set.Nodes), len(built))
		}
		for path, node := range set.Nodes {
			blob, ok := built[path]
			if !ok {
				t.Fatalf("seed %d: builder missing record at path %x", seed, path)
			}
			if !bytes.Equal(blob, node.Blob) {
				t.Fatalf("seed %d: record blob mismatch at path %x", seed, path)
			}
		}
	}
}

// TestStackBuilderEdges covers the degenerate inputs.
func TestStackBuilderEdges(t *testing.T) {
	// Empty stream.
	if got := NewStackBuilder(nil).Finish(); got != (common.Hash{}) {
		t.Fatalf("empty builder root %x", got)
	}
	// Single leaf.
	key := BasicDataKey(common.Address{1})
	value := bytes.Repeat([]byte{7}, 32)
	b := NewStackBuilder(nil)
	if err := b.Add(key, value); err != nil {
		t.Fatal(err)
	}
	inc := newTestTrie()
	setKey(t, inc, key, value)
	if got, want := b.Finish(), inc.Hash(); got != want {
		t.Fatalf("single leaf: %x want %x", got, want)
	}
	// Ordering and conformance are enforced.
	b2 := NewStackBuilder(nil)
	if err := b2.Add(BasicDataKey(common.Address{2}), value); err != nil {
		t.Fatal(err)
	}
	if err := b2.Add(BasicDataKey(common.Address{2}), value); err == nil {
		t.Fatal("expected rejection of a repeated key")
	}
	if err := NewStackBuilder(nil).Add([]byte{0x00, 0x01}, value); err == nil {
		t.Fatal("expected rejection of a non-conformant key")
	}
}
