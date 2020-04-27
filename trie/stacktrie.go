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
	"io"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
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

// ReStackTrie is a reimplementation of the Stacktrie, that fixes
// bugs in the previous implementation, and which also implements
// its own hashing mechanism which is more specific and hopefully
// more efficient that the default hasher.
type ReStackTrie struct {
	nodeType  uint8            // node type (as in branch, ext, leaf)
	val       []byte           // value contained by this node if it's a leaf
	key       []byte           // key chunk covered by this (full|ext) node
	keyOffset int              // offset of the key chunk inside a full key
	children  [16]*ReStackTrie // list of children (for fullnodes and exts)
}

// NewReStackTrie allocates and initializes an empty trie.
func NewReStackTrie() *ReStackTrie {
	return &ReStackTrie{
		nodeType: 3,
	}
}

// List all values that ReStackTrie#nodeType can hold
const (
	branchNode = iota
	extNode
	leafNode
	emptyNode
	hashedNode
)

func (st *ReStackTrie) TryUpdate(key, value []byte) error {
	k := keybytesToHex(key)
	if len(value) == 0 {
		panic("deletion not supported")
	}
	st.insert(k[:len(k)-1], value)
	return nil
}

// Helper function that, given a full key, determines the index
// at which the chunk pointed by st.keyOffset is different from
// the same chunk in the full key.
func (st *ReStackTrie) getDiffIndex(key []byte) int {
	diffindex := 0
	for ; diffindex < len(st.key) && st.key[diffindex] == key[st.keyOffset+diffindex]; diffindex++ {
	}
	return diffindex
}

