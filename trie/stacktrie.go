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
	//"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// StackTrieItem represents an (extension, fullnode) tuple to be stored
// in a "stack" in order to be reused multiple times so as to save many
// allocations.
type StackTrieItem struct {
	ext          shortNode
	branch       fullNode
	depth        int
	useBranch    bool
	keyUntilHere []byte
}

// StackTrie is a "stack" of (extension, fullnode) tuples that are
// used to calculate the hash of a trie. The core idea is that at
// any time, only one branch is expanded and the rest is hashed as
// soon as it is determined it is no longer needed.
type StackTrie struct {
	stack  []StackTrieItem
	top    int
	hasher *hasher
}

// NewStackTrie builds a new stack trie. The whole stack space is
// pre-allocated so as to save reallocations down the road.
func NewStackTrie() *StackTrie {
	return &StackTrie{
		top:    -1,
		stack:  make([]StackTrieItem, 65),
		hasher: newHasher(false),
	}
}

func (st *StackTrie) TryUpdate(key, value []byte) error {
	k := keybytesToHex(key)
	if len(value) == 0 {
		panic("deletion not supported")
	}
	st.insert(&st.stack[0].ext, nil, k, valueNode(value))
	//fmt.Println("trie=", &st.stack[0].ext)
	return nil
}

// alloc prepares the next stage in the stack for reuse.
func (st *StackTrie) alloc() {
	for i := 0; i < 16; i++ {
		st.stack[st.top+1].branch.Children[i] = nil
	}

	st.top++
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
		// the extension, so st.stack[level].ext.Val should point to
		// st.stack[level].branch, and the difference should be foud
		// there.
		var fn *fullNode
		fn = &st.stack[level].branch

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
		st.alloc()
		keyUntilHere := len(st.stack[level].keyUntilHere) + len(st.stack[level].ext.Key) + 1
		st.stack[level].branch.Children[key[keyUntilHere-1]] = &st.stack[st.top].ext
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

		// Special case: the split is at the first byte, in this case
		// the current ext needs to be skipped.
		if whereitdiffers == 0 {
			// Hash the existing node
			saveSlot := st.stack[level].ext.Key[0]
			st.stack[level].ext.Key = st.stack[level].ext.Key[1:]
			var h node
			if len(st.stack[level].ext.Key) == 0 {
				h, _ = st.hasher.hash(&st.stack[level].branch, false)
			} else {
				h, _ = st.hasher.hash(&st.stack[level].ext, false)
			}
			for i := range st.stack[level].branch.Children {
				st.stack[level].branch.Children[i] = nil
			}
			st.stack[level].branch.Children[saveSlot] = h
			// Set the ext key to empty
			st.stack[level].ext.Key = st.stack[level].ext.Key[:0]
			st.top = level

			// Insert the new leaf, starting with allocating more space
			// if needed.
			st.alloc()
			st.stack[st.top].ext.Key = key[offset+1:]
			st.stack[st.top].ext.Val = hv
			st.stack[level].branch.Children[key[offset]] = &st.stack[st.top].ext

			st.stack[st.top].keyUntilHere = key[:offset+1]

			// Update parent reference if this isn't the root
			if level > 0 {
				parentslot := key[offset-1]
				st.stack[level-1].branch.Children[parentslot] = &st.stack[level].branch
			}
		} else {
			// Start by hashing the node right after the extension,
			// to free some space.
			var hashPrevBranch node
			switch st.stack[level].ext.Val.(type) {
			case *fullNode:
				h, _ := st.hasher.hash(st.stack[level].ext.Val, false)
				hashPrevBranch = h.(hashNode)
				st.top = level
			case hashNode, valueNode:
				hashPrevBranch = st.stack[level].ext.Val
			default:
				panic("Encountered unexpected node type")
			}

			// Store the completed subtree in a fullNode at the slot
			// where both keys differ.
			slot := st.stack[level].ext.Key[whereitdiffers]

			// Allocate the next full node, it's going to be
			// reused several times.
			st.alloc()

			// Special case: the keys differ at the last element
			if len(st.stack[level].ext.Key) == whereitdiffers+1 {
				// Directly use the hashed value
				for i := range st.stack[level].branch.Children {
					st.stack[level].branch.Children[i] = nil
				}
				st.stack[level].branch.Children[slot] = hashPrevBranch
			} else {
				// Store the partially-hashed old node in the newly allocated
				// slot, in order to finish the hashing.
				st.stack[st.top].ext.Key = st.stack[level].ext.Key[whereitdiffers+1:]
				st.stack[st.top].ext.Val = hashPrevBranch
				st.stack[st.top].ext.flags = nodeFlag{dirty: true}

				// Directly hash the branch if the extension is empty
				var h node
				if len(st.stack[st.top].ext.Key) == 0 {
					h, _ = st.hasher.hash(&st.stack[st.top].branch, false)
				} else {
					h, _ = st.hasher.hash(&st.stack[st.top].ext, false)
				}
				st.stack[level].branch.Children[slot] = h
			}
			st.stack[level].ext.Val = &st.stack[level].branch
			st.stack[level].ext.Key = st.stack[level].ext.Key[:whereitdiffers]

			// Now use the newly allocated+hashed stack st.stack[level] to store
			// the rest of the inserted (key, value) pair.
			slot = key[whereitdiffers+len(st.stack[level].keyUntilHere)]
			st.stack[st.top].ext.Key = key[whereitdiffers+len(st.stack[level].keyUntilHere)+1:]
			if len(st.stack[st.top].ext.Key) == 0 {
				st.stack[level].branch.Children[slot] = hv
			} else {
				st.stack[level].branch.Children[slot] = &st.stack[st.top].ext
				st.stack[st.top].ext.Val = hv
			}
			st.stack[st.top].keyUntilHere = key[:whereitdiffers+len(st.stack[level].keyUntilHere)+1]
			st.stack[st.top].depth = st.stack[level].depth + 1
		}
	}

	// if ext.length == 0, directly return the full node.
	if len(st.stack[0].ext.Key) == 0 {
		return &st.stack[0].branch
	}
	return &st.stack[0].ext
}

// Hash hashes the stack trie by hashing the first entry in the stack
func (st *StackTrie) Hash() common.Hash {
	if st.top == -1 {
		return emptyRoot
	}

	h, _ := st.hasher.hash(&st.stack[0].ext, false)
	return common.BytesToHash(h.(hashNode))
}
