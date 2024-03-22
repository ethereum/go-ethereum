// Copyright 2024 The go-ethereum Authors
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

package snap

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
)

// genTrie interface is used by the trie to generate merkle tree nodes based
// on a received batch of states.
type genTrie interface {
	// update inserts the state into generator trie.
	update(key, value []byte) error

	// commit flushes the leftover nodes produced in the trie into database.
	// The nodes on right boundary won't be committed unless this function
	// is called. The flag complete should be set to true if there are more
	// items on the right side.
	//
	// This function must be called before flushing database batch.
	commit(complete bool) common.Hash
}

// pathTrie is a wrapper over the stackTrie, incorporating numerous additional
// logics to handle the semi-completed trie and potential leftover dangling
// nodes in the database. It is utilized for constructing the merkle tree nodes
// in path mode during the snap sync process.
type pathTrie struct {
	owner common.Hash     // identifier of trie owner, empty for account trie
	tr    *trie.StackTrie // underlying raw stack trie
	first []byte          // the path of first written node
	last  []byte          // the path of last written node

	// Flag whether the nodes on the left boundary are skipped for committing.
	// If it's set, then nodes on the left boundary are regarded as incomplete
	// due to potentially missing left children.
	noLeftBound bool
	db          ethdb.KeyValueReader
	batch       ethdb.Batch
}

// newPathTrie initializes the path trie.
func newPathTrie(owner common.Hash, noLeftBound bool, db ethdb.KeyValueReader, batch ethdb.Batch) *pathTrie {
	tr := &pathTrie{
		owner:       owner,
		noLeftBound: noLeftBound,
		db:          db,
		batch:       batch,
	}
	tr.tr = trie.NewStackTrie(tr.onTrieNode)
	return tr
}

// onTrieNode is invoked whenever a new node is produced by the stackTrie.
//
// As the produced nodes might be incomplete if they are on the boundaries
// (left or right), this function has the ability to detect the incomplete
// ones and filter them out for committing. Namely, only the nodes belonging
// to completed subtries will be committed.
//
// Additionally, the assumption is made that there may exist leftover dangling
// nodes in the database. This function has the ability to detect all the
// dangling nodes that fall within the committed subtries (on the path covered
// by internal extension nodes) and remove them from the database. This property
// ensures that the entire path space is uniquely occupied by committed subtries.
//
// Furthermore, all leftover dangling nodes along the path from committed tries
// to the root node should be removed as well; otherwise, they might potentially
// disrupt the state healing process, leaving behind an inconsistent state.
func (t *pathTrie) onTrieNode(path []byte, hash common.Hash, blob []byte) {
	// Filter out the nodes on the left boundary if noLeftBound is configured.
	// Nodes are considered to be on the left boundary if it's the first one
	// produced, or on the path of the first produced one.
	if t.noLeftBound && (t.first == nil || bytes.HasPrefix(t.first, path)) {
		if t.first == nil {
			// Memorize the path of first produced node, which is regarded
			// as left boundary. Deep-copy is necessary as the path given
			// is volatile.
			t.first = append([]byte{}, path...)

			// The position of first complete sub trie (e.g. N_3) can be determined
			// by the first produced node(e.g. N_1) correctly, with a branch node
			// (e.g. N_2) as the common parent for shared path prefix. Therefore,
			// the nodes along the path from root to N_1 can be regarded as left
			// boundary. The leftover dangling nodes on left boundary should be
			// cleaned out first before committing any node.
			//
			//                           +-----+
			//                           | N_2 |  parent for shared path prefix
			//                           +-----+
			//                           /-  -\
			//                     +-----+   +-----+
			// First produced one  | N_1 |   | N_3 | First completed sub trie
			//                     +-----+   +-----+
			//
			// Nodes must be cleaned from top to bottom as it's possible the procedure
			// is interrupted in the middle.
			//
			// The node with the path of the first produced node is not removed, as
			// it's a sibling of the first complete sub-trie, not the parent. There
			// is no reason to remove it.
			for i := 0; i < len(path); i++ {
				t.delete(path[:i], false)
			}
		}
		return
	}
	// If boundary filtering is not configured, or the node is not on the left
	// boundary, commit it to database.
	//
	// Note, the nodes fall within the path between extension node and its
	// **in disk** child must be cleaned out before committing the extension
	// node. This is essential in snap sync to avoid leaving dangling nodes
	// within this range covered by extension node which could potentially
	// break the state healing.
	//
	// The target node is detected if its path is the prefix of last written
	// one and path gap is non-zero.
	//
	// Nodes must be cleaned from top to bottom, including the node with the
	// path of the committed extension node itself.
	if t.last != nil && bytes.HasPrefix(t.last, path) && len(t.last)-len(path) > 1 {
		for i := len(path); i < len(t.last); i++ {
			t.delete(t.last[:i], true)
		}
	}
	t.write(path, blob)

	// Update the last flag. Deep-copy is necessary as the provided path is volatile.
	if t.last == nil {
		t.last = append([]byte{}, path...)
	} else {
		t.last = append(t.last[:0], path...)
	}
}

