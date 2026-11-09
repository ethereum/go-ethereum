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

package pbt

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb/database"
)

// memNodeDB is a node database holding the latest record at each path.
type memNodeDB struct{ nodes map[string][]byte }

func newMemNodeDB() *memNodeDB { return &memNodeDB{nodes: make(map[string][]byte)} }

func (db *memNodeDB) NodeReader(common.Hash) (database.NodeReader, error) { return db, nil }

func (db *memNodeDB) Node(_ common.Hash, path []byte, _ common.Hash) ([]byte, error) {
	blob, ok := db.nodes[string(path)]
	if !ok {
		return nil, fmt.Errorf("missing node at path %x", path)
	}
	return blob, nil
}

// commit commits t and applies its node set, returning the new root.
func (db *memNodeDB) commit(t *PartitionedTrie) common.Hash {
	root, set := t.Commit(false)
	if set != nil {
		set.ForEachWithOrder(func(path string, n *trienode.Node) {
			if n.IsDeleted() {
				delete(db.nodes, path)
			} else {
				db.nodes[path] = common.CopyBytes(n.Blob)
			}
		})
	}
	return root
}

// TestCommitRemovesCollapsedRecords empties two sibling groups in one trie
// lifetime. Removing the first one collapses their branch and moves the other
// group to the branch's path; removing that one must delete the branch's old
// record from the database instead of leaving it behind.
func TestCommitRemovesCollapsedRecords(t *testing.T) {
	var (
		other = common.HexToAddress("0xaaaa")
		x     = common.HexToAddress("0x1234")
		slot0 = common.Hash{}.Bytes()                     // other's header stem, account zone
		slotA = common.BigToHash(big.NewInt(64)).Bytes()  // x's bucket, group of tree index 0
		slotB = common.BigToHash(big.NewInt(300)).Bytes() // x's bucket, group of tree index 1
		value = common.Hash{31: 1}.Bytes()
	)

	// The account zone holds other's header, the storage zone x's bucket,
	// a branch whose children are its two groups.
	db := newMemNodeDB()
	tr, err := New(types.EmptyBinaryHash, db)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []struct {
		addr common.Address
		slot []byte
	}{{other, slot0}, {x, slotA}, {x, slotB}} {
		if err := tr.UpdateStorage(w.addr, w.slot, value); err != nil {
			t.Fatal(err)
		}
	}
	root := db.commit(tr)

	// Empty both groups of the bucket in one trie lifetime.
	if tr, err = New(root, db); err != nil {
		t.Fatal(err)
	}
	for _, slot := range [][]byte{slotA, slotB} {
		if err := tr.DeleteStorage(x, slot); err != nil {
			t.Fatal(err)
		}
	}
	root = db.commit(tr)

	// The database must hold exactly the records of the same state built from
	// scratch.
	want := newMemNodeDB()
	ref, _ := New(types.EmptyBinaryHash, want)
	if err := ref.UpdateStorage(other, slot0, value); err != nil {
		t.Fatal(err)
	}
	if wantRoot := want.commit(ref); root != wantRoot {
		t.Fatalf("root mismatch: have %x, want %x", root, wantRoot)
	}
	for path, blob := range db.nodes {
		if wb, ok := want.nodes[path]; !ok {
			t.Errorf("leftover record at path %x", path)
		} else if !bytes.Equal(blob, wb) {
			t.Errorf("record at path %x differs", path)
		}
	}
	for path := range want.nodes {
		if _, ok := db.nodes[path]; !ok {
			t.Errorf("missing record at path %x", path)
		}
	}
}

// largeGroupSlots are 20 header slots of one account: a single group large
// enough for its record to carry the top branch's child hashes.
func largeGroupSlots() map[string][]byte {
	slots := make(map[string][]byte)
	for i := range 20 {
		slots[string(common.BigToHash(big.NewInt(int64(i))).Bytes())] = common.Hash{30: 1, 31: byte(i + 1)}.Bytes()
	}
	return slots
}

