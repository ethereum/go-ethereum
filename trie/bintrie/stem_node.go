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
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
)

// StemNode represents a group of `NodeWith` values sharing the same stem.
type StemNode struct {
	Stem   []byte   // Stem path to get to 256 values
	Values [][]byte // All values, indexed by the last byte of the key.
	depth  int      // Depth of the node
}

// Get retrieves the value for the given key.
func (bt *StemNode) Get(key []byte, _ NodeResolverFn) ([]byte, error) {
	panic("this should not be called directly")
}

// Insert inserts a new key-value pair into the node.
func (bt *StemNode) Insert(key []byte, value []byte, _ NodeResolverFn, depth int) (BinaryNode, error) {
	if !bytes.Equal(bt.Stem, key[:31]) {
		bitStem := bt.Stem[bt.depth/8] >> (7 - (bt.depth % 8)) & 1

		n := &InternalNode{depth: bt.depth}
		bt.depth++
		var child, other *BinaryNode
		if bitStem == 0 {
			n.left = bt
			child = &n.left
			other = &n.right
		} else {
			n.right = bt
			child = &n.right
			other = &n.left
		}

		bitKey := key[n.depth/8] >> (7 - (n.depth % 8)) & 1
		if bitKey == bitStem {
			var err error
			*child, err = (*child).Insert(key, value, nil, depth+1)
			if err != nil {
				return n, fmt.Errorf("insert error: %w", err)
			}
			*other = Empty{}
		} else {
			var values [256][]byte
			values[key[31]] = value
			*other = &StemNode{
				Stem:   slices.Clone(key[:31]),
				Values: values[:],
				depth:  depth + 1,
			}
		}
		return n, nil
	}
	if len(value) != 32 {
		return bt, errors.New("invalid insertion: value length")
	}
	bt.Values[key[31]] = value
	return bt, nil
}

// Copy creates a deep copy of the node.
func (bt *StemNode) Copy() BinaryNode {
	var values [256][]byte
	for i, v := range bt.Values {
		values[i] = slices.Clone(v)
	}
	return &StemNode{
		Stem:   slices.Clone(bt.Stem),
		Values: values[:],
		depth:  bt.depth,
	}
}

// GetHeight returns the height of the node.
func (bt *StemNode) GetHeight() int {
	return 1
}

// Hash returns the hash of the node.
func (bt *StemNode) Hash() common.Hash {
	var data [NodeWidth]common.Hash
	for i, v := range bt.Values {
		if v != nil {
			h := sha256.Sum256(v)
			data[i] = common.BytesToHash(h[:])
		}
	}

	h := sha256.New()
	for level := 1; level <= 8; level++ {
		for i := range NodeWidth / (1 << level) {
			h.Reset()

			if data[i*2] == (common.Hash{}) && data[i*2+1] == (common.Hash{}) {
				data[i] = common.Hash{}
				continue
			}

			h.Write(data[i*2][:])
			h.Write(data[i*2+1][:])
			data[i] = common.Hash(h.Sum(nil))
		}
	}

	h.Reset()
	h.Write(bt.Stem)
	h.Write([]byte{0})
	h.Write(data[0][:])
	return common.BytesToHash(h.Sum(nil))
}

// CollectNodes collects all child nodes at a given path, and flushes it
// into the provided node collector.
func (bt *StemNode) CollectNodes(path []byte, flush NodeFlushFn) error {
	flush(path, bt)
	return nil
}

// GetValuesAtStem retrieves the group of values located at the given stem key.
func (bt *StemNode) GetValuesAtStem(_ []byte, _ NodeResolverFn) ([][]byte, error) {
	return bt.Values[:], nil
}

// InsertValuesAtStem inserts a full value group at the given stem in the internal node.
// Already-existing values will be overwritten.
func (bt *StemNode) InsertValuesAtStem(key []byte, values [][]byte, _ NodeResolverFn, depth int) (BinaryNode, error) {
	if !bytes.Equal(bt.Stem, key[:31]) {
		bitStem := bt.Stem[bt.depth/8] >> (7 - (bt.depth % 8)) & 1

		n := &InternalNode{depth: bt.depth}
		bt.depth++
		var child, other *BinaryNode
		if bitStem == 0 {
			n.left = bt
			child = &n.left
			other = &n.right
		} else {
			n.right = bt
			child = &n.right
			other = &n.left
		}

		bitKey := key[n.depth/8] >> (7 - (n.depth % 8)) & 1
		if bitKey == bitStem {
			var err error
			*child, err = (*child).InsertValuesAtStem(key, values, nil, depth+1)
			if err != nil {
				return n, fmt.Errorf("insert error: %w", err)
			}
			*other = Empty{}
		} else {
			*other = &StemNode{
				Stem:   slices.Clone(key[:31]),
				Values: values,
				depth:  n.depth + 1,
			}
		}
		return n, nil
	}

	// same stem, just merge the two value lists
	for i, v := range values {
		if v != nil {
			bt.Values[i] = v
		}
	}
	return bt, nil
}

func (bt *StemNode) toDot(parent, path string) string {
	me := fmt.Sprintf("stem%s", path)
	ret := fmt.Sprintf("%s [label=\"stem=%x c=%x\"]\n", me, bt.Stem, bt.Hash())
	ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
	for i, v := range bt.Values {
		if v != nil {
			ret = fmt.Sprintf("%s%s%x [label=\"%x\"]\n", ret, me, i, v)
			ret = fmt.Sprintf("%s%s -> %s%x\n", ret, me, me, i)
		}
	}
	return ret
}

// Key returns the full key for the given index.
func (bt *StemNode) Key(i int) []byte {
	var ret [32]byte
	copy(ret[:], bt.Stem)
	ret[StemSize] = byte(i)
	return ret[:]
}
