// Copyright 2019 The go-ethereum Authors
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

package trie

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/stretchr/testify/assert"
)

// Tests that the trie database returns a missing trie node error if attempting
// to retrieve the meta root.
func TestDatabaseMetarootFetch(t *testing.T) {
	db := NewDatabase(memorydb.New())
	if _, err := db.Node(common.Hash{}); err == nil {
		t.Fatalf("metaroot retrieval succeeded")
	}
}

func TestMissingNodeAfterPruningIntermediateNode(t *testing.T) {
	// Pruning uses a bloom filter to remove old trie nodes from the database, which
	// creates some probability of an unprotected trie node being left in a state where
	// some of its descendants have been pruned from the disk. This test ensures that this
	// scenario is handled correctly by the trie database, such that if a block is processed
	// such that the intermediate node with missing children is revived, the removed descendant
	// nodes are re-written to disk correctly.
	assert := assert.New(t)
	memdb := memorydb.New()

	// Create a trieDB and build the following tries:
	// trie1:
	//       R
	//      / \
	//     A    B
	//   /  \    \
	//  1    2    3
	//
	// trie2:
	//       R'
	//     /   \
	//    A'    B
	//     \     \
	//      2     3
	//
	// trie3:
	//       R
	//      / \
	//     A    B
	//   /  \    \
	//  1    2    3

	// Construct [db1] and [trie1]
	db1 := NewDatabase(memdb)
	trie1, err := New(common.Hash{}, db1)
	assert.NoError(err)

	// Construct trie1
	defaultVal := []byte("value")
	k1 := common.BytesToHash([]byte("ra1")).Bytes()
	k2 := common.BytesToHash([]byte("ra2")).Bytes()
	k3 := common.BytesToHash([]byte("rb3")).Bytes()

	assert.NoError(trie1.TryUpdate(k1, defaultVal))
	assert.NoError(trie1.TryUpdate(k2, defaultVal))
	assert.NoError(trie1.TryUpdate(k3, defaultVal))

	root1, _, err := trie1.Commit(nil)
	assert.NoError(err)
	assert.NoError(db1.Commit(root1, true, nil))

	// Construct trie2 by deleting [k1] from the trie
	trie2, err := New(root1, db1)
	assert.NoError(err)

	assert.NoError(trie2.TryDelete(k1))

	root2, _, err := trie2.Commit(nil)
	assert.NoError(err)
	assert.NoError(db1.Commit(root2, true, nil))

	// Confirm that the leaf key for RA1 has been written to disk as
	// expected and delete its parent node to cause a dangling node error.
	nodeIterator := trie1.NodeIterator(nil)
	foundLeaf := false

	for nodeIterator.Next(true) {
		if !nodeIterator.Leaf() {
			continue
		}

		leafKey := common.CopyBytes(nodeIterator.LeafKey())
		if bytes.Equal(leafKey, k1) {
			parentHash := nodeIterator.Parent()
			assert.NoError(db1.DiskDB().Delete(parentHash[:]))
			foundLeaf = true
		}
	}

	// Check that the iterator did not error
	assert.NoError(nodeIterator.Error())
	assert.True(foundLeaf, "failed to find leaf to be deleted")

	// Confirm that we can no longer construct trie1 now has a missing node.
	trie1, err = New(root1, db1)
	assert.NoError(err)

	_, err = trie1.TryGet(k1)
	assert.Error(err)

	// Construct the new database on top of the same [memdb] after the path
	// to the leaf node has been forcibly deleted.
	db2 := NewDatabase(memdb)

	// Construct trie2 and add back k1, such that we revert to the prior state
	// which previously had a missing trie node.
	trie3, err := New(root2, db2)
	assert.NoError(err)

	assert.NoError(trie3.TryUpdate(k1, defaultVal))
	root3, _, err := trie3.Commit(nil)
	assert.NoError(err)
	assert.Equal(root1, root3, "roots should be identical after adding k1 back to trie2")
	assert.NoError(db2.Commit(root3, true, nil))

	_, err = trie3.TryGet(k1)
	assert.NoError(err)
}
