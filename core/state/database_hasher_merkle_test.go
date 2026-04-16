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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

var (
	hasherAddr1 = common.HexToAddress("0x1111111111111111111111111111111111111111")
	hasherAddr2 = common.HexToAddress("0x2222222222222222222222222222222222222222")
	hasherAddr3 = common.HexToAddress("0x3333333333333333333333333333333333333333")

	hasherSlot1 = common.HexToHash("0x01")
	hasherSlot2 = common.HexToHash("0x02")
	hasherSlot3 = common.HexToHash("0x03")

	hasherVal1 = common.HexToHash("0xaa")
	hasherVal2 = common.HexToHash("0xbb")
	hasherVal3 = common.HexToHash("0xcc")
)

// hasherTestConfig captures the prefetch flags varied across subtests.
type hasherTestConfig struct {
	name         string
	prefetch     bool
	prefetchRead bool
}

// hasherTestConfigs enumerates the interesting (prefetch, prefetchRead) combinations:
//   - no prefetch at all
//   - prefetch writes only (read prefetch requests are dropped)
//   - prefetch reads and writes
var hasherTestConfigs = []hasherTestConfig{
	{"noPrefetch", false, false},
	{"prefetchWriteOnly", true, false},
	{"prefetchAll", true, true},
}

func hasherAccount(nonce uint64, balance uint64) AccountMut {
	return AccountMut{
		Account: &Account{
			Nonce:    nonce,
			Balance:  uint256.NewInt(balance),
			CodeHash: types.EmptyCodeHash.Bytes(),
		},
	}
}

func hasherDeleteAccount() AccountMut {
	return AccountMut{Account: nil}
}

// newTestHasher creates a merkleHasher backed by an in-memory database.
func newTestHasher(t *testing.T, db *triedb.Database, root common.Hash, cfg hasherTestConfig) *merkleHasher {
	t.Helper()

	h, err := newMerkleHasher(root, db, cfg.prefetch, cfg.prefetchRead)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h.TermPrefetch() })
	return h
}

// commitAndReopen commits the hasher's state and reopens a fresh hasher from
// the committed root. This simulates a block boundary.
func commitAndReopen(t *testing.T, h *merkleHasher, cfg hasherTestConfig) *merkleHasher {
	t.Helper()

	root, nodes, _, err := h.Commit()
	if err != nil {
		t.Fatal(err)
	}
	if nodes != nil {
		if err := h.db.Update(root, h.root, 0, nodes, nil); err != nil {
			t.Fatal(err)
		}
		if err := h.db.Commit(root, false); err != nil {
			t.Fatal(err)
		}
	}
	h2, err := newMerkleHasher(root, h.db, cfg.prefetch, cfg.prefetchRead)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h2.TermPrefetch() })
	return h2
}

// makeBaseState creates a non-empty state as the starting point for tests.
// The base contains:
//   - addr1: nonce=1, balance=100, storage={slot1: val1, slot2: val2}
//   - addr2: nonce=2, balance=200, no storage
//
// The state is committed and flushed so the hasher returned opens from disk,
// exercising rootReader and existing-trie code paths.
func makeBaseState(t *testing.T, cfg hasherTestConfig) *merkleHasher {
	t.Helper()

	noPrefetch := hasherTestConfig{"base", false, false}
	db := triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil)
	h := newTestHasher(t, db, types.EmptyRootHash, noPrefetch)

	if err := h.UpdateStorage(hasherAddr1, []common.Hash{hasherSlot1, hasherSlot2}, []common.Hash{hasherVal1, hasherVal2}); err != nil {
		t.Fatal(err)
	}
	if err := h.UpdateAccount(
		[]common.Address{hasherAddr1, hasherAddr2},
		[]AccountMut{hasherAccount(1, 100), hasherAccount(2, 200)},
	); err != nil {
		t.Fatal(err)
	}
	return commitAndReopen(t, h, cfg)
}

// TestMerkleHasherBasic verifies that mutating storage and accounts on top of
// a non-empty base state produces a deterministic, non-empty root and that the
// root survives a commit+reopen cycle.
func TestMerkleHasherBasic(t *testing.T) {
	for _, cfg := range hasherTestConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			h := makeBaseState(t, cfg)

			if cfg.prefetch {
				h.PrefetchStorage(hasherAddr1, []common.Hash{hasherSlot3}, false)
				h.PrefetchAccount([]common.Address{hasherAddr1, hasherAddr3}, false)
			}
			// Add slot3 to addr1 and create addr3.
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
			h2 := commitAndReopen(t, h, cfg)
			if h2.Hash() != root {
				t.Fatalf("root mismatch after reopen: got %x, want %x", h2.Hash(), root)
			}
		})
	}
}

