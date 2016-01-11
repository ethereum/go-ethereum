// Copyright 2015 The go-ethereum Authors
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
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// NodeIterator is an iterator to traverse the entire state trie post-order,
// including all of the contract code and contract state tries.
type NodeIterator struct {
	state *StateDB // State being iterated

	stateIt *trie.NodeIterator // Primary iterator for the global state trie
	dataIt  *trie.NodeIterator // Secondary iterator for the data trie of a contract

	accountHash common.Hash // Hash of the node containing the account
	codeHash    common.Hash // Hash of the contract source code
	code        []byte      // Source code associated with a contract

	Hash   common.Hash // Hash of the current entry being iterated (nil if not standalone)
	Entry  interface{} // Current state entry being iterated (internal representation)
	Parent common.Hash // Hash of the first full ancestor node (nil if current is the root)
}

// NewNodeIterator creates an post-order state node iterator.
func NewNodeIterator(state *StateDB) *NodeIterator {
	return &NodeIterator{
		state: state,
	}
}

// Next moves the iterator to the next node, returning whether there are any
// further nodes.
func (it *NodeIterator) Next() bool {
	it.step()
	return it.retrieve()
}

// step moves the iterator to the next entry of the state trie.
func (it *NodeIterator) step() {
	// Abort if we reached the end of the iteration
	if it.state == nil {
		return
	}
	// Initialize the iterator if we've just started
	if it.stateIt == nil {
		it.stateIt = trie.NewNodeIterator(it.state.trie.Trie)
	}
	// If we had data nodes previously, we surely have at least state nodes
	if it.dataIt != nil {
		if cont := it.dataIt.Next(); !cont {
			it.dataIt = nil
		}
		return
	}
	// If we had source code previously, discard that
	if it.code != nil {
		it.code = nil
		return
	}
	// Step to the next state trie node, terminating if we're out of nodes
	if cont := it.stateIt.Next(); !cont {
		it.state, it.stateIt = nil, nil
		return
	}
	// If the state trie node is an internal entry, leave as is
	if !it.stateIt.Leaf {
		return
	}
	// Otherwise we've reached an account node, initiate data iteration
	var account struct {
		Nonce    uint64
		Balance  *big.Int
		Root     common.Hash
		CodeHash []byte
	}
	err := rlp.Decode(bytes.NewReader(it.stateIt.LeafBlob), &account)
	if err != nil {
		panic(err)
	}
	dataTrie, err := trie.New(account.Root, it.state.db)
	if err != nil {
		panic(err)
	}
	it.dataIt = trie.NewNodeIterator(dataTrie)
	if !it.dataIt.Next() {
		it.dataIt = nil
	}
	if bytes.Compare(account.CodeHash, emptyCodeHash) != 0 {
		it.codeHash = common.BytesToHash(account.CodeHash)
		it.code, err = it.state.db.Get(account.CodeHash)
		if err != nil {
			panic(fmt.Sprintf("code %x: %v", account.CodeHash, err))
		}
	}
	it.accountHash = it.stateIt.Parent
}

// retrieve pulls and caches the current state entry the iterator is traversing.
// The method returns whether there are any more data left for inspection.
func (it *NodeIterator) retrieve() bool {
	// Clear out any previously set values
	it.Hash, it.Entry = common.Hash{}, nil

	// If the iteration's done, return no available data
	if it.state == nil {
		return false
	}
	// Otherwise retrieve the current entry
	switch {
	case it.dataIt != nil:
		it.Hash, it.Entry, it.Parent = it.dataIt.Hash, it.dataIt.Node, it.dataIt.Parent
		if it.Parent == (common.Hash{}) {
			it.Parent = it.accountHash
		}
	case it.code != nil:
		it.Hash, it.Entry, it.Parent = it.codeHash, it.code, it.accountHash
	case it.stateIt != nil:
		it.Hash, it.Entry, it.Parent = it.stateIt.Hash, it.stateIt.Node, it.stateIt.Parent
	}
	return true
}