// Helper function to that inserts a (key, value) pair into
// the trie.
func (st *ReStackTrie) insert(key, value []byte) {
	switch st.nodeType {
	case branchNode: /* Branch */
		idx := key[st.keyOffset]
		if st.children[idx] == nil {
			st.children[idx] = NewReStackTrie()
			st.children[idx].keyOffset = st.keyOffset + 1
		}
		for i := idx - 1; i >= 0; i-- {
			if st.children[i] != nil {
				if st.children[i].nodeType != hashedNode {
					st.children[i].val = st.children[i].hash()
					st.children[i].key = nil
					st.children[i].nodeType = hashedNode
				}

				break
			}

		}
		st.children[idx].insert(key, value)
	case extNode: /* Ext */
		// Compare both key chunks and see where they differ
		diffidx := st.getDiffIndex(key)

		// Check if chunks are identical. If so, recurse into
		// the child node. Otherwise, the key has to be split
		// into 1) an optional common prefix, 2) the fullnode
		// representing the two differing path, and 3) a leaf
		// for each of the differentiated subtrees.
		if diffidx == len(st.key) {
			// Ext key and key segment are identical, recurse into
			// the child node.
			st.children[0].insert(key, value)
			return
		}
		// Save the original part. Depending if the break is
		// at the extension's last byte or not, create an
		// intermediate extension or use the extension's child
		// node directly.
		var n *ReStackTrie
		if diffidx < len(st.key)-1 {
			n = NewReStackTrie()
			n.key = st.key[diffidx+1:]
			n.children[0] = st.children[0]
			n.nodeType = extNode
		} else {
			// Break on the last byte, no need to insert
			// an extension node: reuse the current node
			n = st.children[0]
		}
		n.keyOffset = st.keyOffset + diffidx + 1

		var p *ReStackTrie
		if diffidx == 0 {
			// the break is on the first byte, so
			// the current node is converted into
			// a branch node.
			st.children[0] = nil
			p = st
			st.nodeType = branchNode
		} else {
			// the common prefix is at least one byte
			// long, insert a new intermediate branch
			// node.
			st.children[0] = NewReStackTrie()
			st.children[0].nodeType = branchNode
			st.children[0].keyOffset = st.keyOffset + diffidx
			p = st.children[0]
		}

		n.val = n.hash()
		n.nodeType = hashedNode
		n.key = nil

		// Create a leaf for the inserted part
		o := NewReStackTrie()
		o.keyOffset = st.keyOffset + diffidx + 1
		o.key = key[o.keyOffset:]
		o.val = value
		o.nodeType = leafNode

		// Insert both child leaves where they belong:
		origIdx := st.key[diffidx]
		newIdx := key[diffidx+st.keyOffset]
		p.children[origIdx] = n
		p.children[newIdx] = o
		st.key = st.key[:diffidx]

	case leafNode: /* Leaf */
		// Compare both key chunks and see where they differ
		diffidx := st.getDiffIndex(key)

		// Overwriting a key isn't supported, which means that
		// the current leaf is expected to be split into 1) an
		// optional extension for the common prefix of these 2
		// keys, 2) a fullnode selecting the path on which the
		// keys differ, and 3) one leaf for the differentiated
		// component of each key.
		if diffidx >= len(st.key) {
			panic("Trying to insert into existing key")
		}

		// Check if the split occurs at the first nibble of the
		// chunk. In that case, no prefix extnode is necessary.
		// Otherwise, create that
		var p *ReStackTrie
		if diffidx == 0 {
			// Convert current leaf into a branch
			st.nodeType = branchNode
			p = st
			st.children[0] = nil
		} else {
			// Convert current node into an ext,
			// and insert a child branch node.
			st.nodeType = extNode
			st.children[0] = NewReStackTrie()
			st.children[0].nodeType = branchNode
			st.children[0].keyOffset = st.keyOffset + diffidx
			p = st.children[0]
		}

		// Create the two child leaves: the one containing the
		// original value and the one containing the new value
		// The child leave will be hashed directly in order to
		// free up some memory.
		origIdx := st.key[diffidx]
		p.children[origIdx] = NewReStackTrie()
		p.children[origIdx].nodeType = leafNode
		p.children[origIdx].key = st.key[diffidx+1:]
		p.children[origIdx].val = st.val
		p.children[origIdx].keyOffset = p.keyOffset + 1

		p.children[origIdx].val = p.children[origIdx].hash()
		p.children[origIdx].nodeType = hashedNode
		p.children[origIdx].key = nil

		newIdx := key[diffidx+st.keyOffset]
		p.children[newIdx] = NewReStackTrie()
		p.children[newIdx].nodeType = leafNode
		p.children[newIdx].key = key[p.keyOffset+1:]
		p.children[newIdx].val = value
		p.children[newIdx].keyOffset = p.keyOffset + 1

		st.key = st.key[:diffidx]
	case emptyNode: /* Empty */
		st.nodeType = leafNode
		st.key = key[st.keyOffset:]
		st.val = value
	case hashedNode:
		panic("trying to insert into hash")
	default:
		panic("invalid type")
	}
}