// TestMerkleHasherPrefetchReadOnly verifies that read-only prefetching (for
// accounts and storage that are never subsequently mutated) does not corrupt
// state and does not leak goroutines. Both prefetchRead=true (requests are
// processed) and prefetchRead=false (requests are dropped) are tested.
func TestMerkleHasherPrefetchReadOnly(t *testing.T) {
	for _, prefetchRead := range []bool{false, true} {
		name := "readDropped"
		if prefetchRead {
			name = "readProcessed"
		}
		t.Run(name, func(t *testing.T) {
			cfg := hasherTestConfig{name, true, prefetchRead}
			h := makeBaseState(t, cfg)
			rootBefore := h.Hash()

			// Prefetch addr1's account and storage (read-only). Whether
			// these are actually processed depends on prefetchRead.
			h.PrefetchAccount([]common.Address{hasherAddr1, hasherAddr2}, true)
			h.PrefetchStorage(hasherAddr1, []common.Hash{hasherSlot1, hasherSlot2}, true)

			// Only mutate addr2 (no storage) — addr1's prefetched tries
			// are never accessed through a shadow method.
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
			h2 := commitAndReopen(t, h, hasherTestConfig{"verify", false, false})
			if h2.Hash() != root {
				t.Fatalf("root mismatch: got %x, want %x", h2.Hash(), root)
			}
		})
	}
}

// TestMerkleHasherDeleteAccount verifies that deleting an account with storage
// produces an empty storage root in the commit result, with Prev reflecting
// the original non-empty root.
func TestMerkleHasherDeleteAccount(t *testing.T) {
	for _, cfg := range hasherTestConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			h := makeBaseState(t, cfg)

			if cfg.prefetch {
				h.PrefetchAccount([]common.Address{hasherAddr1}, false)
				h.PrefetchStorage(hasherAddr1, []common.Hash{hasherSlot1, hasherSlot2}, false)
			}
			// Delete addr1 (which has storage slots 1,2).
			if err := h.UpdateAccount(
				[]common.Address{hasherAddr1},
				[]AccountMut{hasherDeleteAccount()},
			); err != nil {
				t.Fatal(err)
			}
			_, _, storageRoots, err := h.Commit()
			if err != nil {
				t.Fatal(err)
			}
			sr, ok := storageRoots[hasherAddr1]
			if !ok {
				t.Fatal("deleted account missing from storageRoots")
			}
			if sr.Hash != types.EmptyRootHash {
				t.Fatalf("deleted account storage root: got %x, want EmptyRootHash", sr.Hash)
			}
			if sr.Prev == types.EmptyRootHash {
				t.Fatal("deleted account Prev should be non-empty (had storage)")
			}
		})
	}
}

// TestMerkleHasherDeleteRecreate verifies that deleting an account and
// recreating it with different storage in the same block produces a correct
// root that survives a commit+reopen cycle. The storageRoots report must show
// the original Prev and a new Hash.
func TestMerkleHasherDeleteRecreate(t *testing.T) {
	for _, cfg := range hasherTestConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			h := makeBaseState(t, cfg)

			if cfg.prefetch {
				h.PrefetchAccount([]common.Address{hasherAddr1}, false)
				h.PrefetchStorage(hasherAddr1, []common.Hash{hasherSlot1, hasherSlot2}, false)
			}
			// Delete addr1.
			if err := h.UpdateAccount([]common.Address{hasherAddr1}, []AccountMut{hasherDeleteAccount()}); err != nil {
				t.Fatal(err)
			}
			// Recreate with slot3 only.
			if err := h.UpdateStorage(hasherAddr1, []common.Hash{hasherSlot3}, []common.Hash{hasherVal3}); err != nil {
				t.Fatal(err)
			}
			if err := h.UpdateAccount([]common.Address{hasherAddr1}, []AccountMut{hasherAccount(10, 500)}); err != nil {
				t.Fatal(err)
			}
			root := h.Hash()
			if root == types.EmptyRootHash {
				t.Fatal("expected non-empty root after recreate")
			}
			h2 := commitAndReopen(t, h, hasherTestConfig{"verify", false, false})

			sr := h.storageRoots[hasherAddr1]
			if sr.Hash == types.EmptyRootHash {
				t.Fatal("recreated account should have non-empty storage root")
			}
			if sr.Prev == types.EmptyRootHash {
				t.Fatal("Prev should reflect the pre-deletion storage root")
			}
			if sr.Hash == sr.Prev {
				t.Fatal("Hash and Prev should differ after delete+recreate with different slots")
			}
			if h2.Hash() != root {
				t.Fatalf("root mismatch after reopen: got %x, want %x", h2.Hash(), root)
			}
		})
	}
}

