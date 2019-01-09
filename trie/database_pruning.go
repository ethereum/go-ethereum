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
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// pruner is responsible for pruning the state trie based on liveness checks
// whenever the in-memory garbage collector attempt to dereference a node from
// disk.
type pruner struct {
	db    *Database    // Trie database for accessing dirty and clean data
	tries []*traverser // Individual stateful trie traversers for fast liveness checks

	marks []*prunerTarget // Nodes marked for potential pruning
	batch ethdb.Batch     // Write batch to minimize database trashing
}

// prunerTarget represents a single marked target for potential pruning.
type prunerTarget struct {
	owner common.Hash // Owner account hash of the node to delete
	path  []byte      // Patricia path leading to this node
	hash  common.Hash // Hash of the node to delete
}

// newPruner creates a new trie pruner tied to the liveness of all the currently
// referenced in-memory nodes.
func (db *Database) newPruner() *pruner {
	return &pruner{
		db:    db,
		batch: db.diskdb.NewBatch(),
	}
}

// mark adds a new prune target to be deleted on the pruning run.
func (p *pruner) mark(owner common.Hash, hash common.Hash, path []byte) {
	p.marks = append(p.marks, &prunerTarget{
		owner: owner,
		hash:  hash,
		path:  common.CopyBytes(path),
	})
}

// execute runs the pruning procedure, deleting everything that has no live
// reference any more.
func (p *pruner) execute() {
	// Create the set of traversers based on the live tries
	for key := range p.db.dirties[metaRoot].children {
		_, root := splitNodeKey(key)
		p.tries = append(p.tries, &traverser{
			db: p.db,
			state: &traverserState{
				node: hashNode(root[:]),
				hash: root,
			},
		})
	}
	// Iterate over all the nodes marked for pruning and delete them
	for _, mark := range p.marks {
		p.prune(mark.owner, mark.hash, mark.path)
	}
}

// flush commits any pending database writes. It does not reset the batch since
// we only ever supposed to commit once per prune run.
func (p *pruner) flush() error {
	return p.batch.Write()
}

// prune deletes a trie node from disk if there are no more live references to
// it, cascading until all dangling nodes are removed.
func (p *pruner) prune(owner common.Hash, hash common.Hash, path []byte) {
	// If the node is still live in the memory cache, it's still referenced so we
	// can abort. This case is important when and old trie being pruned references
	// a new node (maybe that node was recreted since), since currently live nodes
	// are stored expanded, not as hashes.
	key := makeNodeKey(owner, hash)
	if p.db.dirties[key] != nil {
		return
	}
	// Iterate over all the live tries and check node liveliness
	crosspath := path
	if owner != (common.Hash{}) {
		crosspath = append(append(keybytesToHex(owner[:]), 0xff), crosspath...)
	}
	unrefs := make(map[common.Hash]bool)
	for _, trie := range p.tries {
		// If the node is still live, abort
		if trie.live(owner, hash, crosspath, unrefs) {
			return
		}
		// Node dead in this trie, cache the result for subsequent traversals
		trie.unref(2, unrefs)
	}
	// Dead node found, delete it from the database
	dead := []byte(makeNodeKey(owner, hash))
	blob, err := p.db.diskdb.Get(dead)
	if blob == nil || err != nil {
		log.Error("Missing prune target", "owner", owner, "hash", hash, "path", fmt.Sprintf("%x", path))
		return
	}
	node := mustDecodeNode(hash[:], blob, 0)

	// Prune the node and its children if it's not a bytecode blob
	p.db.cleans.Delete(key)
	p.batch.Delete(dead)
	p.db.prunenodes++
	p.db.prunesize += common.StorageSize(len(blob))

	iterateRefs(node, path, func(path []byte, hash common.Hash) error {
		p.prune(owner, hash, path)
		return nil
	})
}

// traverser is a stateful trie traversal data structure used by the pruner to
// verify the liveness of a node within a specific trie. The reason for having
// a separate data structure is to allow reusing previous traversals to check
// the liveness of nested nodes (i.e. entire subtried during pruning).
type traverser struct {
	db    *Database       // Trie database for accessing dirty and clean data
	state *traverserState // Leftover state from the previous traversals
}

// traverserState is the internal state of a trie traverser.
type traverserState struct {
	parent *traverserState // Parent traverser to allow backtracking
	prefix []byte          // Path leading up to the root of this traverser
	node   node            // Trie node where this traverser is currently at
	hash   common.Hash     // Hash of the trie node at the traversed position
}