// TestGroupRecordHashCommitsToValues changes a value inside the record of a
// large group. The hash the database computes for the record must change
// with it: stored intermediate hashes must not stand in for the values.
func TestGroupRecordHashCommitsToValues(t *testing.T) {
	x := common.HexToAddress("0x1234")

	// x's header group is the only stem, so its record is the root record,
	// which the path database hashes to find its persisted state root.
	db := newMemNodeDB()
	tr, _ := New(types.EmptyBinaryHash, db)
	for slot, value := range largeGroupSlots() {
		if err := tr.UpdateStorage(x, []byte(slot), value); err != nil {
			t.Fatal(err)
		}
	}
	root := db.commit(tr)

	blob := db.nodes[""]
	if len(blob) == 0 || blob[0] != tagGroup {
		t.Fatalf("root record is not a group record")
	}
	if h, err := DeserializeAndHash(blob); err != nil || h != root {
		t.Fatalf("root record hashes to %x (err %v), want root %x", h, err, root)
	}
	// Overwrite the first value; the bitmap and every length stay valid.
	modified := common.CopyBytes(blob)
	modified[4+int(blob[3])+32] ^= 0xff

	h, err := DeserializeAndHash(modified)
	if err == nil && h == root {
		t.Fatalf("record with a modified value still hashes to the committed root %x", root)
	}
}

// TestLoadedGroupRehashesAfterWrites loads a large group, whose record
// carries the top branch's child hashes, then modifies and moves it. Every
// commit must match the same state built from scratch, record for record.
func TestLoadedGroupRehashesAfterWrites(t *testing.T) {
	var (
		x     = common.HexToAddress("0x1234")
		other = common.HexToAddress("0xaaaa")
		slots = largeGroupSlots()
		slot0 = common.Hash{}.Bytes()
	)
	db := newMemNodeDB()
	tr, _ := New(types.EmptyBinaryHash, db)
	for slot, value := range slots {
		if err := tr.UpdateStorage(x, []byte(slot), value); err != nil {
			t.Fatal(err)
		}
	}
	if err := tr.UpdateStorage(other, slot0, common.Hash{31: 0xee}.Bytes()); err != nil {
		t.Fatal(err)
	}
	root := db.commit(tr)

	// expect compares the database with the given state built from scratch.
	expect := func(step string, root common.Hash, withOther bool) {
		t.Helper()
		want := newMemNodeDB()
		ref, _ := New(types.EmptyBinaryHash, want)
		for slot, value := range slots {
			ref.UpdateStorage(x, []byte(slot), value)
		}
		if withOther {
			ref.UpdateStorage(other, slot0, common.Hash{31: 0xee}.Bytes())
		}
		if wantRoot := want.commit(ref); root != wantRoot {
			t.Fatalf("%s: root %x, want %x", step, root, wantRoot)
		}
		for path, blob := range want.nodes {
			if !bytes.Equal(db.nodes[path], blob) {
				t.Fatalf("%s: record at path %x differs from a fresh build", step, path)
			}
		}
		if len(db.nodes) != len(want.nodes) {
			t.Fatalf("%s: %d records, want %d", step, len(db.nodes), len(want.nodes))
		}
	}

	// Modify a value of the loaded group.
	tr, _ = New(root, db)
	first := string(common.BigToHash(big.NewInt(0)).Bytes())
	slots[first] = common.Hash{31: 0x77}.Bytes()
	if err := tr.UpdateStorage(x, []byte(first), slots[first]); err != nil {
		t.Fatal(err)
	}

	root = db.commit(tr)
	expect("modify", root, true)

	// Move the loaded, unmodified group to the root by deleting its sibling.
	tr, _ = New(root, db)
	if err := tr.DeleteStorage(other, slot0); err != nil {
		t.Fatal(err)
	}
	root = db.commit(tr)
	expect("move", root, false)

	// Modify the group again where it now sits.
	tr, _ = New(root, db)
	slots[first] = common.Hash{31: 0x78}.Bytes()
	if err := tr.UpdateStorage(x, []byte(first), slots[first]); err != nil {
		t.Fatal(err)
	}
	root = db.commit(tr)
	expect("modify after move", root, false)
}
