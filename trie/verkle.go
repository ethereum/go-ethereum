// Copyright 2021 go-ethereum Authors
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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gballet/go-verkle"
)

// VerkleTrie is a wrapper around VerkleNode that implements the trie.Trie
// interface so that Verkle trees can be reused verbatim.
type VerkleTrie struct {
	root verkle.VerkleNode
	db   *Database
}

func NewVerkleTrie(root verkle.VerkleNode, db *Database) *VerkleTrie {
	return &VerkleTrie{
		root: root,
		db:   db,
	}
}

// GetKey returns the sha3 preimage of a hashed key that was previously used
// to store a value.
func (trie *VerkleTrie) GetKey(key []byte) []byte {
	return key
}

// TryGet returns the value for key stored in the trie. The value bytes must
// not be modified by the caller. If a node was not found in the database, a
// trie.MissingNodeError is returned.
func (trie *VerkleTrie) TryGet(key []byte) ([]byte, error) {
	return trie.root.Get(key, trie.db.DiskDB().Get)
}

// TryUpdate associates key with value in the trie. If value has length zero, any
// existing value is deleted from the trie. The value bytes must not be modified
// by the caller while they are stored in the trie. If a node was not found in the
// database, a trie.MissingNodeError is returned.
func (trie *VerkleTrie) TryUpdate(key, value []byte) error {
	return trie.root.Insert(key, value)
}

// TryDelete removes any existing value for key from the trie. If a node was not
// found in the database, a trie.MissingNodeError is returned.
func (trie *VerkleTrie) TryDelete(key []byte) error {
	return trie.root.Delete(key)
}

// Hash returns the root hash of the trie. It does not write to the database and
// can be used even if the trie doesn't have one.
func (trie *VerkleTrie) Hash() common.Hash {
	return trie.root.Hash()
}

// Commit writes all nodes to the trie's memory database, tracking the internal
// and external (for account tries) references.
func (trie *VerkleTrie) Commit(onleaf LeafCallback) (common.Hash, error) {
	flush := make(chan verkle.FlushableNode)
	go func() {
		trie.root.(*verkle.InternalNode).Flush(flush)
		close(flush)
	}()
	fmt.Println("commiting verkle")
	for n := range flush {
		value, err := n.Node.Serialize()
		if err != nil {
			panic(err)
		}
		fmt.Printf("%x %x %v\n", n.Hash[:], value[:], n.Node)

		if err := trie.db.DiskDB().Put(n.Hash[:], value); err != nil {
			return common.Hash{}, err
		}
	}

	return trie.root.Hash(), nil
}

// NodeIterator returns an iterator that returns nodes of the trie. Iteration
// starts at the key after the given start key.
func (trie *VerkleTrie) NodeIterator(startKey []byte) NodeIterator {
	it := &verkleNodeIterator{trie: trie, current: trie.root, stack: []verkleNodeIteratorState{verkleNodeIteratorState{Node: trie.root, Index: -1}}}
	it.Next(false)
	return it
}

// Prove constructs a Merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root), ending
// with the node that proves the absence of the key.
func (trie *VerkleTrie) Prove(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) error {
	panic("not implemented")
}

func (trie *VerkleTrie) Copy(db *Database) *VerkleTrie {
	return &VerkleTrie{
		root: trie.root.Copy(),
		db:   db,
	}
}