// TestMerkleHasherPrefetchDeterminism verifies that the resulting root is
// identical across all prefetch configurations for the same set of mutations.
func TestMerkleHasherPrefetchDeterminism(t *testing.T) {
	var roots []common.Hash
	for _, cfg := range hasherTestConfigs {
		h := makeBaseState(t, cfg)

		if cfg.prefetch {
			h.PrefetchAccount([]common.Address{hasherAddr1, hasherAddr3}, false)
			h.PrefetchStorage(hasherAddr1, []common.Hash{hasherSlot3}, false)
			h.PrefetchStorage(hasherAddr3, []common.Hash{hasherSlot1}, false)
		}
		// Add slot3 to addr1, create addr3 with slot1.
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

// TestMerkleHasherCommitStorageRoots exhaustively checks the Prev/Hash pairs
// returned by Commit for every interesting mutation pattern:
//
//	(1) delete account with non-empty storage
//	(2) delete account with empty storage
//	(3) delete + recreate with new non-empty storage
//	(4) delete + recreate without storage (empty→empty after recreate)
//	(5) delete + recreate: originally empty storage, recreated with storage
//	(6) mutate account only, no storage (empty storage throughout)
//	(7) mutate account only, non-empty storage unchanged
//	(8) mutate account with modified storage
func TestMerkleHasherCommitStorageRoots(t *testing.T) {
	var (
		// Addresses for each case — distinct so they don't interfere.
		addrDeleteNonEmpty   = common.HexToAddress("0xaa01") // (1)
		addrDeleteEmpty      = common.HexToAddress("0xaa02") // (2)
		addrRecreateStorage  = common.HexToAddress("0xaa03") // (3)
		addrRecreateNoStore  = common.HexToAddress("0xaa04") // (4)
		addrRecreateFromNone = common.HexToAddress("0xaa05") // (5)
		addrMutateNoStorage  = common.HexToAddress("0xaa06") // (6)
		addrMutateKeepStore  = common.HexToAddress("0xaa07") // (7)
		addrMutateModStore   = common.HexToAddress("0xaa08") // (8)
	)
	for _, cfg := range hasherTestConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			// ---------- base state (committed to disk) ----------
			noPrefetch := hasherTestConfig{"base", false, false}
			db := triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil)
			base := newTestHasher(t, db, types.EmptyRootHash, noPrefetch)

			// Accounts with storage.
			for _, addr := range []common.Address{addrDeleteNonEmpty, addrRecreateStorage, addrRecreateNoStore, addrMutateKeepStore, addrMutateModStore} {
				if err := base.UpdateStorage(addr, []common.Hash{hasherSlot1}, []common.Hash{hasherVal1}); err != nil {
					t.Fatal(err)
				}
			}
			// All accounts (some with storage above, some without).
			allAddrs := []common.Address{
				addrDeleteNonEmpty, addrDeleteEmpty,
				addrRecreateStorage, addrRecreateNoStore, addrRecreateFromNone,
				addrMutateNoStorage, addrMutateKeepStore, addrMutateModStore,
			}
			allAccounts := make([]AccountMut, len(allAddrs))
			for i := range allAccounts {
				allAccounts[i] = hasherAccount(1, 100)
			}
			if err := base.UpdateAccount(allAddrs, allAccounts); err != nil {
				t.Fatal(err)
			}
			h := commitAndReopen(t, base, cfg)

			// ---------- block mutations ----------

			// (1) Delete account with non-empty storage.
			// (2) Delete account with empty storage.
			if err := h.UpdateAccount(
				[]common.Address{addrDeleteNonEmpty, addrDeleteEmpty},
				[]AccountMut{hasherDeleteAccount(), hasherDeleteAccount()},
			); err != nil {
				t.Fatal(err)
			}
			// (3) Delete + recreate with new storage.
			if err := h.UpdateAccount([]common.Address{addrRecreateStorage}, []AccountMut{hasherDeleteAccount()}); err != nil {
				t.Fatal(err)
			}
			if err := h.UpdateStorage(addrRecreateStorage, []common.Hash{hasherSlot2}, []common.Hash{hasherVal2}); err != nil {
				t.Fatal(err)
			}
			if err := h.UpdateAccount([]common.Address{addrRecreateStorage}, []AccountMut{hasherAccount(2, 200)}); err != nil {
				t.Fatal(err)
			}
			// (4) Delete + recreate without storage (had storage before).
			if err := h.UpdateAccount([]common.Address{addrRecreateNoStore}, []AccountMut{hasherDeleteAccount()}); err != nil {
				t.Fatal(err)
			}
			if err := h.UpdateAccount([]common.Address{addrRecreateNoStore}, []AccountMut{hasherAccount(2, 200)}); err != nil {
				t.Fatal(err)
			}
			// (5) Delete + recreate: originally no storage, recreated with storage.
			if err := h.UpdateAccount([]common.Address{addrRecreateFromNone}, []AccountMut{hasherDeleteAccount()}); err != nil {
				t.Fatal(err)
			}
			if err := h.UpdateStorage(addrRecreateFromNone, []common.Hash{hasherSlot1}, []common.Hash{hasherVal3}); err != nil {
				t.Fatal(err)
			}
			if err := h.UpdateAccount([]common.Address{addrRecreateFromNone}, []AccountMut{hasherAccount(2, 200)}); err != nil {
				t.Fatal(err)
			}
			// (6) Mutate account only, no storage.
			if err := h.UpdateAccount([]common.Address{addrMutateNoStorage}, []AccountMut{hasherAccount(2, 999)}); err != nil {
				t.Fatal(err)
			}
			// (7) Mutate account, non-empty storage unchanged.
			if err := h.UpdateAccount([]common.Address{addrMutateKeepStore}, []AccountMut{hasherAccount(2, 888)}); err != nil {
				t.Fatal(err)
			}
			// (8) Mutate account with modified storage.
			if err := h.UpdateStorage(addrMutateModStore, []common.Hash{hasherSlot1}, []common.Hash{hasherVal2}); err != nil {
				t.Fatal(err)
			}
			if err := h.UpdateAccount([]common.Address{addrMutateModStore}, []AccountMut{hasherAccount(2, 777)}); err != nil {
				t.Fatal(err)
			}
			_, _, roots, err := h.Commit()
			if err != nil {
				t.Fatal(err)
			}
			empty := types.EmptyRootHash

			// (1) Deleted, had storage: Prev=non-empty, Hash=empty.
			sr := roots[addrDeleteNonEmpty]
			if sr.Prev == empty {
				t.Fatal("(1) Prev should be non-empty for deleted account that had storage")
			}
			if sr.Hash != empty {
				t.Fatal("(1) Hash should be EmptyRootHash after deletion")
			}
			// (2) Deleted, had no storage: Prev=empty, Hash=empty.
			sr = roots[addrDeleteEmpty]
			if sr.Prev != empty || sr.Hash != empty {
				t.Fatalf("(2) expected both EmptyRootHash, got Prev=%x Hash=%x", sr.Prev, sr.Hash)
			}
			// (3) Delete+recreate with new storage: Prev=non-empty(original), Hash=non-empty(new), differ.
			sr = roots[addrRecreateStorage]
			if sr.Prev == empty {
				t.Fatal("(3) Prev should be non-empty (had storage before deletion)")
			}
			if sr.Hash == empty {
				t.Fatal("(3) Hash should be non-empty (recreated with storage)")
			}
			if sr.Hash == sr.Prev {
				t.Fatal("(3) Hash and Prev should differ (different storage contents)")
			}
			// (4) Delete+recreate without storage (originally had storage): Prev=non-empty, Hash=empty.
			sr = roots[addrRecreateNoStore]
			if sr.Prev == empty {
				t.Fatal("(4) Prev should be non-empty (had storage before deletion)")
			}
			if sr.Hash != empty {
				t.Fatal("(4) Hash should be EmptyRootHash (recreated without storage)")
			}
			// (5) Delete+recreate: originally no storage, recreated with storage: Prev=empty, Hash=non-empty.
			sr = roots[addrRecreateFromNone]
			if sr.Prev != empty {
				t.Fatal("(5) Prev should be EmptyRootHash (no storage before deletion)")
			}
			if sr.Hash == empty {
				t.Fatal("(5) Hash should be non-empty (recreated with storage)")
			}
			// (6) Mutate account only, no storage: Prev=empty, Hash=empty.
			sr = roots[addrMutateNoStorage]
			if sr.Prev != empty || sr.Hash != empty {
				t.Fatalf("(6) expected both EmptyRootHash, got Prev=%x Hash=%x", sr.Prev, sr.Hash)
			}
			// (7) Mutate account, storage unchanged: Prev=non-empty, Hash=non-empty, Prev==Hash.
			sr = roots[addrMutateKeepStore]
			if sr.Prev == empty {
				t.Fatal("(7) Prev should be non-empty (has storage)")
			}
			if sr.Hash == empty {
				t.Fatal("(7) Hash should be non-empty (storage unchanged)")
			}
			if sr.Prev != sr.Hash {
				t.Fatal("(7) Prev and Hash should be equal (storage was not modified)")
			}
			// (8) Mutate account with modified storage: Prev=non-empty, Hash=non-empty, differ.
			sr = roots[addrMutateModStore]
			if sr.Prev == empty {
				t.Fatal("(8) Prev should be non-empty (had storage)")
			}
			if sr.Hash == empty {
				t.Fatal("(8) Hash should be non-empty (storage modified, not cleared)")
			}
			if sr.Prev == sr.Hash {
				t.Fatal("(8) Prev and Hash should differ (storage was modified)")
			}
		})
	}
}

