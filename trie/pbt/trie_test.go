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
