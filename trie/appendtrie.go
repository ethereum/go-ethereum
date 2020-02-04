// Copyright 2014 The go-ethereum Authors
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
)

// AppendOnlyTrie is a Merkle Patricia Trie, which can only be used for
// constructing a trie from a sequence of sorted leafs, in descending order
type AppendOnlyTrie struct {
	root node
}

func NewAppendOnlyTrie() *AppendOnlyTrie {
	return &AppendOnlyTrie{root:nil}
}

func (t *AppendOnlyTrie) TryUpdate(key, value []byte) error {
	k := keybytesToHex(key)
	if len(value) == 0 {
		panic("deletion not supported")
	}
	t.root = t.insert(t.root, nil, k, valueNode(value))
	return nil
}

func (t *AppendOnlyTrie) insert(n node, prefix, key []byte, value node) node {
	if len(key) == 0 {
		return value
	}
	switch n := n.(type) {
	case *shortNode:
		matchlen := prefixLen(key, n.Key)
		// If the whole key matches, it already exists
		if matchlen == len(n.Key) {
			n.Val = t.insert(n.Val, append(prefix, key[:matchlen]...), key[matchlen:], value)
			n.flags = nodeFlag{dirty: true}
			return n
		}
		// Otherwise branch out at the index where they differ.
		branch := &fullNode{flags: nodeFlag{dirty: true}}
		branch.Children[n.Key[matchlen]]= t.insert(nil, append(prefix, n.Key[:matchlen+1]...), n.Key[matchlen+1:], n.Val)
		// TODO: We can now shoot off n.Val for hashing
		branch.Children[key[matchlen]]= t.insert(nil, append(prefix, key[:matchlen+1]...), key[matchlen+1:], value)

		// Replace this shortNode with the branch if it occurs at index 0.
		if matchlen == 0 {
			return branch
		}
		// Otherwise, replace it with a short node leading up to the branch.
		n.Key = key[:matchlen]
		n.Val = branch
		n.flags = nodeFlag{dirty: true}
		return n

	case *fullNode:
		n.flags = nodeFlag{dirty: true}
		n.Children[key[0]] = t.insert(n.Children[key[0]], append(prefix, key[0]), key[1:], value)
		return n

	case nil:
		return &shortNode{key, value, nodeFlag{dirty: true}}

	case hashNode:
		// We've hit a part of the trie that isn't loaded yet -- this means
		// someone inserted
		panic("hash resolution not supported")

	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}
}

func (t *AppendOnlyTrie) Hash() common.Hash {
	if t.root == nil {
		return emptyRoot
	}
	h := newHasher(false)
	defer returnHasherToPool(h)
	hashed, cached := h.hash(t.root, true)
	t.root = cached
	return common.BytesToHash(hashed.(hashNode))
}