// TestMerkleHasherCopy verifies that Copy produces an independent snapshot:
// mutations on the copy must not affect the original's hash.
func TestMerkleHasherCopy(t *testing.T) {
	cfg := hasherTestConfig{"prefetchAll", true, true}
	h := makeBaseState(t, cfg)

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
	defer cpy.(*merkleHasher).TermPrefetch()

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

// proofNodes collects the raw RLP-encoded trie nodes written by Prove calls.
type proofNodes struct{ nodes [][]byte }

func (p *proofNodes) Put(key []byte, value []byte) error {
	p.nodes = append(p.nodes, common.CopyBytes(value))
	return nil
}
func (p *proofNodes) Delete([]byte) error { return nil }

// TestMerkleHasherWitness verifies that the witness returned by Witness()
// contains every trie node on the Merkle proof path for each accessed account
// and storage slot, including nodes from deleted storage tries.
func TestMerkleHasherWitness(t *testing.T) {
	h := makeBaseState(t, hasherTestConfig{"prefetchAll", true, true})

	// Mutate addr1 storage, then delete and recreate with different
	// storage so that both deletedTries and storageTries are populated.
	h.PrefetchStorage(hasherAddr1, []common.Hash{hasherSlot1}, false)
	if err := h.UpdateStorage(hasherAddr1, []common.Hash{hasherSlot1}, []common.Hash{hasherVal2}); err != nil {
		t.Fatal(err)
	}
	if err := h.UpdateAccount([]common.Address{hasherAddr1}, []AccountMut{hasherDeleteAccount()}); err != nil {
		t.Fatal(err)
	}
	if err := h.UpdateStorage(hasherAddr1, []common.Hash{hasherSlot3}, []common.Hash{hasherVal3}); err != nil {
		t.Fatal(err)
	}
	if err := h.UpdateAccount(
		[]common.Address{hasherAddr1, hasherAddr2},
		[]AccountMut{hasherAccount(10, 500), hasherAccount(2, 300)},
	); err != nil {
		t.Fatal(err)
	}
	witness := &stateless.Witness{
		Codes: make(map[string]struct{}),
		State: make(map[string]struct{}),
	}
	h.CollectWitness(witness)

	if len(witness.State) == 0 {
		t.Fatal("witness should contain trie nodes")
	}
	// Open a separate prover from the same pre-state root. Proofs
	// generated here traverse the same trie paths that the mutating
	// hasher loaded, so every proof node must be in the witness.
	prover, err := newMerkleHasher(h.root, h.db, false, false)
	if err != nil {
		t.Fatal(err)
	}
	defer prover.TermPrefetch()

	// Collect all expected proof nodes into a single set. The union of
	// account proofs (addr1, addr2) and storage proofs (addr1/slot1)
	// should exactly equal witness.State — no missing, no extra.
	expected := make(map[string]struct{})

	for _, addr := range []common.Address{hasherAddr1, hasherAddr2} {
		pn := &proofNodes{}
		if err := prover.ProveAccount(addr, pn); err != nil {
			t.Fatal(err)
		}
		for _, node := range pn.nodes {
			expected[string(node)] = struct{}{}
		}
	}
	// Storage proof for addr1/slot1 (accessed before deletion).
	// Slot2 was in the base state but never read or written during the
	// block, so its leaf node is correctly absent from the witness.
	pn := &proofNodes{}
	if err := prover.ProveStorage(hasherAddr1, hasherSlot1, pn); err != nil {
		t.Fatal(err)
	}
	for _, node := range pn.nodes {
		expected[string(node)] = struct{}{}
	}
	// Every expected proof node must be in the witness.
	for node := range expected {
		if _, ok := witness.State[node]; !ok {
			t.Fatal("proof node missing from witness")
		}
	}
	// The witness must not contain any extra nodes beyond the proofs.
	if len(witness.State) != len(expected) {
		t.Fatalf("witness has %d nodes, expected %d (extra junk present)", len(witness.State), len(expected))
	}
}
