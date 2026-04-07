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

package state

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
)

// newTestBinaryHasher creates a binaryHasher backed by an in-memory path database.
func newTestBinaryHasher(t *testing.T, db *triedb.Database, root common.Hash, cfg hasherTestConfig) *binaryHasher {
	t.Helper()

	h, err := newBinaryHasher(root, db, cfg.prefetch, cfg.prefetchRead)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h.TermPrefetch() })
	return h
}

// commitAndReopenBinary commits the hasher's state and reopens a fresh hasher
// from the committed root. This simulates a block boundary.
func commitAndReopenBinary(t *testing.T, h *binaryHasher, cfg hasherTestConfig) *binaryHasher {
	t.Helper()

	root, nodes, _, err := h.Commit()
	if err != nil {
		t.Fatal(err)
	}
	if nodes != nil {
		if err := h.db.Update(root, h.root, 0, nodes, triedb.NewStateSet()); err != nil {
			t.Fatal(err)
		}
		if err := h.db.Commit(root, false); err != nil {
			t.Fatal(err)
		}
	}
	h2, err := newBinaryHasher(root, h.db, cfg.prefetch, cfg.prefetchRead)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h2.TermPrefetch() })
	return h2
}

// makeBinaryBaseState creates a non-empty state as the starting point for tests.
// The base contains:
//   - addr1: nonce=1, balance=100, storage={slot1: val1, slot2: val2}
//   - addr2: nonce=2, balance=200, no storage
//
// The state is committed and flushed so the hasher returned opens from disk.
func makeBinaryBaseState(t *testing.T, cfg hasherTestConfig) *binaryHasher {
	t.Helper()

	noPrefetch := hasherTestConfig{"base", false, false}
	db := triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.VerkleDefaults)
	h := newTestBinaryHasher(t, db, types.EmptyBinaryHash, noPrefetch)

	if err := h.UpdateStorage(hasherAddr1, []common.Hash{hasherSlot1, hasherSlot2}, []common.Hash{hasherVal1, hasherVal2}); err != nil {
		t.Fatal(err)
	}
	if err := h.UpdateAccount(
		[]common.Address{hasherAddr1, hasherAddr2},
		[]AccountMut{hasherAccount(1, 100), hasherAccount(2, 200)},
	); err != nil {
		t.Fatal(err)
	}
	return commitAndReopenBinary(t, h, cfg)
}

// TestBinaryHasherBasic verifies that mutating storage and accounts on top of
// a non-empty base state produces a deterministic, non-empty root and that the
// root survives a commit+reopen cycle.
func TestBinaryHasherBasic(t *testing.T) {
	for _, cfg := range hasherTestConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			h := makeBinaryBaseState(t, cfg)

			if cfg.prefetch {
				h.PrefetchStorage(hasherAddr1, []common.Hash{hasherSlot3}, false)
				h.PrefetchAccount([]common.Address{hasherAddr1, hasherAddr3}, false)
			}
			if err := h.UpdateStorage(hasherAddr1, []common.Hash{hasherSlot3}, []common.Hash{hasherVal3}); err != nil {
				t.Fatal(err)
			}
			if err := h.UpdateAccount(
				[]common.Address{hasherAddr1, hasherAddr3},
				[]AccountMut{hasherAccount(1, 100), hasherAccount(3, 300)},
			); err != nil {
				t.Fatal(err)
			}
			root := h.Hash()
			if root == types.EmptyRootHash {
				t.Fatal("expected non-empty root after mutations")
			}
			h2 := commitAndReopenBinary(t, h, cfg)
			if h2.Hash() != root {
				t.Fatalf("root mismatch after reopen: got %x, want %x", h2.Hash(), root)
			}
		})
	}
}

// TestBinaryHasherPrefetchReadOnly verifies that read-only prefetching (for
// accounts and storage that are never subsequently mutated) does not corrupt
// state. Both prefetchRead=true (requests are processed) and prefetchRead=false
// (requests are dropped) are tested.
func TestBinaryHasherPrefetchReadOnly(t *testing.T) {
	for _, prefetchRead := range []bool{false, true} {
		name := "readDropped"
		if prefetchRead {
			name = "readProcessed"
		}
		t.Run(name, func(t *testing.T) {
			cfg := hasherTestConfig{name, true, prefetchRead}
			h := makeBinaryBaseState(t, cfg)
			rootBefore := h.Hash()

			// Prefetch addr1's account and storage (read-only).
			h.PrefetchAccount([]common.Address{hasherAddr1, hasherAddr2}, true)
			h.PrefetchStorage(hasherAddr1, []common.Hash{hasherSlot1, hasherSlot2}, true)

			// Only mutate addr2 — addr1's prefetched data is never written.
			if err := h.UpdateAccount(
				[]common.Address{hasherAddr2},
				[]AccountMut{hasherAccount(2, 300)},
			); err != nil {
				t.Fatal(err)
			}
			root := h.Hash()
			if root == rootBefore {
				t.Fatal("expected root to change after balance update")
			}
			h2 := commitAndReopenBinary(t, h, hasherTestConfig{"verify", false, false})
			if h2.Hash() != root {
				t.Fatalf("root mismatch: got %x, want %x", h2.Hash(), root)
			}
		})
	}
}

