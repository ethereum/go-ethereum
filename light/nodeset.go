// Copyright 2017 The go-ethereum Authors
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

package light

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
)

// NodeSet stores a set of trie nodes. It implements trie.Database and can also
// act as a cache for another trie.Database.
type NodeSet struct {
	nodes map[string][]byte
	order []string

	dataSize int
	lock     sync.RWMutex
}

// NewNodeSet creates an empty node set
func NewNodeSet() *NodeSet {
	return &NodeSet{
		nodes: make(map[string][]byte),
	}
}

// Put stores a new node in the set
func (db *NodeSet) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	if _, ok := db.nodes[string(key)]; ok {
		return nil
	}
	keystr := string(key)

	db.nodes[keystr] = common.CopyBytes(value)
	db.order = append(db.order, keystr)
	db.dataSize += len(value)

	return nil
}

// Delete removes a node from the set
func (db *NodeSet) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.nodes, string(key))
	return nil
}

// Get returns a stored node
func (db *NodeSet) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.nodes[string(key)]; ok {
		return entry, nil
	}
	return nil, errors.New("not found")
}

// Has returns true if the node set contains the given key
func (db *NodeSet) Has(key []byte) (bool, error) {
	_, err := db.Get(key)
	return err == nil, nil
}

// KeyCount returns the number of nodes in the set
func (db *NodeSet) KeyCount() int {
	db.lock.RLock()
	defer db.lock.RUnlock()

	return len(db.nodes)
}

// DataSize returns the aggregated data size of nodes in the set
func (db *NodeSet) DataSize() int {
	db.lock.RLock()
	defer db.lock.RUnlock()

	return db.dataSize
}

// NodeList converts the node set to a NodeList
func (db *NodeSet) NodeList() NodeList {
	db.lock.RLock()
	defer db.lock.RUnlock()

	var values NodeList
	for _, key := range db.order {
		values = append(values, db.nodes[key])
	}
	return values
}

// Store writes the contents of the set to the given database
func (db *NodeSet) Store(target ethdb.Writer) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	for key, value := range db.nodes {
		target.Put([]byte(key), value)
	}
}

// NodeList stores an ordered list of trie nodes. It implements ethdb.Writer.
type NodeList []rlp.RawValue

// Store writes the contents of the list to the given database
func (n NodeList) Store(db ethdb.Writer) {
	for _, node := range n {
		db.Put(crypto.Keccak256(node), node)
	}
}

// NodeSet converts the node list to a NodeSet
func (n NodeList) NodeSet() *NodeSet {
	db := NewNodeSet()
	n.Store(db)
	return db
}

// Put stores a new node at the end of the list
func (n *NodeList) Put(key []byte, value []byte) error {
	*n = append(*n, value)
	return nil
}

// Delete panics as there's no reason to remove a node from the list.
func (n *NodeList) Delete(key []byte) error {
	panic("not supported")
}

// DataSize returns the aggregated data size of nodes in the list
func (n NodeList) DataSize() int {
	var size int
	for _, node := range n {
		size += len(node)
	}
	return size
}
