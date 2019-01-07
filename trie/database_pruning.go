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
}

// newPruner creates a new trie pruner tied to the liveness of all the currently
// referenced in-memory nodes, except the specified one (currently being pruned).
func (db *Database) newPruner(skip common.Hash) *pruner {
	// Create the set of traversers based on the live tries
	var traversers []*traverser
	for key := range db.dirties[metaRoot].children {
		if _, root := splitNodeKey(key); root != skip {
			traversers = append(traversers, &traverser{
				db: db,
				state: &tranverserState{
					node: hashNode(root[:]),
				},
			})
		}
	}
	// Assemble and return the pruner
	return &pruner{
		db:    db,
		tries: traversers,
	}
}

// prune deletes a trie node from disk if there are no more live references to
// it, cascading until all dangling nodes are removed.
func (p *pruner) prune(owner common.Hash, hash common.Hash, path []byte, batch ethdb.Batch) {
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
	for _, trie := range p.tries {
		if trie.live(owner, hash, crosspath) {
			return
		}
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
	batch.Delete(dead)
	p.db.prunenodes++
	p.db.prunesize += common.StorageSize(len(blob))

	iterateRefs(node, path, func(path []byte, hash common.Hash) error {
		p.prune(owner, hash, path, batch)
		return nil
	})
}

// traverser is a stateful trie traversal data structure used by the pruner to
// verify the liveness of a node within a specific trie. The reason for having
// a separate data structure is to allow reusing previous traversals to check
// the liveness of nested nodes (i.e. entire subtried during pruning).
type traverser struct {
	db    *Database        // Trie database for accessing dirty and clean data
	state *tranverserState // Leftover state from the previous traversals
}

// tranverserState is the internal state of a trie traverser.
type tranverserState struct {
	parent *tranverserState // Parent traverser to allow backtracking
	prefix []byte           // Path leading up to the root of this traverser
	node   node             // Trie node where this traverser is currently at
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
func (t *traverser) live(owner common.Hash, hash common.Hash, path []byte) bool {
	// Rewind the traverser until it's prefix is actually a prefix of the path
	for !bytes.HasPrefix(path, t.state.prefix) {
		t.state = t.state.parent
	}
	// Traverse downward until the prefix matches the path completely
	path = path[len(t.state.prefix):]
	for len(path) > 0 {
		// If we're at a hash node, expand before continuing
		if n, ok := t.state.node.(hashNode); ok {
			// Generate the database key for this hash node
			var (
				key  string
				hash = common.BytesToHash(n)
			)
			if len(t.state.prefix) < 2*common.HashLength {
				key = makeNodeKey(common.Hash{}, hash)
			} else {
				key = makeNodeKey(owner, hash)
			}
			// Replace the node in the traverser with the expanded one
			if enc, err := t.db.cleans.Get(key); err == nil && enc != nil {
				t.state.node = mustDecodeNode(hash[:], enc, 0)
			} else if node := t.db.dirties[key]; node != nil {
				t.state.node = node.node
			} else {
				blob, err := t.db.diskdb.Get([]byte(key))
				if blob == nil || err != nil {
					panic(fmt.Sprintf("missing referenced node %x (searching for %x:%x at %x%x)", key, owner, hash, t.state.prefix, path))
				}
				t.state.node = mustDecodeNode(hash[:], blob, 0)
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
				t.state, path = &tranverserState{
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
				t.state, path = &tranverserState{
					parent: t.state,
					prefix: append(t.state.prefix, path[:len(n.Key)]...),
					node:   n.Val,
				}, path[len(n.Key):]
				continue
			}
			return false

		case *shortNode:
			if prefixLen(n.Key, path) == len(n.Key) {
				t.state, path = &tranverserState{
					parent: t.state,
					prefix: append(t.state.prefix, path[:len(n.Key)]...),
					node:   n.Val,
				}, path[len(n.Key):]
				continue
			}
			return false

		case rawFullNode:
			if child := n[path[0]]; child != nil {
				t.state, path = &tranverserState{
					parent: t.state,
					prefix: append(t.state.prefix, path[0]),
					node:   child,
				}, path[1:]
				continue
			}
			return false

		case *fullNode:
			if child := n.Children[path[0]]; child != nil {
				t.state, path = &tranverserState{
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
	if have, ok := t.state.node.(hashNode); ok {
		return common.BytesToHash(have) == hash
	}
	if have, _ := t.state.node.cache(); have != nil {
		return common.BytesToHash(have) == hash
	}
	return false
}