// TestBinaryHasherPrefetchDeterminism verifies that the resulting root is
// identical across all prefetch configurations for the same set of mutations.
func TestBinaryHasherPrefetchDeterminism(t *testing.T) {
	var roots []common.Hash
	for _, cfg := range hasherTestConfigs {
		h := makeBinaryBaseState(t, cfg)

		if cfg.prefetch {
			h.PrefetchAccount([]common.Address{hasherAddr1, hasherAddr3}, false)
			h.PrefetchStorage(hasherAddr1, []common.Hash{hasherSlot3}, false)
			h.PrefetchStorage(hasherAddr3, []common.Hash{hasherSlot1}, false)
		}
		if err := h.UpdateStorage(hasherAddr1, []common.Hash{hasherSlot3}, []common.Hash{hasherVal3}); err != nil {
			t.Fatal(err)
		}
		if err := h.UpdateStorage(hasherAddr3, []common.Hash{hasherSlot1}, []common.Hash{hasherVal1}); err != nil {
			t.Fatal(err)
		}
		if err := h.UpdateAccount(
			[]common.Address{hasherAddr1, hasherAddr3},
			[]AccountMut{hasherAccount(1, 100), hasherAccount(3, 300)},
		); err != nil {
			t.Fatal(err)
		}
		roots = append(roots, h.Hash())
	}
	for i := 1; i < len(roots); i++ {
		if roots[i] != roots[0] {
			t.Fatalf("root diverged: config[0]=%x config[%d]=%x", roots[0], i, roots[i])
		}
	}
}

// TestBinaryHasherCopy verifies that Copy produces an independent snapshot:
// mutations on the copy must not affect the original's hash.
func TestBinaryHasherCopy(t *testing.T) {
	cfg := hasherTestConfig{"prefetchAll", true, true}
	h := makeBinaryBaseState(t, cfg)

	h.PrefetchAccount([]common.Address{hasherAddr1}, false)
	h.PrefetchStorage(hasherAddr1, []common.Hash{hasherSlot3}, false)
	if err := h.UpdateStorage(hasherAddr1, []common.Hash{hasherSlot3}, []common.Hash{hasherVal3}); err != nil {
		t.Fatal(err)
	}
	if err := h.UpdateAccount([]common.Address{hasherAddr1}, []AccountMut{hasherAccount(1, 100)}); err != nil {
		t.Fatal(err)
	}
	origRoot := h.Hash()

	cpy := h.Copy()
	defer cpy.(*binaryHasher).TermPrefetch()

	// Mutate the copy: delete slot3, add slot2 with new value.
	if err := cpy.UpdateStorage(hasherAddr1, []common.Hash{hasherSlot3, hasherSlot2}, []common.Hash{{}, hasherVal3}); err != nil {
		t.Fatal(err)
	}
	if err := cpy.UpdateAccount([]common.Address{hasherAddr1}, []AccountMut{hasherAccount(1, 100)}); err != nil {
		t.Fatal(err)
	}
	if cpy.Hash() == origRoot {
		t.Fatal("copy should diverge after mutation")
	}
	if h.Hash() != origRoot {
		t.Fatal("original root changed after mutating copy")
	}
}

// TestBinaryHasherWitness verifies that the witness returned by CollectWitness
// contains trie nodes for accessed accounts and storage. When read-only
// prefetching is enabled, the prefetched (but never written) data must also
// appear in the witness.
func TestBinaryHasherWitness(t *testing.T) {
	// Collect witness WITHOUT read-prefetching: only mutated paths are tracked.
	collectWitness := func(prefetchRead bool) int {
		cfg := hasherTestConfig{"witness", true, prefetchRead}
		h := makeBinaryBaseState(t, cfg)

		// Read-only prefetch of addr1 account and slot1 (never mutated below).
		h.PrefetchAccount([]common.Address{hasherAddr1}, true)
		h.PrefetchStorage(hasherAddr1, []common.Hash{hasherSlot1}, true)

		// Mutate only addr2 (no storage).
		if err := h.UpdateAccount(
			[]common.Address{hasherAddr2},
			[]AccountMut{hasherAccount(2, 300)},
		); err != nil {
			t.Fatal(err)
		}
		h.Hash()

		witness := &stateless.Witness{
			Codes: make(map[string]struct{}),
			State: make(map[string]struct{}),
		}
		h.CollectWitness(witness)
		return len(witness.State)
	}
	nodesWithoutRead := collectWitness(false)
	nodesWithRead := collectWitness(true)

	if nodesWithoutRead == 0 {
		t.Fatal("witness should contain trie nodes even without read prefetching")
	}
	if nodesWithRead <= nodesWithoutRead {
		t.Fatalf("read-only prefetching should add extra nodes to witness: got %d (with read) vs %d (without)", nodesWithRead, nodesWithoutRead)
	}
}