// write commits the node write to provided database batch in path mode.
func (t *pathTrie) write(path []byte, blob []byte) {
	if t.owner == (common.Hash{}) {
		rawdb.WriteAccountTrieNode(t.batch, path, blob)
	} else {
		rawdb.WriteStorageTrieNode(t.batch, t.owner, path, blob)
	}
}

// delete commits the node deletion to provided database batch in path mode.
func (t *pathTrie) delete(path []byte, inner bool) {
	if t.owner == (common.Hash{}) {
		if rawdb.ExistsAccountTrieNode(t.db, path) {
			rawdb.DeleteAccountTrieNode(t.batch, path)
			if inner {
				accountInnerDeleteGauge.Inc(1)
			} else {
				accountOuterDeleteGauge.Inc(1)
			}
		}
		if inner {
			accountInnerLookupGauge.Inc(1)
		} else {
			accountOuterLookupGauge.Inc(1)
		}
		return
	}
	if rawdb.ExistsStorageTrieNode(t.db, t.owner, path) {
		rawdb.DeleteStorageTrieNode(t.batch, t.owner, path)
		if inner {
			storageInnerDeleteGauge.Inc(1)
		} else {
			storageOuterDeleteGauge.Inc(1)
		}
	}
	if inner {
		storageInnerLookupGauge.Inc(1)
	} else {
		storageOuterLookupGauge.Inc(1)
	}
}

// update implements genTrie interface, inserting a (key, value) pair into the
// stack trie.
func (t *pathTrie) update(key, value []byte) error {
	return t.tr.Update(key, value)
}

// commit implements genTrie interface, flushing the right boundary if it's
// regarded as complete. Otherwise, the nodes on the right boundary are discarded
// and cleaned up.
//
// Note, this function must be called before flushing database batch, otherwise,
// dangling nodes might be left in database.
func (t *pathTrie) commit(complete bool) common.Hash {
	// If the right boundary is claimed as complete, flush them out.
	// The nodes on both left and right boundary will still be filtered
	// out if left boundary filtering is configured.
	if complete {
		return t.tr.Hash()
	}
	// If the right boundary is claimed as incomplete, the uncommitted
	// nodes should be discarded, as they might be incomplete due to
	// missing children on the right side. Furthermore, previously committed
	// nodes can be the children of the right boundary nodes; therefore,
	// the nodes of the right boundary must be cleaned out!
	//
	// The position of the last complete sub-trie (e.g., N_1) can be correctly
	// determined by the last produced node (e.g., N_3), with a branch node
	// (e.g., N_2) as the common parent for the shared path prefix. Therefore,
	// the nodes along the path from the root to N_3 can be regarded as the
	// right boundary.
	//
	//                             +-----+
	//                             | N_2 |  parent for shared path prefix
	//                             +-----+
	//                             /-  -\
	//                        +-----+   +-----+
	// Last complete subtrie  | N_1 |   | N_3 | Last produced node
	//                        +-----+   +-----+
	//
	// Another interesting scenario occurs when the trie is committed due to
	// too many items being accumulated in the batch. To flush them out to
	// the database, the path of the last inserted item is temporarily treated
	// as an incomplete right boundary, and nodes on this path are removed.
	//
	// However, this path will be reclaimed as an internal path by inserting
	// more items after the batch flush. Newly produced nodes on this path
	// can be committed with no issues as they are actually complete (also
	// from a database perspective, first deleting and then rewriting is
	// still a valid data update).
	//
	// Nodes must be cleaned from top to bottom as it's possible the procedure
	// is interrupted in the middle.
	for i := 0; i < len(t.last); i++ {
		// The node with the path of the last produced node is not removed, as
		// it's a sibling of the last complete sub-trie, not the parent. There
		// is no reason to remove it.
		t.delete(t.last[:i], false)
	}
	return common.Hash{} // the hash is meaningless for incomplete commit
}

// hashTrie is a wrapper over the stackTrie for implementing genTrie interface.
type hashTrie struct {
	tr *trie.StackTrie
}

// newHashTrie initializes the hash trie.
func newHashTrie(batch ethdb.Batch) *hashTrie {
	return &hashTrie{tr: trie.NewStackTrie(func(path []byte, hash common.Hash, blob []byte) {
		rawdb.WriteLegacyTrieNode(batch, hash, blob)
	})}
}

// update implements genTrie interface, inserting a (key, value) pair into
// the stack trie.
func (t *hashTrie) update(key, value []byte) error {
	return t.tr.Update(key, value)
}

// commit implements genTrie interface, committing the nodes on right boundary.
func (t *hashTrie) commit(complete bool) common.Hash {
	if !complete {
		return common.Hash{} // the hash is meaningless for incomplete commit
	}
	return t.tr.Hash() // return hash only if it's claimed as complete
}
