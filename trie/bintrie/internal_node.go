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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeebo/blake3"
)

func keyToPath(depth int, key []byte) ([]byte, error) {
	path := make([]byte, 0, depth+1)

	if depth > 31*8 {
		return nil, errors.New("node too deep")
	}

	for i := range depth + 1 {
		bit := key[i/8] >> (7 - (i % 8)) & 1
		path = append(path, bit)
	}

	return path, nil
}

// InternalNode is a binary trie internal node.
type InternalNode struct {
	Left, Right BinaryNode
	depth       int
}

// GetValuesAtStem retrieves the group of values located at the given stem key.
func (bt *InternalNode) GetValuesAtStem(stem []byte, resolver NodeResolverFn) ([][]byte, error) {
	if bt.depth > 31*8 {
		return nil, errors.New("node too deep")
	}

	bit := stem[bt.depth/8] >> (7 - (bt.depth % 8)) & 1
	var child *BinaryNode
	if bit == 0 {
		child = &bt.Left
	} else {
		child = &bt.Right
	}

	if hn, ok := (*child).(HashedNode); ok {
		path, err := keyToPath(bt.depth, stem)
		if err != nil {
			return nil, fmt.Errorf("GetValuesAtStem resolve error: %w", err)
		}
		data, err := resolver(path, common.Hash(hn))
		if err != nil {
			return nil, fmt.Errorf("GetValuesAtStem resolve error: %w", err)
		}
		node, err := DeserializeNode(data, bt.depth+1)
		if err != nil {
			return nil, fmt.Errorf("GetValuesAtStem node deserialization error: %w", err)
		}
		*child = node
	}
	return (*child).GetValuesAtStem(stem, resolver)
}

// Get retrieves the value for the given key.
func (bt *InternalNode) Get(key []byte, resolver NodeResolverFn) ([]byte, error) {
	values, err := bt.GetValuesAtStem(key[:31], resolver)
	if err != nil {
		return nil, fmt.Errorf("Get error: %w", err)
	}
	return values[key[31]], nil
}

// Insert inserts a new key-value pair into the trie.
func (bt *InternalNode) Insert(key []byte, value []byte, resolver NodeResolverFn) (BinaryNode, error) {
	var values [256][]byte
	values[key[31]] = value
	return bt.InsertValuesAtStem(key[:31], values[:], resolver, 0)
}

// Copy creates a deep copy of the node.
func (bt *InternalNode) Copy() BinaryNode {
	return &InternalNode{
		Left:  bt.Left.Copy(),
		Right: bt.Right.Copy(),
		depth: bt.depth,
	}
}

// Hash returns the hash of the node.
func (bt *InternalNode) Hash() common.Hash {
	h := blake3.New()
	if bt.Left != nil {
		h.Write(bt.Left.Hash().Bytes())
	} else {
		h.Write(zero[:])
	}
	if bt.Right != nil {
		h.Write(bt.Right.Hash().Bytes())
	} else {
		h.Write(zero[:])
	}
	return common.BytesToHash(h.Sum(nil))
}

// InsertValuesAtStem inserts a full value group at the given stem in the internal node.
// Already-existing values will be overwritten.
func (bt *InternalNode) InsertValuesAtStem(stem []byte, values [][]byte, resolver NodeResolverFn, depth int) (BinaryNode, error) {
	bit := stem[bt.depth/8] >> (7 - (bt.depth % 8)) & 1
	var (
		child *BinaryNode
		err   error
	)
	if bit == 0 {
		child = &bt.Left
	} else {
		child = &bt.Right
	}

	*child, err = (*child).InsertValuesAtStem(stem, values, resolver, depth+1)
	return bt, err
}

// CollectNodes collects all child nodes at a given path, and flushes it
// into the provided node collector.
func (bt *InternalNode) CollectNodes(path []byte, flushfn NodeFlushFn) error {
	if bt.Left != nil {
		var p [256]byte
		copy(p[:], path)
		childpath := p[:len(path)]
		childpath = append(childpath, 0)
		if err := bt.Left.CollectNodes(childpath, flushfn); err != nil {
			return err
		}
	}
	if bt.Right != nil {
		var p [256]byte
		copy(p[:], path)
		childpath := p[:len(path)]
		childpath = append(childpath, 1)
		if err := bt.Right.CollectNodes(childpath, flushfn); err != nil {
			return err
		}
	}
	flushfn(path, bt)
	return nil
}

// GetHeight returns the height of the node.
func (bt *InternalNode) GetHeight() int {
	var (
		leftHeight  int
		rightHeight int
	)
	if bt.Left != nil {
		leftHeight = bt.Left.GetHeight()
	}
	if bt.Right != nil {
		rightHeight = bt.Right.GetHeight()
	}
	return 1 + max(leftHeight, rightHeight)
}

func (bt *InternalNode) toDot(parent, path string) string {
	me := fmt.Sprintf("internal%s", path)
	ret := fmt.Sprintf("%s [label=\"I: %x\"]\n", me, bt.Hash())
	if len(parent) > 0 {
		ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
	}

	if bt.Left != nil {
		ret = fmt.Sprintf("%s%s", ret, bt.Left.toDot(me, fmt.Sprintf("%s%02x", path, 0)))
	}
	if bt.Right != nil {
		ret = fmt.Sprintf("%s%s", ret, bt.Right.toDot(me, fmt.Sprintf("%s%02x", path, 1)))
	}

	return ret
}