// TestBinaryHasherLeafProduction verifies that binaryHasher implements
// LeafProducer and reports stem writes corresponding to each trie
// mutation. Covers the three mutation kinds the hasher performs:
// account update, storage update, and account delete.
func TestBinaryHasherLeafProduction(t *testing.T) {
	db := triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.VerkleDefaults)
	h := newTestBinaryHasher(t, db, types.EmptyBinaryHash, hasherTestConfig{"leaf", false, false})

	// Type assertion: binaryHasher must satisfy LeafProducer.
	lp, ok := Hasher(h).(LeafProducer)
	if !ok {
		t.Fatal("binaryHasher should implement LeafProducer")
	}

	// --- Account update: expect two writes (BasicData + CodeHash) ---
	if err := h.UpdateAccount(
		[]common.Address{hasherAddr1},
		[]AccountMut{hasherAccount(1, 100)},
	); err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}
	writes := lp.DrainStemWrites()
	if len(writes) != 2 {
		t.Fatalf("UpdateAccount: got %d stem writes, want 2 (BasicData + CodeHash)", len(writes))
	}
	// Offsets 0 and 1 respectively, and the BasicData stem matches the
	// CodeHash stem (same address → same 31-byte stem).
	if writes[0].Offset != bintrie.BasicDataLeafKey {
		t.Errorf("write[0].Offset = %d, want %d (BasicDataLeafKey)", writes[0].Offset, bintrie.BasicDataLeafKey)
	}
	if writes[1].Offset != bintrie.CodeHashLeafKey {
		t.Errorf("write[1].Offset = %d, want %d (CodeHashLeafKey)", writes[1].Offset, bintrie.CodeHashLeafKey)
	}
	if writes[0].Stem != writes[1].Stem {
		t.Errorf("stems differ: %x vs %x", writes[0].Stem, writes[1].Stem)
	}
	if len(writes[0].Value) != 32 {
		t.Errorf("write[0].Value length = %d, want 32", len(writes[0].Value))
	}
	if len(writes[1].Value) != 32 {
		t.Errorf("write[1].Value length = %d, want 32", len(writes[1].Value))
	}
	// The code hash leaf should be the empty-code hash (non-zero).
	if !bytes.Equal(writes[1].Value, types.EmptyCodeHash.Bytes()) {
		t.Errorf("write[1].Value = %x, want empty code hash %x", writes[1].Value, types.EmptyCodeHash.Bytes())
	}

	// --- Drain again: should be empty (drain is destructive) ---
	if again := lp.DrainStemWrites(); len(again) != 0 {
		t.Fatalf("second drain should be empty, got %d writes", len(again))
	}

	// --- Storage update: non-zero value produces one write ---
	if err := h.UpdateStorage(hasherAddr1, []common.Hash{hasherSlot1}, []common.Hash{hasherVal1}); err != nil {
		t.Fatalf("UpdateStorage: %v", err)
	}
	writes = lp.DrainStemWrites()
	if len(writes) != 1 {
		t.Fatalf("UpdateStorage: got %d writes, want 1", len(writes))
	}
	// The recorded value should match hasherVal1 (a common.Hash), which
	// is already 32 bytes wide.
	if !bytes.Equal(writes[0].Value, hasherVal1[:]) {
		t.Errorf("UpdateStorage value: got %x, want %x", writes[0].Value, hasherVal1)
	}

	// --- Storage "delete" (zero value): one write with 32 zero bytes ---
	if err := h.UpdateStorage(hasherAddr1, []common.Hash{hasherSlot1}, []common.Hash{{}}); err != nil {
		t.Fatalf("UpdateStorage (zero): %v", err)
	}
	writes = lp.DrainStemWrites()
	if len(writes) != 1 {
		t.Fatalf("UpdateStorage (zero): got %d writes, want 1", len(writes))
	}
	var zeros [32]byte
	if !bytes.Equal(writes[0].Value, zeros[:]) {
		t.Errorf("zero-value storage write should record 32 zero bytes, got %x", writes[0].Value)
	}

	// --- Account delete: two writes with nil values ---
	if err := h.UpdateAccount(
		[]common.Address{hasherAddr1},
		[]AccountMut{{Account: nil}},
	); err != nil {
		t.Fatalf("UpdateAccount delete: %v", err)
	}
	writes = lp.DrainStemWrites()
	if len(writes) != 2 {
		t.Fatalf("delete: got %d writes, want 2 (BasicData + CodeHash clear)", len(writes))
	}
	for i, w := range writes {
		if w.Value != nil {
			t.Errorf("delete write[%d] should have nil Value (clear), got %x", i, w.Value)
		}
	}
	if writes[0].Offset != bintrie.BasicDataLeafKey || writes[1].Offset != bintrie.CodeHashLeafKey {
		t.Errorf("delete offsets: got %d,%d, want %d,%d", writes[0].Offset, writes[1].Offset, bintrie.BasicDataLeafKey, bintrie.CodeHashLeafKey)
	}
}

