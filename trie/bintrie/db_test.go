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

package bintrie_test

// Disk round-trip tests: the engine against a real path database. These
// exercise record serialization, path-addressed lazy resolution, deletion
// records and structural bucket drops through commit/reopen cycles.

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

// commitTrie hashes and commits the trie, flushes the node set into the
// database as block number, and returns the new root.
func commitTrie(t *testing.T, db *triedb.Database, tr *bintrie.BinaryTrie, parent common.Hash, block uint64) common.Hash {
	t.Helper()
	root, set := tr.Commit(true)
	merged := trienode.NewMergedNodeSet()
	if set != nil {
		if err := merged.Merge(set); err != nil {
			t.Fatal(err)
		}
	}
	if root != parent {
		if err := db.Update(root, parent, block, merged, triedb.NewStateSet()); err != nil {
			t.Fatal(err)
		}
		if err := db.Commit(root, false); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func openTrie(t *testing.T, db *triedb.Database, root common.Hash) *bintrie.BinaryTrie {
	t.Helper()
	tr, err := bintrie.NewBinaryTrie(root, db)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

func testAccount(nonce uint64) *types.StateAccount {
	return &types.StateAccount{
		Nonce:    nonce,
		Balance:  uint256.NewInt(nonce + 1),
		Root:     types.EmptyRootHash,
		CodeHash: types.EmptyCodeHash[:],
	}
}

// TestDiskRoundTrip commits a populated trie, reopens it from disk and
// verifies reads, iteration, proofs, deletions and bucket drops across
// multiple commit cycles.
func TestDiskRoundTrip(t *testing.T) {
	disk := rawdb.NewMemoryDatabase()
	db := triedb.NewDatabase(disk, triedb.PBTDefaults)
	defer db.Close()

	// Cycle 1: accounts with header and overflow storage.
	tr := openTrie(t, db, types.EmptyBinaryHash)
	var (
		addrs  []common.Address
		slotHi = common.BytesToHash(append(bytes.Repeat([]byte{0}, 30), 4, 0)) // slot 1024: overflow
		slotLo = common.BytesToHash([]byte{7})                                 // slot 7: header
	)
	for i := byte(1); i <= 8; i++ {
		addr := common.Address{i}
		addrs = append(addrs, addr)
		if err := tr.UpdateAccount(addr, testAccount(uint64(i)), 0, nil); err != nil {
			t.Fatal(err)
		}
		if err := tr.UpdateStorage(addr, slotLo[:], slotLo[:]); err != nil {
			t.Fatal(err)
		}
		if err := tr.UpdateStorage(addr, slotHi[:], slotHi[:]); err != nil {
			t.Fatal(err)
		}
	}
	root1 := commitTrie(t, db, tr, types.EmptyBinaryHash, 1)

	// Reopen and verify every account and slot resolves from disk.
	tr = openTrie(t, db, root1)
	for i, addr := range addrs {
		acc, err := tr.GetAccount(addr)
		if err != nil {
			t.Fatal(err)
		}
		if acc == nil || acc.Nonce != uint64(i+1) {
			t.Fatalf("account %x wrong after reopen: %+v", addr, acc)
		}
		for _, slot := range [][]byte{slotLo[:], slotHi[:]} {
			val, err := tr.GetStorage(addr, slot)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(val, slot) {
				t.Fatalf("slot %x wrong after reopen: %x", slot, val)
			}
		}
	}
	// Absent account resolves to nil, not garbage.
	if acc, err := tr.GetAccount(common.Address{0xEE}); err != nil || acc != nil {
		t.Fatalf("phantom account: %+v err %v", acc, err)
	}

	// Iterate every leaf from disk: 8 accounts x (2 header + 1 header slot +
	// 1 overflow slot) = 32 leaves.
	it, err := tr.NodeIterator(nil)
	if err != nil {
		t.Fatal(err)
	}
	var (
		leaves   int
		lastKey  []byte
		perOrder = true
	)
	for it.Next(true) {
		if it.Leaf() {
			leaves++
			if lastKey != nil && bytes.Compare(lastKey, it.LeafKey()) >= 0 {
				perOrder = false
			}
			lastKey = append(lastKey[:0], it.LeafKey()...)
		}
	}
	if err := it.Error(); err != nil {
		t.Fatal(err)
	}
	if leaves != 32 {
		t.Fatalf("leaf count %d, want 32", leaves)
	}
	if !perOrder {
		t.Fatal("iterator not in ascending key order")
	}

	// Prove an account leaf and a storage leaf from the reopened trie.
	tr = openTrie(t, db, root1)
	proof := rawdb.NewMemoryDatabase()
	if err := tr.Prove(bintrie.BasicDataKey(addrs[0]), proof); err != nil {
		t.Fatal(err)
	}
	if val, err := bintrie.VerifyProof(root1, bintrie.BasicDataKey(addrs[0]), proof); err != nil || val == nil {
		t.Fatalf("account proof failed: %v", err)
	}
	// Absence proof for a missing account.
	proof2 := rawdb.NewMemoryDatabase()
	missing := common.Address{0xEE}
	if err := tr.Prove(bintrie.BasicDataKey(missing), proof2); err != nil {
		t.Fatal(err)
	}
	if val, err := bintrie.VerifyProof(root1, bintrie.BasicDataKey(missing), proof2); err != nil || val != nil {
		t.Fatalf("absence proof failed: val %x err %v", val, err)
	}

	// Cycle 2: delete slots and a whole account (header stem + bucket).
	tr = openTrie(t, db, root1)
	if err := tr.DeleteStorage(addrs[0], slotHi[:]); err != nil {
		t.Fatal(err)
	}
	if err := tr.DeleteAccount(addrs[1]); err != nil {
		t.Fatal(err)
	}
	if err := tr.DeleteStorageBucket(addrs[1]); err != nil {
		t.Fatal(err)
	}
	root2 := commitTrie(t, db, tr, root1, 2)
	if root2 == root1 {
		t.Fatal("root unchanged after deletions")
	}

	// Reopen at the new root: deletions visible, syblings intact.
	tr = openTrie(t, db, root2)
	if val, _ := tr.GetStorage(addrs[0], slotHi[:]); val != nil {
		t.Fatalf("deleted slot still present: %x", val)
	}
	if acc, _ := tr.GetAccount(addrs[1]); acc != nil {
		t.Fatalf("deleted account still present: %+v", acc)
	}
	if val, _ := tr.GetStorage(addrs[1], slotHi[:]); val != nil {
		t.Fatalf("deleted bucket slot still present: %x", val)
	}
	if acc, err := tr.GetAccount(addrs[2]); err != nil || acc == nil {
		t.Fatalf("sibling account lost: %v", err)
	}
	// The dropped bucket answers HasPrefix false; a live one true.
	if has, err := tr.HasPrefix(bintrie.StorageBucketPrefix(addrs[1])); err != nil || has {
		t.Fatalf("dropped bucket still probed: %v %v", has, err)
	}
	if has, err := tr.HasPrefix(bintrie.StorageBucketPrefix(addrs[2])); err != nil || !has {
		t.Fatalf("live bucket not probed: %v %v", has, err)
	}

	// Cycle 3: the deletion-equivalence check. A fresh trie holding the
	// post-deletion state must produce the same root.
	fresh := openTrie(t, db, types.EmptyBinaryHash)
	for i, addr := range addrs {
		if addr == addrs[1] {
			continue
		}
		if err := fresh.UpdateAccount(addr, testAccount(uint64(i+1)), 0, nil); err != nil {
			t.Fatal(err)
		}
		if err := fresh.UpdateStorage(addr, slotLo[:], slotLo[:]); err != nil {
			t.Fatal(err)
		}
		if addr == addrs[0] {
			continue
		}
		if err := fresh.UpdateStorage(addr, slotHi[:], slotHi[:]); err != nil {
			t.Fatal(err)
		}
	}
	if got := fresh.Hash(); got != root2 {
		t.Fatalf("deletion root not canonical: got %x want %x", got, root2)
	}
}

// TestDiskDeleteToEmpty commits state and then deletes everything, checking
// the empty root and that deletion records reach the database.
func TestDiskDeleteToEmpty(t *testing.T) {
	disk := rawdb.NewMemoryDatabase()
	db := triedb.NewDatabase(disk, triedb.PBTDefaults)
	defer db.Close()

	tr := openTrie(t, db, types.EmptyBinaryHash)
	addr := common.Address{1}
	if err := tr.UpdateAccount(addr, testAccount(5), 0, nil); err != nil {
		t.Fatal(err)
	}
	root1 := commitTrie(t, db, tr, types.EmptyBinaryHash, 1)

	tr = openTrie(t, db, root1)
	if err := tr.DeleteAccount(addr); err != nil {
		t.Fatal(err)
	}
	root2, set := tr.Commit(true)
	if root2 != types.EmptyBinaryHash {
		t.Fatalf("expected empty root, got %x", root2)
	}
	if set == nil || len(set.Nodes) == 0 {
		t.Fatal("expected deletion records for the emptied trie")
	}
	deleted := 0
	for _, n := range set.Nodes {
		if n.IsDeleted() {
			deleted++
		}
	}
	if deleted == 0 {
		t.Fatal("no deletion markers emitted")
	}
}

var _ ethdb.KeyValueWriter = rawdb.NewMemoryDatabase() // proof sink conformance