// writeEvenHP writes a key with its hex prefix into a writer (presumably, the
// input of a hasher) and then writes the value. The value can be a maximum of
// 256 bytes, as it is only concerned with writing account leaves and optimize
// for this use case.
func writeHPRLP(writer io.Writer, key, val []byte, leaf bool) {
	// DEBUG don't remove yet
	//var writer bytes.Buffer

	// Determine the _t_ part of the hex prefix
	hp := byte(0)
	if leaf {
		hp = 32
	}

	const maxHeaderSize = 1 /* key byte list header */ +
		1 /* list header for key + value */ +
		1 /* potential size byte if total size > 56 */ +
		1 /* hex prefix if key is even-length*/
	header := [maxHeaderSize]byte{}
	keyOffset := 0
	headerPos := maxHeaderSize - 1

	// Add the hex prefix to its own byte if the key length is even, and
	// as the most significant nibble of the key if it's odd.
	// In the latter case, the first nibble of the key will be part of
	// the header and it will be skipped later when it's added to the
	// hasher sponge.
	if len(key)%2 == 0 {
		header[headerPos] = hp
	} else {
		header[headerPos] = hp | key[0] | 16
		keyOffset = 1
	}
	headerPos--

	// Add the key byte header, the key is 32 bytes max so it's always
	// under 56 bytes - no extra byte needed.
	keyByteSize := byte(len(key) / 2)
	if len(key) > 1 || header[len(header)-1] > 128 {
		header[headerPos] = 0x80 + keyByteSize + 1 /* HP */
		headerPos--
	}

	// If this is a leaf being inserted, the header length for the
	// value part will be two bytes as the leaf is more than 56 bytes
	// long.
	valHeaderLen := 1
	if len(val) > 56 {
		valHeaderLen = 2
	}

	// Add the global header, with optional length, and specify at
	// which byte the header is starting.
	payloadSize := int(keyByteSize) + (len(header) - headerPos - 1) +
		valHeaderLen + len(val) /* value + rlp header */
	var start int
	if payloadSize > 56 {
		header[headerPos] = byte(payloadSize)
		headerPos--
		header[headerPos] = 0xf8
		start = headerPos
	} else {
		header[headerPos] = 0xc0 + byte(payloadSize)
		start = headerPos
	}

	// Write the header into the sponge
	writer.Write(header[start:])

	// Write the key into the sponge
	var m byte
	for i, nibble := range key {
		// Skip the first byte if the key has an odd-length, since
		// it has already been written with the header.
		if i >= keyOffset {
			if (i-keyOffset)%2 == 0 {
				m = nibble
			} else {
				writer.Write([]byte{m*16 + nibble})
			}
		}
	}

	if leaf {
		writer.Write([]byte{0xb8, byte(len(val))})
	} else {
		writer.Write([]byte{0x80 + byte(len(val))})
	}
	writer.Write(val)

	// DEBUG don't remove yet
	//if leaf {
	//fmt.Println("leaf rlp ", writer)
	//} else {
	//fmt.Println("ext rlp ", writer)
	//}
	//io.Copy(w, &writer)
}

func (st *ReStackTrie) hash() []byte {
	/* Shortcut if node is already hashed */
	if st.nodeType == hashedNode {
		return st.val
	}

	d := sha3.NewLegacyKeccak256()
	switch st.nodeType {
	case branchNode:
		payload := [544]byte{}
		pos := 3 // maximum header length given what we know
		for i, v := range st.children {
			if v != nil {
				// Write a 32 byte list to the sponge
				payload[pos] = 0xa0
				pos++
				copy(payload[pos:pos+32], v.hash())
				pos += 32
				st.children[i] = nil // Reclaim mem from subtree
			} else {
				// Write an empty list to the sponge
				payload[pos] = 0x80
				pos++
			}
		}
		// Add empty 17th value
		payload[pos] = 0x80
		pos++

		// Compute the header, length size is either 0, 1 or 2 bytes since
		// there are at least 17 empty list headers, and at most 16 hashes
		// plus an empty header for the value.
		var start int
		if pos-3 < 56 {
			payload[2] = 0xc0 + byte(pos-3)
			start = 2
		} else if pos-3 < 256 {
			payload[2] = byte(pos - 3)
			payload[1] = 0xf8
			start = 1
		} else {
			payload[2] = byte(pos - 3)
			payload[1] = byte((pos - 3) >> 8)
			payload[0] = 0xf9
			start = 0
		}
		d.Write(payload[start:pos])
	case extNode:
		ch := st.children[0].hash()
		writeHPRLP(d, st.key, ch, false)
		st.children[0] = nil // Reclaim mem from subtree
	case leafNode:
		writeHPRLP(d, st.key, st.val, true)
	case emptyNode:
	default:
		panic("Invalid node type")
	}
	return d.Sum(nil)
}

func (st *ReStackTrie) Hash() (h common.Hash) {
	return common.BytesToHash(st.hash())
}