// TestMerkleHasherNoLeafProducer verifies that merkleHasher does NOT
// implement LeafProducer — the interface is strictly opt-in and the MPT
// path has no concept of stem writes.
func TestMerkleHasherNoLeafProducer(t *testing.T) {
	db := triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil)
	h, err := newMerkleHasher(types.EmptyRootHash, db, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := Hasher(h).(LeafProducer); ok {
		t.Fatal("merkleHasher should NOT implement LeafProducer")
	}
}

// TestStateUpdateEncodeBinaryFromLeaves verifies that stateUpdate.encodeBinary
// turns a slice of StemWrite values into the per-offset accountData map that
// pathdb's bintrie codec consumes. Three things matter:
//
//  1. Every leaf becomes one accountData entry, keyed by stem||offset.
//  2. nil-value leaves (account/storage deletes) become nil entries.
//  3. Non-nil leaves are deeply copied — encodeBinary must not retain
//     pointers into the hasher's internal slab.
//
// storages/storageOrigin/accountOrigin remain empty: the bintrie path uses
// only accountData (per the layered-read design) and does not yet support
// state-history rollback.
func TestStateUpdateEncodeBinaryFromLeaves(t *testing.T) {
	// Build a small leaves slice covering each kind of write the binary
	// hasher emits: account update (BasicData + CodeHash), storage write,
	// and a delete (nil value).
	var (
		stemA [bintrie.StemSize]byte
		stemB [bintrie.StemSize]byte
	)
	for i := range stemA {
		stemA[i] = byte(0x10 + i)
		stemB[i] = byte(0xA0 + i)
	}
	basicDataValue := bytes.Repeat([]byte{0xAA}, 32)
	codeHashValue := bytes.Repeat([]byte{0xBB}, 32)
	storageValue := bytes.Repeat([]byte{0xCC}, 32)

	leaves := []StemWrite{
		// Account update at stemA: BasicData + CodeHash.
		{Stem: stemA, Offset: bintrie.BasicDataLeafKey, Value: basicDataValue},
		{Stem: stemA, Offset: bintrie.CodeHashLeafKey, Value: codeHashValue},
		// Storage write at stemB.
		{Stem: stemB, Offset: 7, Value: storageValue},
		// Account delete at a third stem (nil values clear offsets 0+1).
		{Stem: [bintrie.StemSize]byte{0xFF, 0xFF}, Offset: bintrie.BasicDataLeafKey, Value: nil},
		{Stem: [bintrie.StemSize]byte{0xFF, 0xFF}, Offset: bintrie.CodeHashLeafKey, Value: nil},
	}

	su := &stateUpdate{leaves: leaves}
	accounts, accountOrigin, storages, storageOrigin, err := su.encodeBinary()
	if err != nil {
		t.Fatalf("encodeBinary: %v", err)
	}

	if len(accounts) != len(leaves) {
		t.Fatalf("accounts len = %d, want %d", len(accounts), len(leaves))
	}
	if len(storages) != 0 {
		t.Errorf("storages should be empty for bintrie, got %d entries", len(storages))
	}
	if len(accountOrigin) != 0 || len(storageOrigin) != 0 {
		t.Errorf("origin maps should be empty for bintrie")
	}

	// Check each leaf round-trips through the map under its full key.
	for i, w := range leaves {
		var fullKey common.Hash
		copy(fullKey[:bintrie.StemSize], w.Stem[:])
		fullKey[bintrie.StemSize] = w.Offset
		got, ok := accounts[fullKey]
		if !ok {
			t.Errorf("leaf %d: missing key %x", i, fullKey)
			continue
		}
		if w.Value == nil {
			if got != nil {
				t.Errorf("leaf %d: nil leaf became %x", i, got)
			}
			continue
		}
		if !bytes.Equal(got, w.Value) {
			t.Errorf("leaf %d: got %x, want %x", i, got, w.Value)
		}
		// Aliasing check: the encoder must own its bytes.
		if len(got) > 0 && &got[0] == &w.Value[0] {
			t.Errorf("leaf %d: encodeBinary aliased the input slice", i)
		}
	}
}
