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

// genTrie interface is used by the snap syncer to generate merkle tree nodes
// based on a received batch of states.
type genTrie interface {
	// update inserts the state item into generator trie.
	update(key, value []byte) error

	// commit flushes the right boundary nodes if complete flag is true. This
	// function must be called before flushing the associated database batch.
	commit(complete bool) common.Hash
}

// pathTrie is a wrapper over the stackTrie, incorporating numerous additional
// logics to handle the semi-completed trie and potential leftover dangling
// nodes in the database. It is utilized for constructing the merkle tree nodes
// in path mode during the snap sync process.
type pathTrie struct {
	owner common.Hash     // identifier of trie owner, empty for account trie
	tr    *trie.StackTrie // underlying raw stack trie
	first []byte          // the path of first committed node by stackTrie
	last  []byte          // the path of last committed node by stackTrie

	// This flag indicates whether nodes on the left boundary are skipped for
	// committing. If set, the left boundary nodes are considered incomplete
	// due to potentially missing left children.
	skipLeftBoundary bool
	db               ethdb.KeyValueReader
	batch            ethdb.Batch
}

// newPathTrie initializes the path trie.
func newPathTrie(owner common.Hash, skipLeftBoundary bool, db ethdb.KeyValueReader, batch ethdb.Batch) *pathTrie {
	tr := &pathTrie{
		owner:            owner,
		skipLeftBoundary: skipLeftBoundary,
		db:               db,
		batch:            batch,
	}
	tr.tr = trie.NewStackTrie(tr.onTrieNode)
	return tr
}