// live checks whether the trie iterated by this traverser contains the hashnode
// at the given path, minimizing data access and processing by reusing previous
// state instead of starting fresh.
//
// The path is a full canonical path from the account trie root down to the node
// potentially crossing over into a storage trie. The account and storage trie
// paths are separated by a 0xff byte (nibbles range from 0x00-0x10). This byte
// is needed to differentiate between the leaf of the account trie and the root
// of a storage trie (which otherwise would have the same traversal path).
func (t *traverser) live(owner common.Hash, hash common.Hash, path []byte, unrefs map[common.Hash]bool) bool {
	// Rewind the traverser until it's prefix is actually a prefix of the path
	for !bytes.HasPrefix(path, t.state.prefix) {
		t.state = t.state.parent
	}
	// Short circuit the liveness check if we already covered this prefix (if this
	// prefix path was not yet seen in previous tries, no parent could have been
	// seen either, so no point in checkin upwards further than the first hash).
	state := t.state
	for state != nil {
		// If we've found a hash node, check if it's an already known result
		if state.hash != (common.Hash{}) {
			if unrefs[state.hash] {
				return false
			}
			break
		}
		// Not a hash node, traverse further up
		state = state.parent
	}
	// Traverse downward until the prefix matches the path completely
	path = path[len(t.state.prefix):]
	for len(path) > 0 {
		// If we're at a hash node, expand before continuing
		if n, ok := t.state.node.(hashNode); ok {
			// Short circuit if we already encountered this node
			t.state.hash = common.BytesToHash(n)
			if unrefs[t.state.hash] {
				return false
			}
			// Generate the database key for this hash node
			var key string
			if len(t.state.prefix) < 2*common.HashLength {
				key = makeNodeKey(common.Hash{}, t.state.hash)
			} else {
				key = makeNodeKey(owner, t.state.hash)
			}
			// Replace the node in the traverser with the expanded one
			if enc, err := t.db.cleans.Get(key); err == nil && enc != nil {
				t.state.node = mustDecodeNode(t.state.hash[:], enc, 0)
			} else if node := t.db.dirties[key]; node != nil {
				t.state.node = node.node
			} else {
				blob, err := t.db.diskdb.Get([]byte(key))
				if blob == nil || err != nil {
					log.Error("Missing referenced node", "owner", owner, "hash", t.state.hash, "path", fmt.Sprintf("%x%x", t.state.prefix, path))
					return false
					//panic(fmt.Sprintf("missing referenced node %x (searching for %x:%x at %x%x)", key, owner, t.state.hash, t.state.prefix, path))
				}
				t.state.node = mustDecodeNode(t.state.hash[:], blob, 0)
			}
		}
		// If we reached an account node, extract the storage trie root to continue on
		if path[0] == 0xff {
			// Retrieve the storage trie root and abort if empty
			if have, ok := t.state.node.(valueNode); ok {
				var account struct {
					Nonce    uint64
					Balance  *big.Int
					Root     common.Hash
					CodeHash []byte
				}
				if err := rlp.DecodeBytes(have, &account); err != nil {
					panic(err)
				}
				if account.Root == emptyRoot {
					return false
				}
				// Create a new nesting in the traversal and continue on that depth
				t.state, path = &traverserState{
					parent: t.state,
					prefix: append(t.state.prefix, 0xff),
					node:   hashNode(account.Root[:]),
				}, path[1:]
				continue
			}
			panic(fmt.Sprintf("liveness check path swap terminated on non value node: %T", t.state.node))
		}
		// Descend into the trie following the specified path. This code segment must
		// be able to handle both simplified raw nodes kept in this cache as well as
		// cold nodes loaded directly from disk.
		switch n := t.state.node.(type) {
		case *rawShortNode:
			if prefixLen(n.Key, path) == len(n.Key) {
				t.state, path = &traverserState{
					parent: t.state,
					prefix: append(t.state.prefix, path[:len(n.Key)]...),
					node:   n.Val,
				}, path[len(n.Key):]
				continue
			}
			return false

		case *shortNode:
			if prefixLen(n.Key, path) == len(n.Key) {
				t.state, path = &traverserState{
					parent: t.state,
					prefix: append(t.state.prefix, path[:len(n.Key)]...),
					node:   n.Val,
				}, path[len(n.Key):]
				continue
			}
			return false

		case rawFullNode:
			if child := n[path[0]]; child != nil {
				t.state, path = &traverserState{
					parent: t.state,
					prefix: append(t.state.prefix, path[0]),
					node:   child,
				}, path[1:]
				continue
			}
			return false

		case *fullNode:
			if child := n.Children[path[0]]; child != nil {
				t.state, path = &traverserState{
					parent: t.state,
					prefix: append(t.state.prefix, path[0]),
					node:   child,
				}, path[1:]
				continue
			}
			return false

		default:
			panic(fmt.Sprintf("unknown node type: %T", n))
		}
	}
	// The prefix should match perfectly here, check if the hashes matches
	if t.state.hash != (common.Hash{}) { // expanded/cached hash node
		return t.state.hash == hash
	}
	if have, ok := t.state.node.(hashNode); ok { // collapsed hash node
		t.state.hash = common.BytesToHash(have)
		return t.state.hash == hash
	}
	return false
}

// unref marks the current traversal nodes as *not* containing the specific trie
// node having been searched for. It is used by searches in subsequent tries to
// avoid reiterating the exact same sub-tries.
func (t *traverser) unref(count int, unrefs map[common.Hash]bool) {
	state := t.state
	for state != nil && count > 0 {
		// If we've found a hash node, store it as a subresult
		if state.hash != (common.Hash{}) {
			unrefs[state.hash] = true
			count--
		}
		// Traverse further up to the next hash node
		state = state.parent
	}
}
