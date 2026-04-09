// Copyright 2025 go-ethereum Authors
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

package bintrie

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math/bits"
	"runtime"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// parallelDepth returns the tree depth below which Hash() spawns goroutines.
func parallelDepth() int {
	return min(bits.Len(uint(runtime.NumCPU())), 8)
}

// isDirty reports whether a BinaryNode child needs rehashing.
func isDirty(n BinaryNode) bool {
	switch v := n.(type) {
	case *InternalNode:
		return v.mustRecompute
	case *StemNode:
		return v.mustRecompute
	default:
		return false
	}
}

func keyToPath(depth int, key []byte) ([]byte, error) {
	if depth > 31*8 {
		return nil, errors.New("node too deep")
	}
	path := make([]byte, 0, depth+1)
	for i := range depth + 1 {
		bit := key[i/8] >> (7 - (i % 8)) & 1
		path = append(path, bit)
	}
	return path, nil
}

// InternalNode is a binary trie internal node.
type InternalNode struct {
	children [2]BinaryNode // 0: left, 1: right
	depth    int

	mustRecompute bool        // true if the hash needs to be recomputed
	hash          common.Hash // cached hash when mustRecompute == false
}

// GetValuesAtStem retrieves the group of values located at the given stem key.
func (bt *InternalNode) GetValuesAtStem(stem []byte, resolver NodeResolverFn) ([][]byte, error) {
	if bt.depth > 31*8 {
		return nil, errors.New("node too deep")
	}
	bit := stem[bt.depth/8] >> (7 - (bt.depth % 8)) & 1
	if hn, ok := bt.children[bit].(HashedNode); ok {
		path, err := keyToPath(bt.depth, stem)
		if err != nil {
			return nil, fmt.Errorf("GetValuesAtStem resolve error: %w", err)
		}
		data, err := resolver(path, common.Hash(hn))
		if err != nil {
			return nil, fmt.Errorf("GetValuesAtStem resolve error: %w", err)
		}
		node, err := DeserializeNodeWithHash(data, bt.depth+1, common.Hash(hn))
		if err != nil {
			return nil, fmt.Errorf("GetValuesAtStem node deserialization error: %w", err)
		}
		bt.children[bit] = node
	}
	return bt.children[bit].GetValuesAtStem(stem, resolver)
}

// Get retrieves the value for the given key.
func (bt *InternalNode) Get(key []byte, resolver NodeResolverFn) ([]byte, error) {
	values, err := bt.GetValuesAtStem(key[:31], resolver)
	if err != nil {
		return nil, fmt.Errorf("get error: %w", err)
	}
	if values == nil {
		return nil, nil
	}
	return values[key[31]], nil
}

// Insert inserts a new key-value pair into the trie.
func (bt *InternalNode) Insert(key []byte, value []byte, resolver NodeResolverFn, depth int) (BinaryNode, error) {
	var values [256][]byte
	values[key[31]] = value
	return bt.InsertValuesAtStem(key[:31], values[:], resolver, depth)
}

// Copy creates a deep copy of the node.
func (bt *InternalNode) Copy() BinaryNode {
	return &InternalNode{
		children:      [2]BinaryNode{bt.children[0].Copy(), bt.children[1].Copy()},
		depth:         bt.depth,
		mustRecompute: bt.mustRecompute,
		hash:          bt.hash,
	}
}

// Hash returns the hash of the node.
func (bt *InternalNode) Hash() common.Hash {
	if !bt.mustRecompute {
		return bt.hash
	}

	// At shallow depths, parallelize when both children need rehashing:
	// hash left subtree in a goroutine, right subtree inline, then combine.
	// Skip goroutine overhead when only one child is dirty (common case
	// for narrow state updates that touch a single path through the trie).
	if bt.depth < parallelDepth() && isDirty(bt.children[0]) && isDirty(bt.children[1]) {
		var input [64]byte
		var lh common.Hash
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			lh = bt.children[0].Hash()
		}()
		rh := bt.children[1].Hash()
		copy(input[32:], rh[:])
		wg.Wait()
		copy(input[:32], lh[:])
		bt.hash = sha256.Sum256(input[:])
		bt.mustRecompute = false
		return bt.hash
	}

	// Deeper nodes: sequential using pooled hasher (goroutine overhead > hash cost)
	h := newSha256()
	defer returnSha256(h)
	for _, child := range bt.children {
		if child != nil {
			h.Write(child.Hash().Bytes())
		} else {
			h.Write(zero[:])
		}
	}
	bt.hash = common.BytesToHash(h.Sum(nil))
	bt.mustRecompute = false
	return bt.hash
}

// InsertValuesAtStem inserts a full value group at the given stem in the internal node.
// Already-existing values will be overwritten.
func (bt *InternalNode) InsertValuesAtStem(stem []byte, values [][]byte, resolver NodeResolverFn, depth int) (BinaryNode, error) {
	bit := stem[bt.depth/8] >> (7 - (bt.depth % 8)) & 1
	if bt.children[bit] == nil {
		bt.children[bit] = Empty{}
	}
	if hn, ok := bt.children[bit].(HashedNode); ok {
		path, err := keyToPath(bt.depth, stem)
		if err != nil {
			return nil, fmt.Errorf("InsertValuesAtStem resolve error: %w", err)
		}
		data, err := resolver(path, common.Hash(hn))
		if err != nil {
			return nil, fmt.Errorf("InsertValuesAtStem resolve error: %w", err)
		}
		node, err := DeserializeNodeWithHash(data, bt.depth+1, common.Hash(hn))
		if err != nil {
			return nil, fmt.Errorf("InsertValuesAtStem node deserialization error: %w", err)
		}
		bt.children[bit] = node
	}
	var err error
	bt.children[bit], err = bt.children[bit].InsertValuesAtStem(stem, values, resolver, depth+1)
	bt.mustRecompute = true
	return bt, err
}

// CollectNodes collects all child nodes at a given path, and flushes it
// into the provided node collector.
func (bt *InternalNode) CollectNodes(path []byte, flushfn NodeFlushFn) error {
	for i, child := range bt.children {
		if child != nil {
			var p [256]byte
			copy(p[:], path)
			childpath := p[:len(path)]
			childpath = append(childpath, byte(i))
			if err := child.CollectNodes(childpath, flushfn); err != nil {
				return err
			}
		}
	}
	flushfn(path, bt)
	return nil
}

// GetHeight returns the height of the node.
func (bt *InternalNode) GetHeight() int {
	var maxHeight int
	for _, child := range bt.children {
		if child != nil {
			maxHeight = max(maxHeight, child.GetHeight())
		}
	}
	return 1 + maxHeight
}

func (bt *InternalNode) toDot(parent, path string) string {
	me := fmt.Sprintf("internal%s", path)
	ret := fmt.Sprintf("%s [label=\"I: %x\"]\n", me, bt.Hash())
	if len(parent) > 0 {
		ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
	}
	for i, child := range bt.children {
		if child != nil {
			ret = fmt.Sprintf("%s%s", ret, child.toDot(me, fmt.Sprintf("%s%02x", path, i)))
		}
	}
	return ret
}