// onTrieNode is invoked whenever a new node is committed by the stackTrie.
//
// As the committed nodes might be incomplete if they are on the boundaries
// (left or right), this function has the ability to detect the incomplete
// ones and filter them out for committing.
//
// Additionally, the assumption is made that there may exist leftover dangling
// nodes in the database. This function has the ability to detect the dangling
// nodes that fall within the path space of committed nodes (specifically on
// the path covered by internal extension nodes) and remove them from the
// database. This property ensures that the entire path space is uniquely
// occupied by committed nodes.
//
// Furthermore, all leftover dangling nodes along the path from committed nodes
// to the trie root (left and right boundaries) should be removed as well;
// otherwise, they might potentially disrupt the state healing process.
func (t *pathTrie) onTrieNode(path []byte, hash common.Hash, blob []byte) {
	// Filter out the nodes on the left boundary if skipLeftBoundary is
	// configured. Nodes are considered to be on the left boundary if
	// it's the first one to be committed, or the parent/ancestor of the
	// first committed node.
	if t.skipLeftBoundary && (t.first == nil || bytes.HasPrefix(t.first, path)) {
		if t.first == nil {
			// Memorize the path of first committed node, which is regarded
			// as left boundary. Deep-copy is necessary as the path given
			// is volatile.
			t.first = append([]byte{}, path...)

			// The left boundary can be uniquely determined by the first committed node
			// from stackTrie (e.g., N_1), as the shared path prefix between the first
			// two inserted state items is deterministic (the path of N_3). The path
			// from trie root towards the first committed node is considered the left
			// boundary. The potential leftover dangling nodes on left boundary should
			// be cleaned out.
			//
			//                            +-----+
			//                            | N_3 | shared path prefix of state_1 and state_2
			//                            +-----+
			//                            /-   -\
			//                       +-----+   +-----+
			// First committed node  | N_1 |   | N_2 | latest inserted node (contain state_2)
			//                       +-----+   +-----+
			//
			// The node with the path of the first committed one (e.g, N_1) is not
			// removed because it's a sibling of the nodes we want to commit, not
			// the parent or ancestor.
			for i := 0; i < len(path); i++ {
				t.delete(path[:i], false)
			}
		}
		return
	}
	// If boundary filtering is not configured, or the node is not on the left
	// boundary, commit it to database.
	//
	// Note: If the current committed node is an extension node, then the nodes
	// falling within the path between itself and its standalone (not embedded
	// in parent) child should be cleaned out for exclusively occupy the inner
	// path.
	//
	// This is essential in snap sync to avoid leaving dangling nodes within
	// this range covered by extension node which could potentially break the
	// state healing.
	//
	// The extension node is detected if its path is the prefix of last committed
	// one and path gap is larger than one. If the path gap is only one byte,
	// the current node could either be a full node, or a extension with single
	// byte key. In either case, no gaps will be left in the path.
	if t.last != nil && bytes.HasPrefix(t.last, path) && len(t.last)-len(path) > 1 {
		for i := len(path) + 1; i < len(t.last); i++ {
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

func (t *pathTrie) deleteAccountNode(path []byte, inner bool) {
	if inner {
		accountInnerLookupGauge.Inc(1)
	} else {
		accountOuterLookupGauge.Inc(1)
	}
	if !rawdb.ExistsAccountTrieNode(t.db, path) {
		return
	}
	if inner {
		accountInnerDeleteGauge.Inc(1)
	} else {
		accountOuterDeleteGauge.Inc(1)
	}
	rawdb.DeleteAccountTrieNode(t.batch, path)
}

func (t *pathTrie) deleteStorageNode(path []byte, inner bool) {
	if inner {
		storageInnerLookupGauge.Inc(1)
	} else {
		storageOuterLookupGauge.Inc(1)
	}
	if !rawdb.ExistsStorageTrieNode(t.db, t.owner, path) {
		return
	}
	if inner {
		storageInnerDeleteGauge.Inc(1)
	} else {
		storageOuterDeleteGauge.Inc(1)
	}
	rawdb.DeleteStorageTrieNode(t.batch, t.owner, path)
}

// delete commits the node deletion to provided database batch in path mode.
func (t *pathTrie) delete(path []byte, inner bool) {
	if t.owner == (common.Hash{}) {
		t.deleteAccountNode(path, inner)
	} else {
		t.deleteStorageNode(path, inner)
	}
}

// update implements genTrie interface, inserting a (key, value) pair into the
// stack trie.
func (t *pathTrie) update(key, value []byte) error {
	return t.tr.Update(key, value)
}

// commit implements genTrie interface, flushing the right boundary if it's
// considered as complete. Otherwise, the nodes on the right boundary are
// discarded and cleaned up.
//
// Note, this function must be called before flushing database batch, otherwise,
// dangling nodes might be left in database.
func (t *pathTrie) commit(complete bool) common.Hash {
	// If the right boundary is claimed as complete, flush them out.
	// The nodes on both left and right boundary will still be filtered
	// out if left boundary filtering is configured.
	if complete {
		// Commit all inserted but not yet committed nodes(on the right
		// boundary) in the stackTrie.
		hash := t.tr.Hash()
		if t.skipLeftBoundary {
			return common.Hash{} // hash is meaningless if left side is incomplete
		}
		return hash
	}
	// Discard nodes on the right boundary as it's claimed as incomplete. These
	// nodes might be incomplete due to missing children on the right side.
	// Furthermore, the potential leftover nodes on right boundary should also
	// be cleaned out.
	//
	// The right boundary can be uniquely determined by the last committed node
	// from stackTrie (e.g., N_1), as the shared path prefix between the last
	// two inserted state items is deterministic (the path of N_3). The path
	// from trie root towards the last committed node is considered the right
	// boundary (root to N_3).
	//
	//                           +-----+
	//                           | N_3 | shared path prefix of last two states
	//                           +-----+
	//                           /-   -\
	//                      +-----+   +-----+
	// Last committed node  | N_1 |   | N_2 | latest inserted node  (contain last state)
	//                      +-----+   +-----+
	//
	// Another interesting scenario occurs when the trie is committed due to
	// too many items being accumulated in the batch. To flush them out to
	// the database, the path of the last inserted node (N_2) is temporarily
	// treated as an incomplete right boundary, and nodes on this path are
	// removed (e.g. from root to N_3).
	// However, this path will be reclaimed as an internal path by inserting
	// more items after the batch flush. New nodes on this path can be committed
	// with no issues as they are actually complete. Also, from a database
	// perspective, first deleting and then rewriting is a valid data update.
	for i := 0; i < len(t.last); i++ {
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
