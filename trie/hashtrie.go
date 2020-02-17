// Copyright 2020 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/common"
)

// HashTrie is a Merkle Patricia Trie, which can only be used for
// constructing a trie from a sequence of sorted leafs, in descending order
type HashTrie struct {
	root    node
	rootKey []byte
	build   []node
	hasher  *hasher
}

func NewHashTrie() *HashTrie {
	return &HashTrie{root: nil, rootKey: nil, build: nil, hasher: newHasher(false)}
}

func (t *HashTrie) TryUpdate(key, value []byte) error {
	k := keybytesToHex(key)
	if len(value) == 0 {
		panic("deletion not supported")
	}
	t.root = t.insert(t.root, nil, k, valueNode(value))
	return nil
}

func (t *HashTrie) insert(n node, prefix, key []byte, value node) node {
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

		if key[matchlen] < n.Key[matchlen] {
			panic("Keys were inserted unsorted, this should not happen")
		}

		// Otherwise branch out at the index where they differ.
		branch := &fullNode{flags: nodeFlag{dirty: true}}
		hashed, _ := newHasher(false).hash(t.insert(nil, append(prefix, n.Key[:matchlen+1]...), n.Key[matchlen+1:], n.Val), false)
		branch.Children[n.Key[matchlen]] = hashed.(hashNode)

		// Hashing the sub-node, nothing will be added to this sub-branch
		branch.Children[key[matchlen]] = t.insert(nil, append(prefix, key[:matchlen+1]...), key[matchlen+1:], value)

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
		// If any previous child wasn't already hashed, do it now since
		// the keys arrive in order, so if a branch is here then whatever
		// came before can safely be hashed.
		for i := int(key[0]) - 1; i > 0; i -= 1 {
			switch n.Children[i].(type) {
			case *shortNode, *fullNode, *valueNode:
				hashed, _ := newHasher(false).hash(n.Children[i], false)
				n.Children[i] = hashed
			// hash encountred, the rest has already been hashed
			case hashNode:
				break
			default:
				panic("invalid node")
			}
		}
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

func (t *HashTrie) Hash() common.Hash {
	if t.root == nil {
		return emptyRoot
	}
	h := newHasher(false)
	defer returnHasherToPool(h)
	hashed, cached := h.hash(t.root, true)
	t.root = cached
	return common.BytesToHash(hashed.(hashNode))
}

type StackTrieItem struct {
	ext          shortNode
	branch       fullNode
	depth        int
	useBranch    bool
	keyUntilHere []byte
}

type StackTrie struct {
	stack  []StackTrieItem
	top    int
	hasher *hasher
}

func NewStackTrie() *StackTrie {
	return &StackTrie{
		top: -1,
		stack: []StackTrieItem{
			StackTrieItem{},
		},
		hasher: newHasher(false),
	}
}

func (st *StackTrie) TryUpdate(key, value []byte) error {
	k := keybytesToHex(key)
	if len(value) == 0 {
		panic("deletion not supported")
	}
	st.insert(&st.stack[0].ext, nil, k, valueNode(value))
	return nil
}

func (st *StackTrie) insert(n node, prefix, key []byte, value node) node {
	// Special case: the trie is empty
	if st.top == -1 {
		st.top = 0
		st.stack[st.top].depth = 0
		st.stack[st.top].ext.Key = key
		st.stack[st.top].ext.Val, _ = st.hasher.hash(value, false)
		st.stack[st.top].keyUntilHere = []byte("")

		return &st.stack[st.top].ext
	}

	// Use the prefix key to find the stack level in which the code needs to
	// be inserted.
	level := -1
	for index := st.top; index >= 0; index-- {
		level = index
		if bytes.Equal(st.stack[level].keyUntilHere, key[:len(st.stack[level].keyUntilHere)]) {
			// Found the common denominator, stop the search
			break
		}
	}

	// Already hash the value, which it will be anyway
	hv, _ := st.hasher.hash(value, false)

	// The difference happens at this level, find out where
	// exactly. The extension part of the fullnode part?
	extStart := len(st.stack[level].keyUntilHere)
	extEnd := extStart + len(st.stack[level].ext.Key)
	if bytes.Equal(st.stack[level].ext.Key, key[extStart:extEnd]) {
		// The extension and the key are identical on the length of
		// the extension, so st.stack[level].ext.Val should be a fullNode and
		// the difference should be found there. Panic if this is
		// not the case.
		fn := st.stack[level].ext.Val.(*fullNode)

		// The correct entry is the only one that isn't nil
		for i := 15; i >= 0; i-- {
			if fn.Children[i] != nil {
				switch fn.Children[i].(type) {
				// Only hash entries that are not already hashed
				case *fullNode, *shortNode:
					fn.Children[i], _ = st.hasher.hash(fn.Children[i], false)
					st.top = level
				default:
				}
				break
			}
		}

		// That fullNode should have at most one non-hashNode child,
		// hash it because no more nodes will be inserted in it.
		if len(st.stack) == st.top+1 {
			st.stack = append(st.stack, StackTrieItem{})
		}

		st.top++
		keyUntilHere := len(st.stack[level].keyUntilHere) + len(st.stack[level].ext.Key) + 1
		st.stack[level].branch.Children[key[keyUntilHere]] = &st.stack[st.top].ext
		st.stack[st.top].keyUntilHere = key[:keyUntilHere]
		st.stack[st.top].ext.Key = key[keyUntilHere:]
		st.stack[st.top].ext.Val = hv
		st.stack[st.top].ext.flags = nodeFlag{dirty: true}
		st.stack[st.top].depth = st.stack[level].depth + 1
	} else {
		// extension keys differ, need to create a split and
		// hash the former node.
		whereitdiffers := 0
		offset := len(st.stack[level].keyUntilHere)
		for i := range st.stack[level].ext.Key {
			if key[offset+i] != st.stack[level].ext.Key[i] {
				whereitdiffers = i
				break
			}
		}

		// Start by hashing the node right after the extension,
		// to free some space.
		var hn node
		switch st.stack[level].ext.Val.(type) {
		case *fullNode:
			h, _ := st.hasher.hash(st.stack[level].ext.Val, false)
			hn = h.(hashNode)
		case hashNode, valueNode:
			hn = st.stack[level].ext.Val
		default:
			panic("Encountered unexpected node type")
		}

		// Allocate the next full node, it's going to be
		// reused several times.
		if len(st.stack) == st.top+1 {
			st.stack = append(st.stack, StackTrieItem{})
		}
		st.top++

		// Store the partially-hashed old node in the newly allocated
		// slot, in order to finish the hashing.
		slot := st.stack[level].ext.Key[whereitdiffers]
		st.stack[st.top].ext.Key = st.stack[level].ext.Key[whereitdiffers+1:]
		st.stack[st.top].ext.Val = hn
		st.stack[st.top].ext.flags = nodeFlag{dirty: true}

		// Hasher directement la branche si l'ext est vide
		h, _ := st.hasher.hash(&st.stack[st.top].ext, false)
		st.stack[level].branch.Children[slot] = h.(hashNode)
		st.stack[level].ext.Val = &st.stack[level].branch
		st.stack[level].ext.Key = st.stack[level].ext.Key[:whereitdiffers]

		// Now use the newly allocated+hashed stack st.stack[level] to store
		// the rest of the inserted (key, value) pair.
		slot = key[whereitdiffers+len(st.stack[level].keyUntilHere)]
		st.stack[level].branch.Children[slot] = &st.stack[st.top].ext
		st.stack[st.top].ext.Key = key[whereitdiffers+len(st.stack[level].keyUntilHere)+1:]
		st.stack[st.top].ext.Val = hv
		st.stack[st.top].keyUntilHere = key[:whereitdiffers+len(st.stack[level].keyUntilHere)+1]
		st.stack[st.top].depth = st.stack[level].depth + 1

	}

	// if ext.length == 0, directly return the full node.
	if len(st.stack[0].ext.Key) == 0 {
		return &st.stack[0].branch
	}
	return &st.stack[0].ext
}

func (st *StackTrie) Hash() common.Hash {
	if st.top == -1 {
		return emptyRoot
	}

	h, _ := st.hasher.hash(&st.stack[0].ext, false)
	return common.BytesToHash(h.(hashNode))
}
